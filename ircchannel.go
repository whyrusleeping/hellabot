package hbot

import (
	"os"
	"encoding/json"
	"fmt"
)

type IrcChannel struct {
	Name string
	con *IrcCon
	Counts map[string]int
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

