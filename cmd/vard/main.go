// Command vard is the server, and the migration runner that goes with it.
//
// With no command, or with serve, it runs the provider and account API on one
// listener and the admin API on the other until it is told to stop. migrate
// up, down or status steps the database through migrations/, and migrate new
// <name> writes a new, empty migration there. admin create makes the first
// administrator from the command line.
package main

import (
	"fmt"
	"os"
)

// version is set at build time (-ldflags "-X main.version=…", from
// `git describe`, which names the commit too).
var version = "dev"

const usage = `usage: %s [command]

  serve     run the server (the default)
  migrate   up, down or status of the database migrations, or new <name>
  admin     create the first administrator: admin create -email … -first-name …,
            with the password on standard input
`

func main() {
	args := os.Args[1:]
	command := "serve"
	if len(args) > 0 {
		command, args = args[0], args[1:]
	}

	switch command {
	case "serve":
		os.Exit(serveCommand())
	case "migrate":
		os.Exit(migrateCommand(args))
	case "admin":
		os.Exit(adminCommand(args))
	case "help", "-h", "-help", "--help":
		fmt.Fprintf(os.Stdout, usage, os.Args[0])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", command)
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		os.Exit(2)
	}
}
