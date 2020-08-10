package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	hbot "github.com/whyrusleeping/hellabot"
)

// This trigger will op people in the given list who ask by saying "-opme"
var oplist = []string{"whyrusleeping", "tlane", "ltorvalds"}
var opPeople = hbot.Trigger{
	Condition: func(bot *hbot.Bot, mes *hbot.Message) bool {
		if mes.Content == "-opme" {
			for _, s := range oplist {
				if mes.From == s {
					return true
				}
			}
		}
		return false
	},
	Action: func(irc *hbot.Bot, mes *hbot.Message) bool {
		irc.ChMode(mes.To, mes.From, "+o")
		return false
	},
}

// This trigger will say the contents of the file "info" when prompted
var sayInfoMessage = hbot.Trigger{
	Condition: func(bot *hbot.Bot, mes *hbot.Message) bool {
		return mes.Command == "PRIVMSG" && mes.Content == "-info"
	},
	Action: func(irc *hbot.Bot, mes *hbot.Message) bool {
		fi, err := os.Open("info")
		if err != nil {
			return false
		}
		info, _ := ioutil.ReadAll(fi)

		irc.Send("PRIVMSG " + mes.From + " : " + string(info))
		return false
	},
}

// This trigger will listen for -toggle, -next and -prev and then
// perform the mpc action of the same name to control an mpd server running
// on localhost
var mpc = hbot.Trigger{
	Condition: func(bot *hbot.Bot, mes *hbot.Message) bool {
		return mes.Command == "PRIVMSG" && (mes.Content == "-toggle" || mes.Content == "-next" || mes.Content == "-prev")
	},
	Action: func(irc *hbot.Bot, mes *hbot.Message) bool {
		var mpcCMD string
		switch mes.Content {
		case "-toggle":
			mpcCMD = "toggle"
		case "-next":
			mpcCMD = "next"
		case "-prev":
			mpcCMD = "prev"
		default:
			fmt.Println("Invalid command.")
			return false
		}
		cmd := exec.Command("/usr/bin/mpc", mpcCMD)
		err := cmd.Run()
		if err != nil {
			fmt.Printf("error: %s\n", err)
		}
		return true
	},
}
