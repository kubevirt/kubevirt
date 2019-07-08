package templates

import (
	"fmt"
	"os"
	"strings"
)

func ProgramName() string {
	elements := strings.Split(os.Args[0], "/")
	return strings.Replace(elements[len(elements)-1], "-", " ", -1)
}

func PrepareTemplate(template string) string {
	return strings.Replace(template, "{{.ProgramName}}", ProgramName(), -1)
}

func PrependProgramName(template string) string {
	return fmt.Sprintf("%s %s", ProgramName(), template)
}

// UsageTemplate returns the usage template for all subcommands
func UsageTemplate() string {
	return PrepareTemplate(`Usage:{{if .Runnable}}
  {{.ProgramName}} {{.Use}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Use "{{.ProgramName}} options" for a list of global command-line options (applies to all commands).{{end}}
`)
}

// MainUsageTemplate returns the usage template for the root command
func MainUsageTemplate() string {
	return PrepareTemplate(`Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Use "{{.ProgramName}} <command> --help" for more information about a given command.
Use "{{.ProgramName}} options" for a list of global command-line options (applies to all commands).
`)
}

// OptionsUsageTemplate returns a template which prints all global available commands
func OptionsUsageTemplate() string {
	return `The following options can be passed to any command:{{if .HasAvailableInheritedFlags}}

{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`
}
