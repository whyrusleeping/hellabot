package hbot

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"gopkg.in/sorcix/irc.v1"

	"bytes"
	"crypto/tls"
	"encoding/base64"
)

// Bot implements an irc bot to be connected to a given server
type Bot struct {

	// This is set if we have hijacked a connection
	reconnecting bool
	// Channel for user to read incoming messages
	Incoming chan *Message
	con      net.Conn
	outgoing chan string
	handlers []Handler
	// When did we start? Used for uptime
	started time.Time
	// Unix domain abstract socket address for reconnects (linux only)
	unixastr string
	// Unix domain socket address for other Unixes
	unixsock string
	unixlist net.Listener
	// Log15 loggger
	log.Logger
	didJoinChannels sync.Once
	didGetPrefix    sync.Once

	// Exported fields
	Host          string
	Password      string
	Channels      []string
	SSL           bool
	SASL          bool
	HijackSession bool
	// Bot's prefix
	Prefix *irc.Prefix
	// An optional function that connects to an IRC server over plaintext:
	Dial func(network, addr string) (net.Conn, error)
	// An optional function that connects to an IRC server over a secured connection:
	DialTLS func(network, addr string, tlsConf *tls.Config) (*tls.Conn, error)
	// This bots nick
	Nick string
	// Duration to wait between sending of messages to avoid being
	// kicked by the server for flooding (default 200ms)
	ThrottleDelay time.Duration
	// Maxmimum time between incoming data
	PingTimeout time.Duration

	TLSConfig tls.Config
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
		unixsock:      fmt.Sprintf("/tmp/%s-%s-bot.sock", host, nick),
		Host:          host,
		Nick:          nick,
		ThrottleDelay: 200 * time.Millisecond,
		PingTimeout:   300 * time.Second,
		HijackSession: false,
		SSL:           false,
		SASL:          false,
		Channels:      []string{"#test"},
		Password:      "",
		// Somewhat sane default if for some reason we can't retrieve the bot's prefix
		// for example, if the server doesn't advertise joins
		Prefix: &irc.Prefix{
			Name: nick,
			User: nick,
			Host: strings.Repeat("*", 510-400-len(nick)*2),
		},
	}
	for _, option := range options {
		option(&bot)
	}
	// Discard logs by default
	bot.Logger = log.New("id", logext.RandId(8), "host", bot.Host, "nick", log.Lazy{bot.getNick})

	bot.Logger.SetHandler(log.DiscardHandler())
	bot.AddTrigger(pingPong)
	bot.AddTrigger(joinChannels)
	bot.AddTrigger(getPrefix)
	return &bot, nil
}

// Uptime returns the uptime of the bot
func (bot *Bot) Uptime() string {
	return fmt.Sprintf("Started: %s, Uptime: %s", bot.started, time.Since(bot.started))
}

func (bot *Bot) getNick() string {
	return bot.Nick
}

func (bot *Bot) connect(host string) (err error) {
	bot.Debug("Connecting")
	dial := bot.Dial
	if dial == nil {
		dial = net.Dial
	}
	dialTLS := bot.DialTLS
	if dialTLS == nil {
		dialTLS = tls.Dial
	}

	if bot.SSL {
		bot.con, err = dialTLS("tcp", host, &bot.TLSConfig)
	} else {
		bot.con, err = dial("tcp", host)
	}
	return err
}

// Incoming message gathering routine
func (bot *Bot) handleIncomingMessages() {
	scan := bufio.NewScanner(bot.con)
	for scan.Scan() {
		// Disconnect if we have seen absolutely nothing for 300 seconds
		bot.con.SetDeadline(time.Now().Add(bot.PingTimeout))
		msg := ParseMessage(scan.Text())
		bot.Debug("Incoming", "raw", scan.Text(), "msg.To", msg.To, "msg.From", msg.From, "msg.Params", msg.Params, "msg.Trailing", msg.Trailing)
		go func() {
			for _, h := range bot.handlers {
				if h.Handle(bot, msg) {
					break
				}
			}
		}()
		bot.Incoming <- msg
	}
	close(bot.Incoming)
}

