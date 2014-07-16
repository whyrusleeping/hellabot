package hbot

import (
	"fmt"
	"net"
	"time"
	"bufio"

	"crypto/tls"
	"crypto/md5"
	"encoding/base64"
)

// Verbosity of server logging
var Verbosity int

type IrcCon struct {
	// Channel for user to read incoming messages
	Incoming chan *Message

	// Map of irc channels this bot is joined to
	Channels map[string]*IrcChannel

	//Server password (optional) only used if set
	Password string

	// SSL
	UseSSL bool

	// Do SASL authentication
	DoSasl bool

	con net.Conn
	outgoing chan string
	tr []*Trigger

	// This bots nick
	nick string

	// Unix domain socket address for reconnects (linux only)
	unixastr string

	// Whether or not this is a reconnect instance
	reconnect bool
}

// Connect to an irc server
func NewIrcConnection(host, nick string, ssl bool) (*IrcCon, error) {
	irc := new(IrcCon)

	irc.Incoming = make(chan *Message, 16)
	irc.outgoing = make(chan string, 16)
	irc.Channels = make(map[string]*IrcChannel)
	irc.nick = nick
	irc.unixastr = fmt.Sprintf("@%s/irc", nick)
	irc.UseSSL = ssl

	// Attempt reconnection
	if !irc.HijackSession() {
		err := irc.Connect(host)
		if err != nil {
			return nil,err
		}
	}

	irc.AddTrigger(pingPong)
	return irc, nil
}

func (irc *IrcCon) Connect(host string) (err error) {
	irc.Log(3, "Connect")
	if irc.UseSSL {
		irc.con,err = tls.Dial("tcp", host, &tls.Config{})
	} else {
		irc.con,err = net.Dial("tcp", host)
	}
	return
}

// Incoming message gathering routine
func (irc *IrcCon) handleIncomingMessages() {
	scan := bufio.NewScanner(irc.con)
	for scan.Scan() {
		mes := ParseMessage(scan.Text())
		consumed := false
		if c,ok := irc.Channels[mes.To]; ok {
			c.istream <- mes
		}
		for _,t := range irc.tr {
			if t.Condition(mes) {
				consumed = t.Action(irc,mes)
			}
			if consumed {
				break
			}
		}
		if !consumed {
			irc.Incoming <- mes
		}
	}
}

// Handles message speed throtling
func (irc *IrcCon) handleOutgoingMessages() {
	for s := range irc.outgoing {
		irc.Log(4, "Sending: '%s'", s)
		_,err := fmt.Fprint(irc.con, s + "\r\n")
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Millisecond * 200)
	}
}

func (irc *IrcCon) Log(level int, format string, args ...interface{}) {
	if Verbosity >= level {
		fmt.Printf(format + "\n", args...)
	}
}
// Perform SASL authentication
// ref: https://github.com/atheme/charybdis/blob/master/doc/sasl.txt
func (irc *IrcCon) Authenticate(user, pass string) {
	irc.Log(3, "Beginning SASL Authentication")
	irc.Send("CAP REQ :sasl")
	irc.sendUserCommand(irc.nick, irc.nick, "8")
	irc.SetNick(irc.nick)
	irc.Send("AUTHENTICATE PLAIN")

	// TODO: Verify that this is the proper way
	hash := md5.Sum([]byte(pass))
	passtext := base64.StdEncoding.EncodeToString(hash[:])
	irc.Send("AUTHENTICATE " + passtext)
	irc.Send("CAP END")
}

// A basic set of registration commands
func (irc *IrcCon) StandardRegistration() {
	//Server registration
	if irc.Password != "" {
		irc.Send("PASS " + irc.Password)
	}
	irc.sendUserCommand(irc.nick, irc.nick, "8")
	irc.SetNick(irc.nick)
}

// Set username, real name, and mode
func (irc *IrcCon) sendUserCommand(user, realname, mode string) {
	irc.Send(fmt.Sprintf("USER %s %s * :%s", user, mode, realname))
}

func (irc *IrcCon) SetNick(nick string) {
	irc.Send(fmt.Sprintf("NICK %s", nick))
}

// Start up servers various running methods
func (irc *IrcCon) Start() {
	irc.Log(3, "Start bot processes.")
	go irc.handleIncomingMessages()
	go irc.handleOutgoingMessages()

	go irc.StartUnixListener()

	// Only register on an initial connection
	if !irc.reconnect {
		if irc.DoSasl {
			irc.Authenticate(irc.nick, irc.Password)
		} else {
			irc.StandardRegistration()
		}
	}
}

// Send a message to 'who' (user or channel)
func (irc *IrcCon) Msg(who, text string) {
	irc.Send("PRIVMSG " + who + " :" + text)
}

// Send any command to the server
func (irc *IrcCon) Send(command string) {
	irc.outgoing <- command
}

// Used to change users modes in a channel
// operator = "+o" deop = "-o"
// ban = "+b"
func (irc *IrcCon) ChMode(user, channel, mode string) {
	irc.Send("MODE " + channel + " " + mode + " " + user)
}

// Join a channel and register its struct in the IrcCons channel map
func (irc *IrcCon) Join(ch string) *IrcChannel {
	irc.Send("JOIN " + ch)
	ichan := &IrcChannel{
		Name: ch,
		con: irc,
		Counts: make(map[string]int),
		istream: make(chan *Message),
	}
	go ichan.handleMessages()

	irc.Channels[ch] = ichan
	ichan.TryLoadStats(ch[1:] + ".stats")
	return ichan
}

func (irc *IrcCon) AddTrigger(t *Trigger) {
	irc.tr = append(irc.tr, t)
}

// A trigger is used to subscribe and react to events on the Irc Server
type Trigger struct {
	// Returns true if this trigger applies to the passed in message
	Condition func (*Message) bool

	// The action to perform if Condition is true
	// return true if the message was 'consumed'
	Action func (*IrcCon,*Message) bool
}

// A trigger to respond to the servers ping pong messages
// If PingPong messages are not responded to, the server assumes the
// client has timed out and will close the connection.
// Note: this is automatically added in the IrcCon constructor
var pingPong = &Trigger{
	func (m *Message) bool {
		return m.Command == "PING"
	},
	func (irc *IrcCon, m *Message) bool {
		irc.Send("PONG :" + m.Content)
		return true
	},
}
