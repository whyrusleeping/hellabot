package main

import (
	"github.com/whyrusleeping/hellabot"
)

//This trigger will op people in the given list who ask by saying "-opme"
var oplist = []string{"whyrusleeping", "tlane", "ltorvalds"}
var OpPeople = &hbot.Trigger {
	func (mes *hbot.Message) bool {
		if mes.Content == "-opme" {
			for _,s := range oplist {
				if mes.From == s {
					return true
				}
			}
		}
		return false
	},
	func (irc *hbot.IrcCon, mes *hbot.Message) bool {
		irc.ChMode(mes.To, mes.From, "+o")
	},
}
