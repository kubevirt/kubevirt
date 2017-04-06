package main

import (
	"os"

	"fmt"
	flag "github.com/spf13/pflag"
	"kubevirt.io/kubevirt/pkg/virtctl"
	"kubevirt.io/kubevirt/pkg/virtctl/console"
	"log"
)

func main() {

	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	registry := map[string]virtctl.App{
		"console": &console.Console{},
		"options": &virtctl.Options{},
	}

	for cmd, app := range registry {
		f := app.FlagSet()
		f.Bool("help", false, "Print usage.")
		f.MarkHidden("help")
		f.Usage = func() {
			fmt.Fprint(os.Stderr, app.Usage())
		}

		if os.Args[1] != cmd {
			continue
		}
		flags, err := Parse(f)

		h, _ := flags.GetBool("help")
		if h || err != nil {
			f.Usage()
			return
		}
		os.Exit(app.Run(flags))
	}

	Usage()
	os.Exit(1)
}

func Parse(flags *flag.FlagSet) (*flag.FlagSet, error) {
	flags.AddFlagSet((&virtctl.Options{}).FlagSet())
	err := flags.Parse(os.Args[1:])
	return flags, err
}

func Usage() {
	fmt.Fprintln(os.Stderr,
		`virtctl controll VM related operations on your kubernetes cluster.

Basic Commands:
  console        Connect to a serial console on a VM

Use "virtctl <command> --help" for more information about a given command.
Use "virtctl options" for a list of global command-line options (applies to all commands).
	`)
}
