// This is an example program showing the usage of hellabot
package main

import (
	"flag"
	"fmt"

	"github.com/flexd/hellabot"
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

	irc.AddTrigger(SayInfoMessage)
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
var SayInfoMessage = hbot.Trigger{
	func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "PRIVMSG" && m.Content == "-info"
	},
	func(irc *hbot.Bot, mes *hbot.Message) bool {
		irc.Msg(mes.To, "Hello")
		return false
	},
}
