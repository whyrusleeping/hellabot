package hbot

import (
	"net"
	"fmt"
	"bitbucket.org/madmo/sendfd"
)

func (irc *IrcCon) StartUnixListener() { }

// Attempt to hijack session previously running bot
func (irc *IrcCon) HijackSession() bool {
	return false
}

