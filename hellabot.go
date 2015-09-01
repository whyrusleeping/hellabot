package hbot

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sorcix/irc"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"

	"bytes"
	"crypto/tls"
	"encoding/base64"
)

type Bot struct {

	// This is set if we have hijacked a connection
	reconnecting bool
	// Channel for user to read incoming messages
	Incoming chan *Message
	con      net.Conn
	outgoing chan string
	triggers []Trigger
	// When did we start? Used for uptime
	started time.Time
	// Unix domain socket address for reconnects (linux only)
	unixastr string
	unixlist net.Listener
	// Log15 loggger
	log.Logger
	didJoinChannels sync.Once

	// Exported fields
	Host          string
	Password      string
	Channels      []string
	SSL           bool
	SASL          bool
	HijackSession bool
	// This bots nick
	Nick string
	// Duration to wait between sending of messages to avoid being
	// kicked by the server for flooding (default 200ms)
	ThrottleDelay time.Duration
	// Maxmimum time between incoming data
	PingTimeout time.Duration
}

func (bot *Bot) String() string {
	return fmt.Sprintf("Server: %s, Channels: %v, Nick: %s", bot.Host, bot.Channels, bot.Nick)
}

// NewBot creates a new instance of Bot
func NewBot(host, nick string, options ...func(*Bot)) (*Bot, error) {
	// Defaults are set here
	bot := Bot{
		Incoming:      make(chan *Message, 16),
		outgoing:      make(chan string, 16),
		started:       time.Now(),
		unixastr:      fmt.Sprintf("@%s-%s/bot", host, nick),
		Host:          host,
		Nick:          nick,
		ThrottleDelay: 200 * time.Millisecond,
		PingTimeout:   300 * time.Second,
		HijackSession: false,
		SSL:           false,
		SASL:          false,
		Channels:      []string{"#test"},
		Password:      "",
	}
	for _, option := range options {
		option(&bot)
	}
	// Discard logs by default
	bot.Logger = log.New("id", logext.RandId(8), "host", bot.Host, "nick", log.Lazy{bot.getNick})

	bot.Logger.SetHandler(log.DiscardHandler())
	bot.AddTrigger(pingPong)
	bot.AddTrigger(joinChannels)
	return &bot, nil
}

// Getter for uptime, so it's not possible to modify from the outside.
func (bot *Bot) Uptime() string {
	return fmt.Sprintf("Started: %s, Uptime: %s", bot.started, time.Since(bot.started))
}
func (bot *Bot) getNick() string {
	return bot.Nick
}
func (bot *Bot) connect(host string) (err error) {
	bot.Debug("Connecting")
	if bot.SSL {
		bot.con, err = tls.Dial("tcp", host, &tls.Config{})
	} else {
		bot.con, err = net.Dial("tcp", host)
	}
	return
}

// Incoming message gathering routine
func (bot *Bot) handleIncomingMessages() {
	scan := bufio.NewScanner(bot.con)
	for scan.Scan() {
		// Disconnect if we have seen absolutely nothing for 300 seconds
		bot.con.SetDeadline(time.Now().Add(bot.PingTimeout))
		msg := ParseMessage(scan.Text())
		bot.Debug("Incoming", "raw", scan.Text(), "msg.To", msg.To, "msg.From", msg.From, "msg.Params", msg.Params, "msg.Trailing", msg.Trailing)
		consumed := false
		for _, t := range bot.triggers {
			if t.Condition(bot, msg) {
				consumed = t.Action(bot, msg)
			}
			if consumed {
				break
			}
		}
		if !consumed {
			bot.Incoming <- msg
		}
	}
	close(bot.Incoming)
}

// Handles message speed throtling
func (bot *Bot) handleOutgoingMessages() {
	for s := range bot.outgoing {
		bot.Debug("Outgoing", "data", s)
		_, err := fmt.Fprint(bot.con, s+"\r\n")
		if err != nil {
			bot.Error("write error: ", err)
			return
		}
		time.Sleep(bot.ThrottleDelay)
	}
}

// Perform SASL authentication
// ref: https://github.com/atheme/charybdis/blob/master/doc/sasl.txt
func (bot *Bot) SASLAuthenticate(user, pass string) {
	bot.Debug("Beginning SASL Authentication")
	bot.Send("CAP REQ :sasl")
	bot.SetNick(bot.Nick)
	bot.sendUserCommand(bot.Nick, bot.Nick, "8")

	bot.WaitFor(func(mes *Message) bool {
		return mes.Content == "sasl" && len(mes.Params) > 1 && mes.Params[1] == "ACK"
	})
	bot.Debug("Recieved SASL ACK")
	bot.Send("AUTHENTICATE PLAIN")

	bot.WaitFor(func(mes *Message) bool {
		return mes.Command == "AUTHENTICATE" && len(mes.Params) == 1 && mes.Params[0] == "+"
	})

	bot.Debug("Got auth message!")

	out := bytes.Join([][]byte{[]byte(user), []byte(user), []byte(pass)}, []byte{0})
	encpass := base64.StdEncoding.EncodeToString(out)
	bot.Send("AUTHENTICATE " + encpass)
	bot.Send("AUTHENTICATE +")
	bot.Send("CAP END")
}

func (bot *Bot) WaitFor(filter func(*Message) bool) {
	for mes := range bot.Incoming {
		if filter(mes) {
			return
		}
	}
	return
}

