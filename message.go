package hbot

import (
	"strings"
	"time"

	"github.com/sorcix/irc"
)

type Message struct {
	// irc.Message from sorcix
	irc.Message
	// Content generally refers to the text of a PRIVMSG
	Content string

	//Time at which this message was recieved
	TimeStamp time.Time

	// Entity that this message was addressed to (channel or user)
	To string

	// Nick of the messages sender (equivalent to Prefix.Name)
	// Outdated, please use .Name
	From string

	// For debugging only, do not rely on this staying in the API
	Raw string
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
	mes.Raw = line

	var pref int
	if line[0] == ':' {
		pref = strings.IndexByte(line, ' ')

		// Require prefix
		if pref < 2 {
			return nil
		}

		mes.Prefix = irc.ParsePrefix(line[1:pref])
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
