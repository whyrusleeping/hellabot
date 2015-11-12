package hbot

import (
	"fmt"
	"net"

	"github.com/mudler/sendfd"
)

// StartUnixListener starts up a unix domain socket listener for reconnects to
// be sent through
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
func (irc *Bot) hijackSession() bool {
	unaddr, err := net.ResolveUnixAddr("unix", irc.unixastr) // The only way to get an error here is if the first parameter is not one of "unix", "unixgram" or "unixpacket". That will never happen.
	if err != nil {
		panic(err)
	}
	con, err := net.DialUnix("unix", nil, unaddr)
	if err != nil {
		irc.Info("Couldnt restablish connection, no prior bot.", "err", err)
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
	irc.reconnecting = true
	irc.con = netcon
	return true
}
