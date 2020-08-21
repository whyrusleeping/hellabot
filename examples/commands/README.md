# Slightly more advanced example
A slightly more advanced example that reads a TOML config file (with very basic validation),
and has a command system where you can add commands to a list and avoids us having to repeat ourselves with triggers.

!help will return all commands added, and tell the user the prefix for them (for example "!").
!help <command> will return the help text provided when you add the command, like this

### Example command to add
```go
	cmdList.AddCommand(command.Command{
		Name:        "cve", // Trigger word
		Description: "Fetches information about the CVE number from http://cve.circl.lu/", // Description
		Usage:       "!cve CVE-2017-7494", // Usage example
		Run:         core.GetCVE, // Function or method to run when it triggers (gets passed everything after the command word as a slice)
	})

```

### The function signature for the "Run" function for commands
```go
// Func represents the Go function that will be executed when a command triggers
type Func func(m *hbot.Message, args []string)
```

It's very basic, but it works pretty well. It could easily be expanded to do more validation (for example require N arguments) that we could check in the "Process"-method, but I just do that in the individual commands and it has worked just fine.

Run this example with go run main.go -config dev.toml
