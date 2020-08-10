// This is an example program showing the usage of hellabot
package main

import (
	"flag"
	"fmt"
	"time"

	hbot "github.com/whyrusleeping/hellabot"
	log "gopkg.in/inconshreveable/log15.v2"
)

var serv = flag.String("server", "irc.coldfront.net:6667", "hostname and port for irc server to connect to")
var nick = flag.String("nick", "hellabot", "nickname for the bot")

func main() {
	flag.Parse()

	hijackSession := func(bot *hbot.Bot) {
		bot.HijackSession = true
	}
	channels := func(bot *hbot.Bot) {
		bot.Channels = []string{"#test"}
	}
	irc, err := hbot.NewBot(*serv, *nick, hijackSession, channels)
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
	Condition: func(bot *hbot.Bot, mes *hbot.Message) bool {
		return mes.Command == "PRIVMSG" && mes.Content == "-info"
	},
	Action: func(irc *hbot.Bot, mes *hbot.Message) bool {
		irc.Reply(mes, "Hello")
		return false
	},
}

// This trigger replies Hello when you say hello
var longTrigger = hbot.Trigger{
	Condition: func(bot *hbot.Bot, mes *hbot.Message) bool {
		return mes.Command == "PRIVMSG" && mes.Content == "-long"
	},
	Action: func(irc *hbot.Bot, mes *hbot.Message) bool {
		irc.Reply(mes, "This is the first message")
		time.Sleep(5 * time.Second)
		irc.Reply(mes, "This is the second message")

		return false
	},
}
