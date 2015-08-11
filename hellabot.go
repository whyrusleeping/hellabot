package hbot

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/sorcix/irc"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"

	"bytes"
	"crypto/tls"
	"encoding/base64"
)

type Bot struct {
	// Channel for user to read incoming messages
	Incoming chan *Message

	// Channels to join after connection
	JoinAfterConnection []string

	//Server password (optional) only used if set
	Password string

	// SSL
	UseSSL bool

	// Do SASL authentication
	DoSASL bool

	con      net.Conn
	outgoing chan string
	triggers []*Trigger

	// This bots nick
	nick string

	// Unix domain socket address for reconnects (linux only)
	unixastr string
	unixlist net.Listener

	// Whether or not this is a reconnect instance
	reconnect bool

	// Duration to wait between sending of messages to avoid being
	// kicked by the server for flooding (default 200ms)
	ThrottleDelay time.Duration
	// Log15 loggger
	log.Logger
}

type Config struct {
	Server, Nick   string
	Channels       []string
	SSL, reconnect bool
}

// load a Json config file
func LoadConfig(f string) (Config, error) {

	file, err := os.Open(f)

	if err != nil {
		fmt.Println("Couldn't read config file")
		return Config{}, err
	}
	defer file.Close()

	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		fmt.Println("Couldn't parse json file")
		return Config{}, err
	}
	return config, err

}

// Connect to an irc server, reading configuration from json file
func NewBotFromJSON(config Config) (*Bot, Config, error) {

	fmt.Println("Nickname: " + config.Nick)
	fmt.Println("Server: " + config.Server)
	nick := flag.String("nick", config.Nick, "nickname for the bot")
	serv := flag.String("server", config.Server, "hostname and port for irc server to connect to")
	flag.Parse()
	irc, err := NewBot(*serv, *nick, config.SSL, config.reconnect)
	if config.Channels != nil {
		fmt.Println("Channels to join on connect")
		for _, s := range config.Channels {
			fmt.Println("Channel: " + s)
			irc.JoinAfterConnection = append(irc.JoinAfterConnection, s)
		}
	}

	return irc, config, err
}

// Connect to an irc server
func NewBot(host, nick string, ssl, recon bool) (*Bot, error) {
	bot := new(Bot)

	bot.Incoming = make(chan *Message, 16)
	bot.outgoing = make(chan string, 16)
	bot.nick = nick
	bot.unixastr = fmt.Sprintf("@%s-%s/bot", host, nick)
	bot.UseSSL = ssl
	bot.ThrottleDelay = time.Millisecond * 200
	bot.Logger = log.New("id", logext.RandId(8), "host", host, "nick", log.Lazy{bot.getNick})
	bot.Logger.SetHandler(log.DiscardHandler())

	// Attempt reconnection
	var hijack bool
	if recon {
		hijack = bot.HijackSession()
		bot.Debug("Hijack", hijack)
	}

	if !hijack {
		err := bot.Connect(host)
		if err != nil {
			return nil, err
		}
		bot.Info("Connected successfully!")
	}

	bot.AddTrigger(pingPong)
	return bot, nil
}
func (bot *Bot) getNick() string {
	return bot.nick
}
func (bot *Bot) Connect(host string) (err error) {
	bot.Debug("Connect")
	if bot.UseSSL {
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
		bot.con.SetDeadline(time.Now().Add(300 * time.Second))
		msg := ParseMessage(scan.Text())
		bot.Debug("Raw", "line", scan.Text(), "msg.To", msg.To, "msg.From", msg.From, "msg.Params", msg.Params, "msg.Trailing", msg.Trailing)
		consumed := false
		for _, t := range bot.triggers {
			if t.Condition(msg) {
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
	bot.SetNick(bot.nick)
	bot.sendUserCommand(bot.nick, bot.nick, "8")

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
	bot.sendUserCommand(bot.nick, bot.nick, "8")
	bot.SetNick(bot.nick)
}

// Set username, real name, and mode
func (bot *Bot) sendUserCommand(user, realname, mode string) {
	bot.Send(fmt.Sprintf("USER %s %s * :%s", user, mode, realname))
}

func (bot *Bot) SetNick(nick string) {
	bot.nick = nick
	bot.Send(fmt.Sprintf("NICK %s", nick))
}

// Start up servers various running methods
func (bot *Bot) Start() {
	bot.Debug("Start bot processes.")

	go bot.handleIncomingMessages()
	go bot.handleOutgoingMessages()

	go bot.StartUnixListener()

	// Only register on an initial connection
	if !bot.reconnect {
		if bot.DoSASL {
			bot.SASLAuthenticate(bot.nick, bot.Password)
		} else {
			bot.StandardRegistration()
		}
	}

	for _, s := range bot.JoinAfterConnection {
		bot.Join(s)
	}
}

// Send a message to 'who' (user or channel)
func (bot *Bot) Msg(who, text string) {
	// if len(text) == 0, return instead of trying to send a empty message
	if len(text) == 0 {
		return
	}
	for len(text) > 400 {
		bot.Send("PRIVMSG " + who + " :" + text[:400])
		text = text[400:]
	}
	bot.Send("PRIVMSG " + who + " :" + text)
}

// Notice sends a NOTICE message to 'who' (user or channel)
func (bot *Bot) Notice(who, text string) {
	// if len(text) == 0, return instead of trying to send a empty notice
	if len(text) == 0 {
		return
	}
	for len(text) > 400 {
		bot.Send("NOTICE " + who + " :" + text[:400])
		text = text[400:]
	}
	bot.Send("NOTICE " + who + " :" + text)
}

// Send a action to 'who' (user or channel)
func (bot *Bot) Action(who, text string) {
	// if len(text) == 0, return instead of trying to send a empty action
	if len(text) == 0 {
		return
	}
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

func (bot *Bot) AddTrigger(t *Trigger) {
	bot.triggers = append(bot.triggers, t)
}

// A trigger is used to subscribe and react to events on the bot Server
type Trigger struct {
	// Returns true if this trigger applies to the passed in message
	Condition func(*Message) bool

	// The action to perform if Condition is true
	// return true if the message was 'consumed'
	Action func(*Bot, *Message) bool
}

// A trigger to respond to the servers ping pong messages
// If PingPong messages are not responded to, the server assumes the
// client has timed out and will close the connection.
// Note: this is automatically added in the IrcCon constructor
var pingPong = &Trigger{
	func(m *Message) bool {
		return m.Command == "PING"
	},
	func(bot *Bot, m *Message) bool {
		bot.Send("PONG :" + m.Content)
		return true
	},
}

type Message struct {
	// irc.Message from sorcix
	irc.Message
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
	m.Message = *irc.ParseMessage(raw)
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
