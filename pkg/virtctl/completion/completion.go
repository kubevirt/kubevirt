package completion

import (
	"os"

	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion SHELL",
		Short: "Output shell completion code for the specified shell (bash or zsh)",
		Example: `# Load the virtctl completion code for bash into the current shell
source <(virtctl completion bash)

# Load the virtctl completion code for zsh[1] into the current shell
source <(virtctl completion zsh)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "bash" {
				return cmd.Parent().GenBashCompletion(os.Stdout)
			} else if args[0] == "zsh" {
				return cmd.Parent().GenZshCompletion(os.Stdout)
			}
			return nil
		},
		ValidArgs: []string{"bash", "zsh"},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}
