package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var runCommand = func(cmd *exec.Cmd) error {
	return cmd.Run()
}

func RunLocalClient(kind, namespace, name string, options *SSHOptions, clientArgs []string) error {
	args := []string{"-o"}
	args = append(args, buildProxyCommandOption(kind, namespace, name, options.SSHPort))

	if len(options.AdditionalSSHLocalOptions) > 0 {
		args = append(args, options.AdditionalSSHLocalOptions...)
	}
	if options.IdentityFilePathProvided {
		args = append(args, "-i", options.IdentityFilePath)
	}

	args = append(args, clientArgs...)

	cmd := exec.Command(options.LocalClientName, args...)
	fmt.Println("running:", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return runCommand(cmd)
}

func buildProxyCommandOption(kind, namespace, name string, port int) string {
	proxyCommand := strings.Builder{}
	proxyCommand.WriteString("ProxyCommand=")
	proxyCommand.WriteString(os.Args[0])
	proxyCommand.WriteString(" port-forward --stdio=true ")
	proxyCommand.WriteString(fmt.Sprintf("%s/%s.%s", kind, name, namespace))
	proxyCommand.WriteString(" ")

	proxyCommand.WriteString(strconv.Itoa(port))

	return proxyCommand.String()
}

func (o *SSH) buildSSHTarget(kind, namespace, name string) (opts []string) {
	target := strings.Builder{}
	if len(o.options.SSHUsername) > 0 {
		target.WriteString(o.options.SSHUsername)
		target.WriteRune('@')
	}
	target.WriteString(kind)
	target.WriteRune('/')
	target.WriteString(name)
	target.WriteRune('.')
	target.WriteString(namespace)

	opts = append(opts, target.String())
	if o.command != "" {
		opts = append(opts, o.command)
	}
	return
}