// A basic set of registration commands
func (bot *Bot) StandardRegistration() {
	//Server registration
	if bot.Password != "" {
		bot.Send("PASS " + bot.Password)
	}
	bot.Debug("Sending standard registration")
	bot.sendUserCommand(bot.Nick, bot.Nick, "8")
	bot.SetNick(bot.Nick)
}

// Set username, real name, and mode
func (bot *Bot) sendUserCommand(user, realname, mode string) {
	bot.Send(fmt.Sprintf("USER %s %s * :%s", user, mode, realname))
}

func (bot *Bot) SetNick(nick string) {
	bot.Nick = nick
	bot.Send(fmt.Sprintf("NICK %s", nick))
}

// Run starts the bot and connects to the server. Blocks until we disconnect from the server.
func (bot *Bot) Run() {
	bot.Debug("Starting bot goroutines")

	// Attempt reconnection
	var hijack bool
	if bot.HijackSession {
		if bot.SSL {
			bot.Crit("Can't Hijack a SSL connection")
		}
		hijack = bot.hijackSession()
		bot.Debug("Hijack", "Did we?", hijack)
	}

	if !hijack {
		err := bot.connect(bot.Host)
		if err != nil {
			bot.Error(err.Error())
		}
		bot.Info("Connected successfully!")
	}

	go bot.handleIncomingMessages()
	go bot.handleOutgoingMessages()

	go bot.StartUnixListener()

	// Only register on an initial connection
	if !bot.reconnecting {
		if bot.SASL {
			bot.SASLAuthenticate(bot.Nick, bot.Password)
		} else {
			bot.StandardRegistration()
		}
	}
	for m := range bot.Incoming {
		if m == nil {
			log.Info("Disconnected")
			return
		}
	}
}

// Send a message to 'who' (user or channel)
func (bot *Bot) Msg(who, text string) {
	for len(text) > 400 {
		bot.Send("PRIVMSG " + who + " :" + text[:400])
		text = text[400:]
	}
	bot.Send("PRIVMSG " + who + " :" + text)
}

// Notice sends a NOTICE message to 'who' (user or channel)
func (bot *Bot) Notice(who, text string) {
	for len(text) > 400 {
		bot.Send("NOTICE " + who + " :" + text[:400])
		text = text[400:]
	}
	bot.Send("NOTICE " + who + " :" + text)
}

// Send a action to 'who' (user or channel)
func (bot *Bot) Action(who, text string) {
	msg := fmt.Sprintf("\u0001ACTION %s\u0001", text)
	bot.Msg(who, msg)
}

// Sets the channel 'c' topic (requires bot has proper permissions)
func (bot *Bot) Topic(c, topic string) {
	str := fmt.Sprintf("TOPIC %s :%s", c, topic)
	bot.Send(str)
}

// Send any command to the server
func (bot *Bot) Send(command string) {
	bot.outgoing <- command
}

// Used to change users modes in a channel
// operator = "+o" deop = "-o"
// ban = "+b"
func (bot *Bot) ChMode(user, channel, mode string) {
	bot.Send("MODE " + channel + " " + mode + " " + user)
}

// Join a channel
func (bot *Bot) Join(ch string) {
	bot.Send("JOIN " + ch)
}

func (bot *Bot) Close() error {
	if bot.unixlist != nil {
		return bot.unixlist.Close()
	}
	return nil
}

func (bot *Bot) AddTrigger(t Trigger) {
	bot.triggers = append(bot.triggers, t)
}

// A trigger is used to subscribe and react to events on the bot Server
type Trigger struct {
	// Returns true if this trigger applies to the passed in message
	Condition func(*Bot, *Message) bool

	// The action to perform if Condition is true
	// return true if the message was 'consumed'
	Action func(*Bot, *Message) bool
}

// A trigger to respond to the servers ping pong messages
// If PingPong messages are not responded to, the server assumes the
// client has timed out and will close the connection.
// Note: this is automatically added in the IrcCon constructor
var pingPong = Trigger{
	func(bot *Bot, m *Message) bool {
		return m.Command == "PING"
	},
	func(bot *Bot, m *Message) bool {
		bot.Send("PONG :" + m.Content)
		return true
	},
}
var joinChannels = Trigger{
	func(bot *Bot, m *Message) bool {
		return m.Command == irc.RPL_WELCOME || m.Command == irc.RPL_ENDOFMOTD // 001 or 372
	},
	func(bot *Bot, m *Message) bool {
		bot.didJoinChannels.Do(func() {
			for _, channel := range bot.Channels {
				bot.Send(fmt.Sprintf("JOIN %s", channel))
			}
		})
		return true
	},
}

type Message struct {
	// irc.Message from sorcix
	*irc.Message
	// Content generally refers to the text of a PRIVMSG
	Content string

	//Time at which this message was recieved
	TimeStamp time.Time

	// Entity that this message was addressed to (channel or user)
	To string

	// Nick of the messages sender (equivalent to Prefix.Name)
	// Outdated, please use .Name
	From string
}

// ParseMessage takes a string and attempts to create a Message struct.
// Returns nil if the Message is invalid.
// TODO: Maybe just use sorbix/irc if we can be without the custom stuff?
func ParseMessage(raw string) (m *Message) {
	m = new(Message)
	m.Message = irc.ParseMessage(raw)
	m.Content = m.Trailing

	if len(m.Params) > 0 {
		m.To = m.Params[0]
	} else if m.Command == "JOIN" {
		m.To = m.Trailing
	}
	if m.Prefix != nil {
		m.From = m.Prefix.Name
	}
	m.TimeStamp = time.Now()

	return m
}
