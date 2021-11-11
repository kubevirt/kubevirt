package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func runLocalCommandClient(kind, namespace, name string, options *SSHOptions) error {
	args := []string{}
	args = append(args, buildProxyCommandOption(kind, namespace, name, options))
	args = append(args, buildSSHTarget(kind, namespace, name, options))

	cmd := exec.Command("ssh", args...)
	fmt.Println("running:", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func buildProxyCommandOption(kind, namespace, name string, options *SSHOptions) string {
	proxyCommand := strings.Builder{}
	proxyCommand.WriteString("-o ProxyCommand=")
	proxyCommand.WriteString(os.Args[0])
	proxyCommand.WriteString(" port-forward --stdio=true ")
	proxyCommand.WriteString(fmt.Sprintf("%s/%s.%s", kind, name, namespace))
	proxyCommand.WriteString(" ")

	proxyCommand.WriteString(strconv.Itoa(options.SshPort))

	return proxyCommand.String()
}

func buildSSHTarget(kind, namespace, name string, options *SSHOptions) string {
	target := strings.Builder{}
	if len(options.SshUsername) > 0 {
		target.WriteString(options.SshUsername)
		target.WriteRune('@')
	}
	target.WriteString(kind)
	target.WriteRune('/')
	target.WriteString(name)
	target.WriteRune('.')
	target.WriteString(namespace)
	return target.String()
}
