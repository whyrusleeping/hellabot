package hbot

import (
	"strings"
)

//Represents any message coming from the server
/*
type Message struct {
	Type string
	From string
	To string
	Content string
}*/

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

	//
	To string
	From string
}

func ParseMessage(line string) *Message {
	// Ignore empty messages.
	if line = strings.Trim(line, "\x20\r\n\x00"); len(line) < 2 {
		return nil
	}

	mes := new(Message)

	pref := 0
	if line[0] == ':' {
		pref = strings.IndexByte(line, ' ')

		//Require prefix
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

	//Skip space
	cmd++

	// Find prefix for trailer
	pref = strings.IndexByte(line[cmd:], ':')

	if pref < 0 {
		// There is no trailing argument!
		mes.Params = strings.Split(line[cmd:], " ")
		return mes
	}

	pref += cmd

	//Parse args?
	if pref > cmd {
		mes.Params = strings.Split(line[cmd:pref-1], " ")
	}

	mes.Content = line[pref+1:]

	if len(mes.Params) > 0 {
		mes.To = mes.Params[0]
	}
	mes.From = mes.Prefix.Name

	return mes
}

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
