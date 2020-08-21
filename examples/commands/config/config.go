package config

import (
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// FromFile reads the specified TOML configuration file and returns a Config object.
func FromFile(configFile string) Config {
	_, err := os.Stat(configFile)
	if err != nil {
		log.Fatal("Config file is missing: ", "path", configFile)
	}

	var config Config
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatal("error decoding config file", "error", err)
	}
	return config
}

// Config holds the bot's configuration
type Config struct {
	Server         string
	Nick           string
	ServerPassword string
	Channels       []string
	SSL            bool
}

// ValidateConfig checks that the config object has all the values it should, and fatally fails if not.
func ValidateConfig(config Config) {
	if config.Server == "" {
		log.Fatal("empty server address, can't continue")
	} else if !strings.Contains(config.Server, ":") {
		log.Fatal("server address needs to be in format <host/ip>:<port>")
	}
	if config.Nick == "" {
		log.Fatal("empty nickname, can't continue")
	}
	if len(config.Channels) == 0 {
		log.Fatal("no channels configured, can't continue")
	}
}
