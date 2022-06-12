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

func (o *SSH) runLocalCommandClient(kind, namespace, name string) error {

	args := []string{}
	args = append(args, o.buildProxyCommandOption(kind, namespace, name))
	args = append(args, o.buildSSHTarget(kind, namespace, name))

	cmd := exec.Command("ssh", args...)
	fmt.Println("running:", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return runCommand(cmd)
}

func (o *SSH) buildProxyCommandOption(kind, namespace, name string) string {
	proxyCommand := strings.Builder{}
	proxyCommand.WriteString("-o ProxyCommand=")
	proxyCommand.WriteString(os.Args[0])
	proxyCommand.WriteString(" port-forward --stdio=true ")
	proxyCommand.WriteString(fmt.Sprintf("%s/%s.%s", kind, name, namespace))
	proxyCommand.WriteString(" ")

	proxyCommand.WriteString(strconv.Itoa(o.options.SshPort))

	return proxyCommand.String()
}

func (o *SSH) buildSSHTarget(kind, namespace, name string) string {
	target := strings.Builder{}
	if o.options.IdentityFilePathProvided {
		target.WriteString(fmt.Sprintf(" -i %s ", o.options.IdentityFilePath))
	}
	if len(o.options.SshUsername) > 0 {
		target.WriteString(o.options.SshUsername)
		target.WriteRune('@')
	}
	target.WriteString(kind)
	target.WriteRune('/')
	target.WriteString(name)
	target.WriteRune('.')
	target.WriteString(namespace)
	return target.String()
}
