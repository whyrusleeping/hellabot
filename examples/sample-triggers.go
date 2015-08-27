package main

import (
	"fmt"
	"os/exec"

	hbot "github.com/JReyLBC/hellabot"
)

// This trigger will op people in the given list who ask by saying "-opme"
var oplist = []string{"whyrusleeping", "tlane", "ltorvalds"}

type OpPeople struct{}

func (op *OpPeople) Condition(bot *hbot.Bot, mes *hbot.Message) bool {
	if mes.Content == "-opme" {
		for _, s := range oplist {
			if mes.From == s {
				return true
			}
		}
	}
	return false
}
func (op *OpPeople) Action(irc *hbot.Bot, mes *hbot.Message) bool {
	irc.ChMode(mes.To, mes.From, "+o")
	return false
}

var opPeople = &OpPeople{}

type Mpc struct{}

func (mpc *Mpc) Condition(bot *hbot.Bot, m *hbot.Message) bool {
	return m.Command == "PRIVMSG" && (m.Content == "-toggle" || m.Content == "-next" || m.Content == "-prev")
}

func (mpc *Mpc) Action(irc *hbot.Bot, m *hbot.Message) bool {
	var mpcCMD string
	switch m.Content {
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
}

// This trigger will listen for -toggle, -next and -prev and then
// perform the mpc action of the same name to control an mpd server running
// on localhost
var mpc = &Mpc{}
