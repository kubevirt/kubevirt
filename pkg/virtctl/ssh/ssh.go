/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package ssh

import (
	"os"

	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	"fmt"
	"io"

	"golang.org/x/crypto/ssh"

	"os/user"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ssh (vm)",
		Short:   "Connect via ssh to a virtual machine, proxied via websockets.",
		Example: usage(),
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Console{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}

	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().BoolP("stdin", "i", false, "Pass stdin to the container")
	cmd.Flags().BoolP("tty", "t", false, "Stdin is a TTY")
	return cmd
}

type Console struct {
	clientConfig clientcmd.ClientConfig
}

func usage() string {
	usage := "# Connect via ssh to the virtual machine myvm:\n"
	usage += "virtctl ssh user@myvm"
	return usage
}

func (c *Console) Run(cmd *cobra.Command, args []string) error {
	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}

	split := strings.Split(args[0], "@")
	user.Current()
	username := ""
	vm := ""

	if len(split) == 2 {
		username = split[0]
		vm = split[1]
	} else if len(split) == 1 {
		vm = split[0]
		user, err := user.Current()
		if err != nil {
			return err
		}
		username = user.Name
	}

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	defer close(done)
	conn, err := virtCli.VM(namespace).SSH(vm, done)
	if err != nil {
		return err
	}

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(func() (string, error) {
				fmt.Printf("%s`s password: ", args[0])
				bytePwd, err := terminal.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return "", err
				}
				fmt.Println("")
				return string(bytePwd), err
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshCon, chans, reqs, err := ssh.NewClientConn(conn.UnderlyingConn(), "whatever", sshConfig)
	if err != nil {
		return err
	}
	cli := ssh.NewClient(sshCon, chans, reqs)

	session, err := cli.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %s", err)
	}
	defer session.Close()

	if tty, _ := cmd.Flags().GetBool("tty"); tty {
		modes := ssh.TerminalModes{
			ssh.ECHO:          0,     // disable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
			return fmt.Errorf("request for pseudo terminal failed: %s", err)
		}
	}

	errChan := make(chan error)

	if interactive, _ := cmd.Flags().GetBool("stdin"); interactive {

		stdin, err := session.StdinPipe()
		if err != nil {
			return fmt.Errorf("Unable to setup stdin for session: %v", err)
		}
		go func() {
			_, err := io.Copy(stdin, os.Stdin)
			errChan <- err
		}()
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for session: %v", err)
	}
	go func() {
		_, err := io.Copy(os.Stdout, stdout)
		errChan <- err
	}()

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stderr for session: %v", err)
	}
	go func() {
		_, err := io.Copy(os.Stderr, stderr)
		errChan <- err
	}()

	go func() {
		errChan <- session.Run(strings.Join(args[1:], " "))
	}()
	err = <-errChan
	fmt.Println()

	return err
}
