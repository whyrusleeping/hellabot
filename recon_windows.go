package hbot

func (irc *Bot) StartUnixListener() {}

// Attempt to hijack session previously running bot
func (irc *Bot) HijackSession() bool {
	return false
}
