package templates

import (
	"fmt"
	"slices"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

// GetProgramName returns the command name to display in help texts.
// If `virtctl` is installed via krew to be used as a kubectl plugin, it's run via a symlink, so the basename then
// is `kubectl-virt`. In this case we want to accommodate the user by adjusting the help text (usage, examples and
// the like) by displaying `kubectl virt <command>` instead of `virtctl <command>`.
// see https://github.com/kubevirt/kubevirt/issues/2356 for more details
// see also templates.go
func GetProgramName(binary string) string {
	if strings.HasSuffix(binary, "-virt") {
		return strings.TrimSuffix(binary, "-virt") + " virt"
	}
	return binary
}

func ActsAsRootCommand(cmd *cobra.Command, filters []string, programName string, groups ...CommandGroup) {
	if cmd == nil {
		panic("nil root command")
	}
	t := &templater{
		RootCmd:       cmd,
		UsageTemplate: MainUsageTemplate,
		HelpTemplate:  MainHelpTemplate,
		CommandGroups: groups,
		Filtered:      filters,
		ProgramName:   programName,
	}
	cmd.SilenceUsage = true
	cmd.SetUsageFunc(t.UsageFunc())
	cmd.SetHelpFunc(t.HelpFunc())
}

func UseOptionsTemplates(cmd *cobra.Command) {
	t := &templater{
		UsageTemplate: OptionsUsageTemplate,
		HelpTemplate:  "",
	}
	cmd.SetUsageFunc(t.UsageFunc())
	cmd.SetHelpFunc(t.HelpFunc())
}

type templater struct {
	UsageTemplate string
	HelpTemplate  string
	RootCmd       *cobra.Command
	CommandGroups
	Filtered    []string
	ProgramName string
}

func (templater *templater) HelpFunc() func(*cobra.Command, []string) {
	return func(c *cobra.Command, s []string) {
		t := template.New("help")
		t.Funcs(templater.templateFuncs())
		template.Must(t.Parse(templater.HelpTemplate))
		err := t.Execute(c.OutOrStdout(), c)
		if err != nil {
			c.Println(err)
		}
	}
}

func (templater *templater) UsageFunc() func(*cobra.Command) error {
	return func(c *cobra.Command) error {
		t := template.New("usage")
		t.Funcs(templater.templateFuncs())
		template.Must(t.Parse(templater.UsageTemplate))
		return t.Execute(c.OutOrStderr(), c)
	}
}

func (templater *templater) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"trim":                strings.TrimSpace,
		"trimRight":           func(s string) string { return strings.TrimRightFunc(s, unicode.IsSpace) },
		"flagsNotIntersected": flagsNotIntersected,
		"visibleFlags":        visibleFlags,
		"flagsUsages":         flagsUsages,
		"cmdGroupsString":     templater.cmdGroupsString,
		"rootCmd":             templater.rootCmdName,
		"optionsCmdFor":       templater.optionsCmdFor,
		"usageLine":           templater.usageLine,
		"reverseParentsNames": templater.reverseParentsNames,
		"prepare":             templater.prepare,
	}
}

func (templater *templater) cmdGroups(c *cobra.Command, all []*cobra.Command) []CommandGroup {
	if len(templater.CommandGroups) > 0 && c == templater.RootCmd {
		all = filter(all, templater.Filtered...)
		return AddNewGroup(templater.CommandGroups, "Other Commands:", all)
	}
	all = filter(all, "options")
	return []CommandGroup{
		{
			Message:  "Available Commands:",
			Commands: all,
		},
	}
}

func (templater *templater) cmdGroupsString(c *cobra.Command) string {
	groups := []string{}
	for _, cmdGroup := range templater.cmdGroups(c, c.Commands()) {
		cmds := []string{cmdGroup.Message}
		for _, cmd := range cmdGroup.Commands {
			if cmd.IsAvailableCommand() {
				cmds = append(cmds, "  "+rpad(cmd.Name(), cmd.NamePadding())+"   "+cmd.Short)
			}
		}
		groups = append(groups, strings.Join(cmds, "\n"))
	}
	return strings.Join(groups, "\n\n")
}