// Handles message speed throtling
func (bot *Bot) handleOutgoingMessages() {
	for s := range bot.outgoing {
		bot.Debug("Outgoing", "data", s)
		_, err := fmt.Fprint(bot.con, s+"\r\n")
		if err != nil {
			bot.Error("handleOutgoingMessages fmt.Fprint error", "err", err)
			return
		}
		time.Sleep(bot.ThrottleDelay)
	}
}

// SASLAuthenticate performs SASL authentication
// ref: https://github.com/atheme/charybdis/blob/master/doc/sasl.txt
func (bot *Bot) SASLAuthenticate(user, pass string) {
	var saslHandler = Trigger{
		Condition: func(bot *Bot, m *Message) bool {
			return (strings.TrimSpace(m.Content) == "sasl" && len(m.Params) > 1 && m.Params[1] == "ACK") ||
				(m.Command == "AUTHENTICATE" && len(m.Params) == 1 && m.Params[0] == "+")
		},
		Action: func(bot *Bot, m *Message) bool {
			if strings.TrimSpace(m.Content) == "sasl" && len(m.Params) > 1 && m.Params[1] == "ACK" {
				bot.Debug("Recieved SASL ACK")
				bot.Send("AUTHENTICATE PLAIN")
			}

			if m.Command == "AUTHENTICATE" && len(m.Params) == 1 && m.Params[0] == "+" {
				bot.Debug("Got auth message!")
				out := bytes.Join([][]byte{[]byte(user), []byte(user), []byte(pass)}, []byte{0})
				encpass := base64.StdEncoding.EncodeToString(out)
				bot.Send("AUTHENTICATE " + encpass)
				bot.Send("AUTHENTICATE +")
				bot.Send("CAP END")
			}
			return false
		},
	}

	bot.AddTrigger(saslHandler)
	bot.Debug("Beginning SASL Authentication")
	bot.Send("CAP REQ :sasl")
	bot.SetNick(bot.Nick)
	bot.sendUserCommand(bot.Nick, bot.Nick, "8")
}

// WaitFor will block until a message matching the given filter is received
func (bot *Bot) WaitFor(filter func(*Message) bool) {
	for mes := range bot.Incoming {
		if filter(mes) {
			return
		}
	}
	return
}

// StandardRegistration performsa a basic set of registration commands
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

