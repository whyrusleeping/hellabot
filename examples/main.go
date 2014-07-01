package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"flag"
	"github.com/whyrusleeping/hellabot"
)

var SayInfoMessage = &hbot.Trigger{
	func (m *hbot.Message) bool {
		return m.Type == "PRIVMSG" && m.Content == "-info"
	},
	func (irc *hbot.IrcCon, mes *hbot.Message) bool {
		fi,err := os.Open("info")
		if err != nil {
			return false
		}
		info,_ := ioutil.ReadAll(fi)

		irc.Send("PRIVMSG " + mes.From + " : " + string(info))
		return false
	},
}

func main() {
	nick := flag.String("nick", "hellabot", "nickname for the bot")
	serv := flag.String("server", "irc:6667", "hostname and port for irc server to connect to")
	ichan := flag.String("chan", "#go-nuts", "channel for bot to join")
	flag.Parse()

	irc := hbot.NewIrcConnection(*serv, *nick)

	//Respond to PING-PONG messages
	//Necessary to stay logged in
	irc.AddTrigger(hbot.PingPong)

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
