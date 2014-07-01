#HellaBot

One hella-awesome irc bot. Hellabot is an easily hackable event based irc bot
framework. To respond to an event, simple create a "Trigger" struct containing
two functions, one for the condition, and one for the action.

###Example Trigger:

	var MyTrigger = &Trigger{
		func (mes *Message) bool {
			return mes.From == "whyrusleeping"
		},
		func (irc *IrcCon, mes *Message) bool {
			irc.Channels[mes.To].Say("whyrusleeping said something")
		},
	}

This trigger makes the bot announce to everyone that i said something
in whatever channel we are in. To make the bot actually use this,
add it like so:

	mybot := NewIrcConnection("irc.freenode.com:6667","hellabot")
	mybot.AddTrigger(MyTrigger)
	mybot.Start()

##Connection Passing

Hellabot is able to restart without dropping its connection to the server
(on linux machines) by passing the tcp connection through a unix domain socket.
This allows you to update triggers and other addons without actually logging
your bot out of irc.

##Why?

What do I need an IRC bot for you ask? Why, I've gone through the trouble of
compiling a list of fun things for you!

- AutoOp Bot: ops you when you join the channel
- Stats counting bot: counts how often people talk in a channel
- Mock users you dont like by repeating what they say
- Fire a usb dart launcher on a given command
- Award praise to people for guessing a random number
- And many other 'fun' things!
