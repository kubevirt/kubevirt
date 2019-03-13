package templates

// UsageTemplate returns the usage template for all subcommands
func UsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Use "virtctl options" for a list of global command-line options (applies to all commands).{{end}}
`
}

// MainUsageTemplate returns the usage template for the root command
func MainUsageTemplate() string {
	return `Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Use "{{.CommandPath}} <command> --help" for more information about a given command.
Use "{{.CommandPath}} options" for a list of global command-line options (applies to all commands).
`
}

// OptionsUsageTemplate returns a template which prints all global available commands
func OptionsUsageTemplate() string {
	return `The following options can be passed to any command:{{if .HasAvailableInheritedFlags}}

{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`
}
