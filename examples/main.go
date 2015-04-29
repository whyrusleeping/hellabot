// This is an example program showing the usage of hellabot
package main

import (
	"flag"
	"fmt"
	"github.com/whyrusleeping/hellabot"
)

func main() {
	nick := flag.String("nick", "hellabot", "nickname for the bot")
	serv := flag.String("server", "irc:6667", "hostname and port for irc server to connect to")
	ichan := flag.String("chan", "#go-nuts", "channel for bot to join")
	flag.Parse()

	irc, err := hbot.NewIrcConnection(*serv, *nick, false, false)
	if err != nil {
		panic(err)
	}

	// Say a message from a file when prompted
	irc.AddTrigger(SayInfoMessage)

	// Start up bot
	irc.Start()

	// Join a channel
	mychannel := irc.Join(*ichan)
	mychannel.Say("Hey")

	// Read off messages from the server
	for mes := range irc.Incoming {
		if mes == nil {
			fmt.Println("Disconnected.")
			return
		}
		// Log raw message struct
		fmt.Println(mes)
	}
	fmt.Println("Bot shutting down.")
}
