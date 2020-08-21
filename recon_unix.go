// +build freebsd openbsd dragonfly netbsd darwin solaris

package hbot

import (
	"fmt"
	"net"
	"syscall"

	"github.com/ftrvxmtrx/fd"
)

// StartUnixListener starts up a unix domain socket listener for reconnects to
// be sent through
func (bot *Bot) StartUnixListener() {
	unaddr, err := net.ResolveUnixAddr("unix", bot.unixsock)
	if err != nil {
		panic(err)
	}

	// Unlink the socket so we don't have to worry about removing it
	// We can ignore any error here
	syscall.Unlink(bot.unixsock)

	list, err := net.ListenUnix("unix", unaddr)
	if err != nil {
		panic(err)
	}
	defer list.Close()
	bot.unixlist = list

	con, err := list.AcceptUnix()
	if err != nil {
		fmt.Println("unix listener error: ", err)
		return
	}
	defer con.Close()

	fi, err := bot.con.(*net.TCPConn).File()
	if err != nil {
		panic(err)
	}

	err = fd.Put(con, fi)
	if err != nil {
		panic(err)
	}

	select {
	case <-bot.Incoming:
	default:
		close(bot.Incoming)
	}
	close(bot.outgoing)
}

// Attempt to hijack session previously running bot
func (bot *Bot) hijackSession() bool {
	con, err := net.Dial("unix", bot.unixsock)
	if err != nil {
		bot.Info("Couldnt restablish connection, no prior bot.", "err", err)
		return false
	}
	defer con.Close()

	ncon, err := fd.Get(con.(*net.UnixConn), 1, nil)
	if err != nil {
		panic(err)
	}
	defer ncon[0].Close()

	netcon, err := net.FileConn(ncon[0])
	if err != nil {
		panic(err)
	}
	bot.reconnecting = true
	bot.con = netcon
	return true
}
