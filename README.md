# HellaBot

One hella-awesome irc bot. Hellabot is an easily hackable event based irc bot
framework with the ability to be updated without losing connection to the
server. To respond to an event, simply create a "Trigger" struct containing
two functions, one for the condition, and one for the action.

###Example Trigger

```go
var MyTrigger = &Trigger{
	func (mes *Message) bool {
		return mes.From == "whyrusleeping"
	},
	func (irc *IrcCon, mes *Message) bool {
		irc.Channels[mes.To].Say("whyrusleeping said something")
		return false
	},
}
```

This trigger makes the bot announce to everyone that I said something
in whatever channel we are in. To make the bot actually use this,
add it like so:

```go
mybot,err := NewIrcConnection("irc.freenode.net:6667","hellabot",false)
// Handle err if you like
mybot.AddTrigger(MyTrigger)
mybot.Start()
```

The 'To' field on the message object in triggers will refer to the channel that
a given message is in, unless it is a server message, or a user to user private
message, in which case it will be the target user's name.

For more example triggers, check the examples directory.

### General Usage
All incoming messages not consumed by a trigger are placed into the IrcCon's
Incoming channel. If not removed, they will fill up the channel and cause the
program to hang. To avoid this either write a for-range loop to pull and log
messages off of Incoming, or simply add a trigger that does nothing but consume
all messages and make sure it is the last trigger added.

```go
var EatEverything = &Trigger{
	func (mes *Message) bool {
		return true
	},
	func (irc *IrcCon, mes *Message) bool {
		return true
	},
}

mybot.AddTrigger(EatEverything)
```

Alternatively:

```go
for mes := range mybot.Incoming {
	log.Println(mes)
}
```

### The Message struct

The message struct is primarily what you will be dealing with when building
triggers or reading off the Incoming channel.

```go
type Message struct {
	// The message prefix contains information about who sent the message
	*Prefix

	// Content generally refers to the text of a PRIVMSG
	Content string

	// Message command, ie PRIVMSG, MODE, JOIN, NICK, etc
	Command string

	// Command parameters
	// For example, which mode for MODE commands
	Params []string

	//Time at which this message was recieved
	TimeStamp time.Time

	// Entity that this message was addressed to (channel or user)
	To string

	// Nick of the messages sender (equivalent to Prefix.Name)
	// Outdated, please use .Name
	From string
}

type Prefix struct {
	// The senders nick
	Name string

	// The senders username
	User string

	// The senders hostname
	Host string
}
```


### Connection Passing

Hellabot is able to restart without dropping its connection to the server
(on Linux machines) by passing the TCP connection through a UNIX domain socket.
This allows you to update triggers and other addons without actually logging
your bot out of the IRC, avoiding the loss of op status and spamming the channel
with constant join/part messages. To do this, simply run the program again with
the same nick and without killing the first program (different nicks wont reuse
the same bot instance). The first program will shutdown cleanly, and the new one
will take over.

### Security

Hellabot supports both SSL and SASL for secure connections to whichever server
you like. To enable SSL simple pass 'true' as the third argument to the
NewIrcConnection function.

```
mysslcon,err := NewIrcConnection("irc.freenode.net:6667","hellabot",true)
// Handle err if you like
```

To use SASL to authenticate with the server:

```go
mysslcon.DoSasl = true
mysslcon.Password = "MyPassword"
mysslcon.Start()
```

Note: SASL does not require SSL.

### Passwords

For servers that require passwords in the initial registration, simply set
the Password field of the IrcCon struct before calling its Start method.

### Debugging

The hbot package has a global variable called Verbosity. It controls
hellabot's internal logging levels. There are six levels of logging at the time
of this writing, but not all are currently used.

```go
// For error conditions
LError 
LWarning

// For tracing code paths
LTrace

// For providing extra info about hellabots state
LInfo
LNotice

// For logging every little detail that happens
LNoise
```


### Why?

What do you need an IRC bot for you ask? Why, I've gone through the trouble of
compiling a list of fun things for you! Some of these are what hellabot is
currently being used for:

- AutoOp Bot: ops you when you join the channel
- Stats counting bot: counts how often people talk in a channel
- Mock users you don't like by repeating what they say
- Fire a USB dart launcher on a given command
- Control an MPD radio stream based on chat commands
- Award praise to people for guessing a random number
- Scrape news sites for relevant articles and send them to a channel
- And many other 'fun' things!

### References

[Client Protocol, RFC 2812](http://tools.ietf.org/html/rfc2812)
[SASL Authentication Documentation](https://tools.ietf.org/html/draft-mitchell-irc-capabilities-01)

### Credits

[sorcix](http://github.com/sorcix) for his Message Parsing code
