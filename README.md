# HellaBot

One hella-awesome irc bot. Hellabot is an easily hackable event based irc bot
framework with the ability to be updated without losing connection to the
server. To respond to an event, simple create a "Trigger" struct containing
two functions, one for the condition, and one for the action.

###Example Trigger

	var MyTrigger = &Trigger{
		func (mes *Message) bool {
			return mes.From == "whyrusleeping"
		},
		func (irc *IrcCon, mes *Message) bool {
			irc.Channels[mes.To].Say("whyrusleeping said something")
			return false
		},
	}

This trigger makes the bot announce to everyone that I said something
in whatever channel we are in. To make the bot actually use this,
add it like so:

	mybot := NewIrcConnection("irc.freenode.net:6667","hellabot",false)
	mybot.AddTrigger(MyTrigger)
	mybot.Start()

The 'To' field on the message object in triggers will refer to the channel that
a given message is in, unless it is a server message or a user to user private
message, In which case it will be the target users name.

For more example triggers, check the examples directory.

### General Usage
All incoming messages not consumed by a trigger are placed into the IrcCon's
Incoming channel. If not removed, they will fill up the channel and cause the
program to hang. To avoid this either write a for-range loop to pull and log
messages off of Incoming, or simply add a trigger than does nothing but consume
all messages and make sure it is the last trigger added.

	var EatEverything = &Trigger{
		func (mes *Message) bool {
			return true
		},
		func (irc *IrcCon, mes *Message) bool {
			return true
		},
	}

	mybot.AddTrigger(EatEverything)

Alternatively:

	for mes := range mybot.Incoming {
		log.Println(mes)
	}

### The Message struct

The message struct is the primary struct you will be dealing with when building
triggers or reading off the Incoming channel.

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



### Connection Passing

Hellabot is able to restart without dropping its connection to the server
(on linux machines) by passing the tcp connection through a unix domain socket.
This allows you to update triggers and other addons without actually logging
your bot out of irc. To do this, simply run the program again with the same nick 
and without killing the first program (different nicks wont reuse the same bot
instance), the first program will shutdown cleanly and the new one will take
over.

### Security

Hellabot supports both SSL and SASL for secure connections to whichever server
you like. To enable SSL simple pass 'true' as the third argument to the
NewIrcConnection function.

	mysslcon := NewIrcConnection("irc.freenode.net:6667","hellabot",true)

To use SASL to authenticate with the server:

	mysslcon.DoSasl = true
	mysslcon.Password = "MyPassword"
	mysslcon.Start()

Note: SASL does not require SSL.

### Passwords

For servers that require passwords in the initial registration, simple set
the Password field of the IrcCon struct before calling Start on it.

### Why?

What do you need an IRC bot for you ask? Why, I've gone through the trouble of
compiling a list of fun things for you! (Some of these are what hellabot is
currently being used for)

- AutoOp Bot: ops you when you join the channel
- Stats counting bot: counts how often people talk in a channel
- Mock users you dont like by repeating what they say
- Fire a usb dart launcher on a given command
- Control an MPD radio stream based on chat commands
- Award praise to people for guessing a random number
- Scrape news sites for relevant articles and send them to a channel
- And many other 'fun' things!

### Credits

[sorcix](http://github.com/sorcix) for his Message Parsing code