func (templater *templater) rootCmdName(c *cobra.Command) string {
	return templater.rootCmd(c).CommandPath()
}

func (templater *templater) reverseParentsNames(c *cobra.Command) []string {
	reverseParentsNames := []string{}
	parents := templater.parents(c)
	for i := len(parents) - 1; i >= 0; i-- {
		reverseParentsNames = append(reverseParentsNames, parents[i].Name())
	}
	return reverseParentsNames
}

func (templater *templater) isRootCmd(c *cobra.Command) bool {
	return templater.rootCmd(c) == c
}

func (templater *templater) parents(c *cobra.Command) []*cobra.Command {
	parents := []*cobra.Command{c}
	for current := c; !templater.isRootCmd(current) && current.HasParent(); {
		current = current.Parent()
		parents = append(parents, current)
	}
	return parents
}

func (templater *templater) rootCmd(c *cobra.Command) *cobra.Command {
	if c != nil && !c.HasParent() {
		return c
	}
	if templater.RootCmd == nil {
		panic("nil root cmd")
	}
	return templater.RootCmd
}

func (templater *templater) optionsCmdFor(c *cobra.Command) string {
	if !c.Runnable() {
		return ""
	}
	rootCmdStructure := templater.parents(c)
	for i := len(rootCmdStructure) - 1; i >= 0; i-- {
		cmd := rootCmdStructure[i]
		if _, _, err := cmd.Find([]string{"options"}); err == nil {
			path := cmd.CommandPath()
			root := rootCmdStructure[len(rootCmdStructure)-1]
			if strings.HasPrefix(path, root.Name()) {
				path = strings.Replace(path, root.Name(), templater.ProgramName, 1)
			}
			return path + " options"
		}
	}
	return ""
}

func (templater *templater) usageLine(c *cobra.Command) string {
	const suffix = "[options]"
	usage := c.UseLine()
	if c == templater.RootCmd {
		return usage
	}
	if c.HasFlags() && !strings.Contains(usage, suffix) {
		usage += " " + suffix
	}
	return templater.replaceRootWithProgramName(usage)
}

func (templater *templater) prepare(s string) string {
	return strings.ReplaceAll(s, "{{ProgramName}}", templater.ProgramName)
}

func (templater *templater) replaceRootWithProgramName(s string) string {
	root := templater.rootCmd(nil)
	if strings.HasPrefix(s, root.Name()) {
		return strings.Replace(s, root.Name(), templater.ProgramName, 1)
	}
	return s
}

// flagsUsages will print out the virtctl help flags
func flagsUsages(f *flag.FlagSet) string {
	if f == nil {
		return ""
	}
	return strings.TrimRight(f.FlagUsages(), "\n")
}

func rpad(s string, padding int) string {
	t := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(t, s)
}

func flagsNotIntersected(l *flag.FlagSet, r *flag.FlagSet) *flag.FlagSet {
	f := flag.NewFlagSet("notIntersected", flag.ContinueOnError)
	l.VisitAll(func(flag *flag.Flag) {
		if r.Lookup(flag.Name) == nil {
			f.AddFlag(flag)
		}
	})
	return f
}

func visibleFlags(l *flag.FlagSet) *flag.FlagSet {
	hidden := "help"
	f := flag.NewFlagSet("visible", flag.ContinueOnError)
	l.VisitAll(func(flag *flag.Flag) {
		if flag.Name != hidden {
			f.AddFlag(flag)
		}
	})
	return f
}

func filter(cmds []*cobra.Command, names ...string) []*cobra.Command {
	out := []*cobra.Command{}
	for _, c := range cmds {
		if c.Hidden {
			continue
		}
		if slices.Contains(names, c.Name()) {
			continue
		}
		out = append(out, c)
	}
	return out
}
