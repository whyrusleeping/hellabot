// +build !linux,!freebsd,!openbsd,!dragonfly,!netbsd,!darwin,!solaris

package hbot

func (irc *Bot) StartUnixListener() {}

// Attempt to hijack session previously running bot
func (irc *Bot) hijackSession() bool {
	return false
}
