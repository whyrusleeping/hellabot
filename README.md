HellaBot
========

One hella-awesome irc bot. Hellabot is an easily hackable event based irc bot
framework. To respond to an event, simple create a "Trigger" struct containing
two functions, one for the condition, and one for the action.

Example Trigger:

	var MyTrigger = &Trigger{
		func (mes *Message) bool {
			return mes.From == "whyrusleeping"
		},
		func (irc *IrcCon, mes *Message) bool {
			irc.Channels[mes.To].Say("whyrusleeping said something")
		}

This trigger makes the bot announce to everyone that i said something
in whatever channel we are in. To make the bot actually use this,
add it like so:

	mybot := NewIrcConnection("irc.freenode.com:6667","hellabot")
	mybot.AddTrigger(MyTrigger)
	mybot.Start()



