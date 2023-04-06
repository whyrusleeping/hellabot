package hbot

import (
	"bytes"
	"encoding/base64"
	"strings"
	"sync"
)

type saslAuth struct {
	mu     sync.Mutex
	enable bool
	user   string
	pass   string
}

func (h *saslAuth) SetAuth(user, pass string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.user = user
	h.pass = pass
}

func (h *saslAuth) IsAuthMessage(m *Message) bool {
	return (strings.TrimSpace(m.Content) == "sasl" && m.Param(1) == "ACK") ||
		(m.Command == "AUTHENTICATE" && m.Param(0) == "+") ||
		(m.Command == "903" || m.Command == "904")
}

func (h *saslAuth) Handle(bot *Bot, m *Message) bool {

	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.IsAuthMessage(m) {
		return false
	}

	if strings.TrimSpace(m.Content) == "sasl" && m.Param(1) == "ACK" {
		bot.Debug("Recieved SASL ACK")
		bot.Send("AUTHENTICATE PLAIN")
	}

	if m.Command == "AUTHENTICATE" && m.Param(0) == "+" {
		bot.Debug("Got auth message!")
		out := bytes.Join([][]byte{[]byte(h.user), []byte(h.user), []byte(h.pass)}, []byte{0})
		encpass := base64.StdEncoding.EncodeToString(out)
		bot.Send("AUTHENTICATE " + encpass)
	}

	// 903 RPL_SASLSUCCESS
	// 904 ERR_SASLFAIL
	if m.Command == "903" || m.Command == "904" {
		bot.Send("CAP END")
	}

	return false
}

// SASLAuthenticate performs SASL authentication
// ref: https://github.com/atheme/charybdis/blob/master/doc/sasl.txt
func (bot *Bot) SASLAuthenticate(user, pass string) {
	bot.sasl.SetAuth(user, pass)
	bot.addSASL.Do(func() { bot.AddTrigger(bot.sasl) })
	bot.Debug("Beginning SASL Authentication")
	bot.Send("CAP REQ :sasl")
	bot.SetNick(bot.Nick)
	bot.sendUserCommand(bot.Nick, bot.Realname)
}
