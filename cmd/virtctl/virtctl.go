package main

import (
	"os"

	"fmt"
	flag "github.com/spf13/pflag"
	"kubevirt.io/kubevirt/pkg/virtctl"
	"log"
)

func main() {

	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	registry := map[string]virtctl.App{
		"options": &virtctl.Options{},
	}

	for cmd, app := range registry {
		f := app.FlagSet()
		f.Bool("help", false, "Print usage.")
		f.MarkHidden("help")
		flags, err := Parse(cmd, f)

		if err != nil {
			if flags == nil {
				// No subcommand specified
				break
			}
		}
		if flags != nil {
			h, _ := flags.GetBool("help")
			if h || err != nil {
				fmt.Fprint(os.Stderr, app.Usage())
				return
			}
			os.Exit(app.Run(flags))
		}
	}

	Usage()
	os.Exit(1)
}

func Parse(cmd string, flags *flag.FlagSet) (*flag.FlagSet, error) {
	flags.AddFlagSet((&virtctl.Options{}).FlagSet())
	err := flags.Parse(os.Args[1:])
	if len(flags.Args()) == 0 {
		return nil, fmt.Errorf("No subcommand found")
	}
	foundCmd := flags.Arg(0)
	if foundCmd != cmd {
		return nil, nil
	}
	return flags, err
}

func Usage() {
	fmt.Fprintln(os.Stderr,
		`virtctl controll VM related operations on your kubernetes cluster.

Basic Commands:

Use "virtctl <command> --help" for more information about a given command.
Use "virtctl options" for a list of global command-line options (applies to all commands).
	`)
}
