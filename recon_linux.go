package hbot

import (
	"fmt"
	"net"

	"github.com/mudler/sendfd"
)

func (irc *Bot) StartUnixListener() {
	unaddr, err := net.ResolveUnixAddr("unix", irc.unixastr)
	if err != nil {
		panic(err)
	}
	list, err := net.ListenUnix("unix", unaddr)
	if err != nil {
		panic(err)
	}

	irc.unixlist = list
	con, err := list.AcceptUnix()
	if err != nil {
		fmt.Println("unix listener error: ", err)
		return
	}
	list.Close()

	fi, err := irc.con.(*net.TCPConn).File()
	if err != nil {
		panic(err)
	}

	err = sendfd.SendFD(con, fi)
	if err != nil {
		panic(err)
	}

	select {
	case <-irc.Incoming:
	default:
		close(irc.Incoming)
	}
	close(irc.outgoing)
}

// Attempt to hijack session previously running bot
func (irc *Bot) HijackSession() bool {
	unaddr, err := net.ResolveUnixAddr("unix", irc.unixastr)
	if err != nil {
		irc.Error("could not resolve unix socket")
		return false
	}

	con, err := net.DialUnix("unix", nil, unaddr)
	if err != nil {
		irc.Error("Couldnt restablish connection, no prior bot:", err)
		return false
	}

	ncon, err := sendfd.RecvFD(con)
	if err != nil {
		panic(err)
	}

	netcon, err := net.FileConn(ncon)
	if err != nil {
		panic(err)
	}

	irc.reconnect = true
	irc.con = netcon
	return true
}