// SetNick sets the bots nick on the irc server
func (bot *Bot) SetNick(nick string) {
	bot.Nick = nick
	bot.Prefix.Name = nick
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
			return
		}
		hijack = bot.hijackSession()
		bot.Debug("Hijack", "Did we?", hijack)
	}

	if !hijack {
		err := bot.connect(bot.Host)
		if err != nil {
			bot.Crit("bot.Connect error", "err", err.Error())
			return
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

// Reply sends a message to where the message came from (user or channel)
func (bot *Bot) Reply(m *Message, text string) {
	var target string
	if strings.Contains(m.To, "#") {
		target = m.To
	} else {
		target = m.From
	}
	bot.Msg(target, text)
}

// Msg sends a message to 'who' (user or channel)
func (bot *Bot) Msg(who, text string) {
	const command = "PRIVMSG"
	for _, line := range splitText(text, command, who, bot.Prefix) {
		bot.Send(command + " " + who + " :" + line)
	}
}

// Notice sends a NOTICE message to 'who' (user or channel)
func (bot *Bot) Notice(who, text string) {
	const command = "NOTICE"
	for _, line := range splitText(text, command, who, bot.Prefix) {
		bot.Send(command + " " + who + " :" + line)
	}
}

// Splits a given string into a string slice, in chunks ending
// either with \n, or with \r\n, or splitting text to maximally allowed size.
func splitText(text, command, who string, prefix *irc.Prefix) []string {
	var ret []string

	// Maximum message size that fits into 512 bytes.
	// Carriage return and linefeed are not counted here as they
	// are added by handleOutgoingMessages()
	maxSize := 510 - len(fmt.Sprintf(":%s %s %s :", prefix, command, who))

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		for len(line) > maxSize {
			totalSize := 0
			runeSize := 0
			// utf-8 aware splitting
			for _, v := range line {
				runeSize = utf8.RuneLen(v)
				if totalSize+runeSize > maxSize {
					ret = append(ret, line[:totalSize])
					line = line[totalSize:]
					totalSize = runeSize
					continue
				}
				totalSize += runeSize
			}

		}
		ret = append(ret, line)
	}
	return ret
}

// Action sends an action to 'who' (user or channel)
func (bot *Bot) Action(who, text string) {
	msg := fmt.Sprintf("\u0001ACTION %s\u0001", text)
	bot.Msg(who, msg)
}

// Topic sets the channel 'c' topic (requires bot has proper permissions)
func (bot *Bot) Topic(c, topic string) {
	str := fmt.Sprintf("TOPIC %s :%s", c, topic)
	bot.Send(str)
}

// Send any command to the server
func (bot *Bot) Send(command string) {
	bot.outgoing <- command
}

// ChMode is used to change users modes in a channel
// operator = "+o" deop = "-o"
// ban = "+b"
func (bot *Bot) ChMode(user, channel, mode string) {
	bot.Send("MODE " + channel + " " + mode + " " + user)
}

// Join a channel
func (bot *Bot) Join(ch string) {
	bot.Send("JOIN " + ch)
}

// Part a channel
func (bot *Bot) Part(ch, msg string) {
	bot.Send("PART " + ch + " " + msg)
}

// Close closes the bot
func (bot *Bot) Close() error {
	if bot.unixlist != nil {
		return bot.unixlist.Close()
	}
	return nil
}

// AddTrigger adds a trigger to the bot's handlers
func (bot *Bot) AddTrigger(h Handler) {
	bot.handlers = append(bot.handlers, h)
}

// Handler is used to subscribe and react to events on the bot Server
type Handler interface {
	Handle(*Bot, *Message) bool
}

// Trigger is a Handler which is guarded by a condition
type Trigger struct {
	// Returns true if this trigger applies to the passed in message
	Condition func(*Bot, *Message) bool

	// The action to perform if Condition is true
	// return true if the message was 'consumed'
	Action func(*Bot, *Message) bool
}

// Handle executes the trigger action if the condition is satisfied
func (t Trigger) Handle(b *Bot, m *Message) bool {
	return t.Condition(b, m) && t.Action(b, m)
}

// A trigger to respond to the servers ping pong messages
// If PingPong messages are not responded to, the server assumes the
// client has timed out and will close the connection.
// Note: this is automatically added in the IrcCon constructor
var pingPong = Trigger{
	Condition: func(bot *Bot, m *Message) bool {
		return m.Command == "PING"
	},
	Action: func(bot *Bot, m *Message) bool {
		bot.Send("PONG :" + m.Content)
		return true
	},
}

var joinChannels = Trigger{
	Condition: func(bot *Bot, m *Message) bool {
		return m.Command == irc.RPL_WELCOME || m.Command == irc.RPL_ENDOFMOTD // 001 or 372
	},
	Action: func(bot *Bot, m *Message) bool {
		bot.didJoinChannels.Do(func() {
			for _, channel := range bot.Channels {
				splitchan := strings.SplitN(channel, ":", 2)
				fmt.Println("splitchan is:", splitchan)
				if len(splitchan) == 2 {
					channel = splitchan[0]
					password := splitchan[1]
					bot.Send(fmt.Sprintf("JOIN %s %s", channel, password))
				} else {
					bot.Send(fmt.Sprintf("JOIN %s", channel))
				}
			}
		})
		return true
	},
}

// Get the bot's prefix by catching its own join
var getPrefix = Trigger{
	Condition: func(bot *Bot, m *Message) bool {
		return (m.Command == "JOIN" && m.Name == bot.Nick)
	},
	Action: func(bot *Bot, m *Message) bool {
		bot.didGetPrefix.Do(func() {
			bot.Prefix = m.Prefix
			bot.Debug("Got prefix", "prefix", bot.Prefix.String())
		})
		return false
	},
}

func SaslAuth(pass string) func(*Bot) {
	return func(b *Bot) {
		b.SASL = true
		b.Password = pass
	}
}

func ReconOpt() func(*Bot) {
	return func(b *Bot) {
		b.HijackSession = true
	}
}

// Message represents a message received from the server
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
