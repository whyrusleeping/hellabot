package hbot

import (
	"os"
	"encoding/json"
	"fmt"
)

//Permission Flags
const (
	Voice = uint32(1) << iota
	Operator
	Wallops
	Invisibility
)

var ModeMap = map[string]uint32{
	"v" : Voice,
	"o" : Operator,
	"w" : Wallops,
	"i" : Invisibility,
}

type IrcUser struct {
	Nick string
	User string
	Host string

	Perms uint32
}

type IrcChannel struct {
	Name string
	con *IrcCon
	Counts map[string]int
	Perms uint32

	Users map[string]*IrcUser
}

// Attempt to load chat frequency stats from a file
func (c *IrcChannel) TryLoadStats(finame string) bool {
	fi,err := os.Open(finame)
	if err != nil {
		return false
	}
	defer fi.Close()

	dec := json.NewDecoder(fi)

	err = dec.Decode(&c.Counts)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

// Write chat frequencies to a file
func (c *IrcChannel) SaveStats(finame string) {
	fi,err := os.Create(finame)
	if err != nil {
		panic(err)
	}
	defer fi.Close()

	enc := json.NewEncoder(fi)
	enc.Encode(c.Counts)
}

// Send a message to this irc channel
func (c *IrcChannel) Say(text string) {
	_,err := fmt.Fprintf(c.con.con, "PRIVMSG %s :%s\r\n", c.Name, text)
	if err != nil {
		panic(err)
	}
}

// Sets the channels topic (requires bot has proper permissions)
func (c *IrcChannel) Topic(topic string) {
	str := fmt.Sprintf("TOPIC %s :%s", c.Name, topic)
	c.con.Send(str)
}

// Kick a user in this channel, reason optional (requires permissions)
func (c *IrcChannel) Kick(user, reason string) {
	c.con.Send(fmt.Sprintf("KICK %s %s :%s", c.Name, user, reason))
}
