# HellaBot

[![GoDoc](https://godoc.org/github.com/whyrusleeping/hellabot?status.png)](https://godoc.org/github.com/whyrusleeping/hellabot)

One hella-awesome Internet Relay Chat (IRC) bot. Hellabot is an easily hackable event based IRC bot
framework with the ability to be updated without losing connection to the
server. To respond to an event, simply create a "Trigger" struct containing
two functions, one for the condition, and one for the action.

### Example Trigger

```go
var MyTrigger = hbot.Trigger{
	func (b *hbot.Bot, m *hbot.Message) bool {
		return m.From == "whyrusleeping"
	},
	func (b *hbot.Bot, m *hbot.Message) bool {
		b.Reply(m, "whyrusleeping said something")
		return false
	},
}
```

The trigger makes the bot announce to everyone that something was said in the current channel. Use the code snippet below to make the bot and add the trigger.

```go
mybot, err := hbot.NewBot("irc.freenode.net:6667","hellabot")
if err != nil {
    panic(err)
}
mybot.AddTrigger(MyTrigger)
mybot.Run() // Blocks until exit
```


The 'To' field on the message object in triggers will refer to the channel that
a given message is in, unless it is a server message, or a user to user private
message. In such cases, the field will be the target user's name.

For more example triggers, check the examples directory.

### The Message struct

The message struct is primarily what you will be dealing with when building
triggers or reading off the Incoming channel.
This is mainly the sorcix.Message struct with some additions.
See https://github.com/sorcix/irc/blob/master/message.go#L153

```go
 // Message represents a message received from the server
 type Message struct {
     // irc.Message from sorcix
     *irc.Message
     // Content generally refers to the text of a PRIVMSG
     Content string

     //Time at which this message was recieved
     TimeStamp time.Time

     // Entity that this message was addressed to (channel or user)
     To string

     // Nick of the messages sender (equivalent to Prefix.Name)
     // Outdated, please use .Name
     From string
 }
```


### Connection Passing

Hellabot is able to restart without dropping its connection to the server
(on Linux machines) by passing the TCP connection through a UNIX domain socket.
This allows you to update triggers and other addons without actually logging
your bot out of the IRC, avoiding the loss of op status and spamming the channel
with constant join/part messages. To do this, run the program again with
the same nick and without killing the first program (different nicks wont reuse
the same bot instance). The first program will shutdown, and the new one
will take over.

****This does not work with SSL connections, because we can't hand over a SSL connections state.****

### Security

Hellabot supports both SSL and SASL for secure connections to whichever server
you like. To enable SSL, pass the following option to the NewBot function.

```go
sslOptions := func(bot *hbot.Bot) {
    bot.SSL = true
}

mysslcon, err := hbot.NewBot("irc.freenode.net:6667","hellabot",sslOptions)
// Handle err as you like

mysslcon.Run() # Blocks until disconnect.
```

To use SASL to authenticate with the server:

```go
saslOption = func(bot *hbot.Bot) {
    bot.SASL = true
    bot.Password = "somepassword"
}

mysslcon, err := hbot.NewBot("irc.freenode.net:6667", "hellabot", saslOption)
// Handle err as you like

mysslcon.Run() # Blocks until disconnect.
```

Note: SASL does not require SSL but can be used in combination.

### Passwords

For servers that require passwords in the initial registration, simply set
the Password field of the Bot struct before calling its Start method.

### Debugging

Hellabot uses github.com/inconshreveable/log15 for logging.
See http://godoc.org/github.com/inconshreveable/log15

By default it discards all logs. In order to see any logs, give it a better handler.
Example: This would only show INFO level and above logs, logging to STDOUT

```go
import log "gopkg.in/inconshreveable/log15.v2"

logHandler := log.LvlFilterHandler(log.LvlInfo, log.StdoutHandler)
mybot.Logger.SetHandler(logHandler)
```
Note: This might be revisited in the future.

### Why?

What do you need an IRC bot for you ask? Well, I've gone through the trouble of
compiling a list of fun things for you! The following are some of the things hellabot is
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


### Contributors
- @whyrusleeping
- @flexd
- @Luzifer
- @mudler
- @JReyLBC
- @ForrestWeston
- @miloprice

