package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func runLocalCommandClient(kind, namespace, name string) error {
	args := []string{}
	args = append(args, buildProxyCommandOption(kind, namespace, name))
	args = append(args, buildSSHTarget(kind, namespace, name))

	cmd := exec.Command("ssh", args...)
	fmt.Println("running:", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func buildProxyCommandOption(kind, namespace, name string) string {
	proxyCommand := strings.Builder{}
	proxyCommand.WriteString("-o ProxyCommand=")
	proxyCommand.WriteString(os.Args[0])
	proxyCommand.WriteString(" port-forward --stdio=true ")
	proxyCommand.WriteString(fmt.Sprintf("%s/%s.%s", kind, name, namespace))
	proxyCommand.WriteString(" ")

	proxyCommand.WriteString(strconv.Itoa(sshPort))

	return proxyCommand.String()
}

func buildSSHTarget(kind, namespace, name string) string {
	target := strings.Builder{}
	if len(sshUsername) > 0 {
		target.WriteString(sshUsername)
		target.WriteRune('@')
	}
	target.WriteString(kind)
	target.WriteRune('/')
	target.WriteString(name)
	target.WriteRune('.')
	target.WriteString(namespace)
	return target.String()
}
