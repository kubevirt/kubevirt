package virtctl

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virtctl/console"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/pkg/virtctl/imageupload"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
	"kubevirt.io/kubevirt/pkg/virtctl/version"
	"kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/pkg/virtctl/vnc"
)

func NewVirtctlCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "virtctl",
		Short:         "virtctl controls virtual machine related operations on your kubernetes cluster.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}

	optionsCmd := &cobra.Command{
		Use:    "options",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}
	optionsCmd.SetUsageTemplate(templates.OptionsUsageTemplate())
	//TODO: Add a ClientConfigFactory which allows substituting the KubeVirt client with a mock for unit testing
	clientConfig := kubecli.DefaultClientConfig(rootCmd.PersistentFlags())
	AddGlogFlags(rootCmd.PersistentFlags())
	rootCmd.SetUsageTemplate(templates.MainUsageTemplate())
	rootCmd.AddCommand(
		console.NewCommand(clientConfig),
		vnc.NewCommand(clientConfig),
		vm.NewStartCommand(clientConfig),
		vm.NewStopCommand(clientConfig),
		vm.NewRestartCommand(clientConfig),
		expose.NewExposeCommand(clientConfig),
		version.VersionCommand(clientConfig),
		imageupload.NewImageUploadCommand(clientConfig),
		optionsCmd,
	)
	return rootCmd
}

func Execute(args []string) error {
	log.InitializeLogging("virtctl")
	// check whether there is a MIME config file
	// if yes, then set up the args for commands and execute
	// otherwise normal operation
	confArgs, err := parseMime(args)
	if err != nil {
		return err
	}

	cmd := NewVirtctlCommand()

	if confArgs != nil {
		cmd.SetArgs(confArgs)
	} else {
		cmd.SetArgs(args)
	}

	if err := cmd.Execute(); err != nil {
		return err
	}

	return nil
}

func parseMime(args []string) ([]string, error) {
	var parsedArgs []string

	for ind := 0; ind < len(args); ind++ {

		filePath, err := checkForFile(args[ind])
		if err == nil && filePath != "" {
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("Cannot read config file: %s", filePath)
			}

			mimeArgs, err := parseMimeConfig(string(data))
			if err != nil {
				return nil, err
			}
			for _, mimeArg := range mimeArgs {
				parsedArgs = append(parsedArgs, mimeArg)
			}

			return parsedArgs, nil
		} else if err != nil {
			return nil, err
		}

		if args[ind] == "--kubeconfig" || args[ind] == "--server" {
			parsedArgs = append(parsedArgs, args[ind], args[ind+1])
			ind = ind + 1
		}

	}

	return nil, nil
}

func checkForFile(arg string) (string, error) {
	var err error
	filePath := ""

	if strings.HasSuffix(arg, ".vvv") {
		exist, _ := util.FileExists(arg)
		if exist {
			filePath = arg
		} else {
			err = errors.New("File does not exist")
		}
	}

	return filePath, err
}

func parseMimeConfig(data string) ([]string, error) {
	// read only single line as expected
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Scan()
	line := scanner.Text()

	tokens := strings.Split(line, " ")
	if len(tokens) != 3 {
		// invalid param line
		return nil, fmt.Errorf("Invalid file format, 3 parameters required, %d received", len(tokens))
	}

	// first token have to be one of vnc|console
	if tokens[0] != "vnc" && tokens[0] != "console" {
		return nil, fmt.Errorf("Protocol have to be one of: vnc, console. Got: %s", tokens[0])
	}

	for _, token := range tokens {
		matched, err := regexp.MatchString(`^[a-z\d\-_]+$`, token)
		if err != nil {
			return nil, fmt.Errorf("Cannot parse token: %s", token)
		}
		if !matched {
			return nil, fmt.Errorf("Token containing illegal character: %s", token)
		}
	}

	tokens[1] = "--namespace=" + tokens[1]

	return tokens, nil
}
