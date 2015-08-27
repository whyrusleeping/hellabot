// This is an example program showing the usage of hellabot
package main

import (
	"flag"
	"fmt"

	hbot "github.com/JReyLBC/hellabot"
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
	irc.Logger.SetHandler(log.StdoutHandler)
	// logHandler := log.LvlFilterHandler(log.LvlInfo, log.StdoutHandler)
	// or
	// irc.Logger.SetHandler(logHandler)
	// or
	// irc.Logger.SetHandler(log.StreamHandler(os.Stdout, log.JsonFormat()))

	// Start up bot
	irc.Start()

	// Read off messages from the server
	for mes := range irc.Incoming {
		if mes == nil {
			fmt.Println("Disconnected.")
			return
		}
		// Log raw message struct
		//fmt.Println(mes)
	}
	fmt.Println("Bot shutting down.")
}

type SayInfoMessage struct{}

func (sim *SayInfoMessage) Action(bot *hbot.Bot, m *hbot.Message) bool {
	return m.Command == "PRIVMSG" && m.Content == "-info"
}

func (sim *SayInfoMessage) Condition(irc *hbot.Bot, mes *hbot.Message) bool {
	irc.Msg(mes.To, "Hello")
	return false
}

// This trigger replies Hello when you say hello
var sayInfoMessage = &SayInfoMessage{}
