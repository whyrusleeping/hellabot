// This is an example program showing the usage of hellabot
package main

import (
	"flag"
	"fmt"
	"time"

	hbot "github.com/whyrusleeping/hellabot"
	log "gopkg.in/inconshreveable/log15.v2"
)

var serv = flag.String("server", "testnet.oragono.io:6697", "hostname and port for irc server to connect to")
var nick = flag.String("nick", "hellabot", "nickname for the bot")
var password = flag.String("password", "hellabotspassword", "password for the bot")

func main() {
	flag.Parse()

	channels := func(bot *hbot.Bot) {
		bot.Channels = []string{"#test"}
	}
	saslOption := func(bot *hbot.Bot) {
		bot.SSL = true
		bot.SASL = true
		bot.Password = *password
	}

	irc, err := hbot.NewBot(*serv, *nick, saslOption, channels)
	if err != nil {
		panic(err)
	}

	irc.AddTrigger(sayInfoMessage)
	irc.AddTrigger(longTrigger)
	irc.Logger.SetHandler(log.StdoutHandler)
	// logHandler := log.LvlFilterHandler(log.LvlInfo, log.StdoutHandler)
	// or
	// irc.Logger.SetHandler(logHandler)
	// or
	// irc.Logger.SetHandler(log.StreamHandler(os.Stdout, log.JsonFormat()))

	// Start up bot (this blocks until we disconnect)
	irc.Run()
	fmt.Println("Bot shutting down.")
}

// This trigger replies Hello when you say hello
var sayInfoMessage = hbot.Trigger{
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "PRIVMSG" && m.Content == "-info"
	},
	Action: func(irc *hbot.Bot, m *hbot.Message) bool {
		irc.Reply(m, "Hello")
		return false
	},
}

// This trigger replies Hello when you say hello
var longTrigger = hbot.Trigger{
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "PRIVMSG" && m.Content == "-long"
	},
	Action: func(irc *hbot.Bot, m *hbot.Message) bool {
		irc.Reply(m, "This is the first message")
		time.Sleep(5 * time.Second)
		irc.Reply(m, "This is the second message")

		return false
	},
}
