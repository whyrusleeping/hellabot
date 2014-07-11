package hbot

import (
	"strings"
	"time"
)

// Message prefix, information about who sent this message
type Prefix struct {
	Name string
	User string
	Host string
}

type Message struct {
	*Prefix
	Content string
	Command string
	Params []string

	TimeStamp time.Time

	// Entity that this message was addressed to (channel or user)
	To string

	// Nick of the messages sender (equivalent to Prefix.Name)
	// Outdated, please use .Name
	From string
}

// Parse a string of text from the irc server into a Message struct
// Taken from: https://github.com/sorcix/irc
// All credit for this function goes to github user sorcix
func ParseMessage(line string) *Message {
	// Ignore empty messages.
	if line = strings.Trim(line, "\x20\r\n\x00"); len(line) < 2 {
		return nil
	}

	mes := new(Message)

	var pref int
	if line[0] == ':' {
		pref = strings.IndexByte(line, ' ')

		// Require prefix
		if pref < 2 {
			return nil
		}

		mes.Prefix = ParsePrefix(line[1:pref])
		// Skip space at the end of the prefix
		pref++
	}

	// Find end of command
	cmd := pref + strings.IndexByte(line[pref:], ' ')

	// Extract command
	if cmd > pref {
		mes.Command = line[pref:cmd]
	} else {
		mes.Command = line[pref:]
		return mes
	}

	// Skip space
	cmd++

	// Find prefix for trailer
	pref = strings.IndexByte(line[cmd:], ':')

	if pref < 0 {
		// There is no trailing argument!
		mes.Params = strings.Split(line[cmd:], " ")
		return mes
	}

	pref += cmd
	if pref > cmd {
		mes.Params = strings.Split(line[cmd:pref-1], " ")
	}

	// Everything after the last colon is the message contents
	mes.Content = line[pref+1:]

	if len(mes.Params) > 0 {
		mes.To = mes.Params[0]
	}
	if mes.Prefix != nil {
		mes.From = mes.Prefix.Name
	}
	mes.TimeStamp = time.Now()

	return mes
}

// Parse user information from string
// format: nick!user@hostname
// Taken from: https://github.com/sorcix/irc
// All credit for this function goes to github user sorcix
func ParsePrefix(prefix string) *Prefix {
	p := new(Prefix)

	user := strings.IndexByte(prefix, '!')
	host := strings.IndexByte(prefix, '@')

	switch {
	case user > 0 && host > user:
		p.Name = prefix[:user]
		p.User = prefix[user+1 : host]
		p.Host = prefix[host+1:]

	case user > 0:
		p.Name = prefix[:user]
		p.User = prefix[user+1:]

	case host > 0:
		p.Name = prefix[:host]
		p.Host = prefix[host+1:]

	default:
		p.Name = prefix
	}

	return p
}
