package main

import (
	"flag"
	"log"

	hbot "github.com/whyrusleeping/hellabot"
	"github.com/whyrusleeping/hellabot/examples/commands/command"
	"github.com/whyrusleeping/hellabot/examples/commands/config"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

// Flags for passing arguments to the program
var configFile = flag.String("config", "production.toml", "path to config file")

// core holds the command environment (bot connection and db)
var core *command.Core

// cmdList holds our command list, which tells the bot what to respond to.
var cmdList *command.List

// Main method
func main() {
	// Parse flags, this is needed for the flag package to work.
	// See https://godoc.org/flag
	flag.Parse()
	// Read the TOML Config
	conf := config.FromFile(*configFile)
	// Validate the config to see it's not missing anything vital.
	config.ValidateConfig(conf)

	// Setup our options anonymous function.. This gets called on the hbot.Bot object internally, applying the options inside.
	options := func(bot *hbot.Bot) {
		bot.SSL = conf.SSL
		if conf.ServerPassword != "" {
			bot.Password = conf.ServerPassword
		}
		bot.Channels = conf.Channels
	}
	// Create a new instance of hbot.Bot
	bot, err := hbot.NewBot(conf.Server, conf.Nick, options)
	if err != nil {
		log.Fatal(err)
	}
	// Setup the command environment
	core = &command.Core{bot, &conf}
	// Add the command trigger (this is what triggers all command handling)
	bot.AddTrigger(CommandTrigger)
	// Set the default bot logger to stdout
	bot.Logger.SetHandler(log15.StdoutHandler)
	// Initialize the command list
	cmdList = &command.List{
		Prefix:   "!",
		Commands: make(map[string]command.Command),
	}
	// Add commands to handle
	cmdList.AddCommand(command.Command{
		Name:        "kudos",
		Description: "Send kudos to a teammate- '!kudos <teammate>'",
		Usage:       "!kudos <teammate>",
		Run:         core.Kudos,
	})
	cmdList.AddCommand(command.Command{
		Name:        "cve",                                                                // Trigger word
		Description: "Fetches information about the CVE number from http://cve.circl.lu/", // Description
		Usage:       "!cve CVE-2017-7494",                                                 // Usage example
		Run:         core.GetCVE,                                                          // Function or method to run when it triggers
	})

	// Start up bot (blocks until disconnect)
	bot.Run()
	log.Println("Bot shutting down.")
}

// CommandTrigger passes all incoming messages to the commandList parser.
var CommandTrigger = hbot.Trigger{
	func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "PRIVMSG"
	},
	func(bot *hbot.Bot, m *hbot.Message) bool {
		cmdList.Process(bot, m)
		return false
	},
}
