package main

import (
	"fmt"
	"flag"
	"github.com/whyrusleeping/hellabot"
)

func main() {
	nick := flag.String("nick", "hellabot", "nickname for the bot")
	serv := flag.String("server", "irc:6667", "hostname and port for irc server to connect to")
	ichan := flag.String("chan", "#go-nuts", "channel for bot to join")
	flag.Parse()

	irc := hbot.NewIrcConnection(*serv, *nick)

	//Say a message from a file when prompted
	irc.AddTrigger(SayInfoMessage)

	//Start up bot
	irc.Start()

	//join a channel
	mychannel := irc.Join(*ichan)
	mychannel.Say("Hey")

	for mes := range irc.Incoming {
		if mes == nil {
			fmt.Println("Disconnected.")
			return
		}
		//print out raw message struct
		fmt.Println(mes)
	}
	fmt.Println("Bot shutting down.")
}
