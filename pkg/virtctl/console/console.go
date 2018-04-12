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

package console

import (
	"fmt"
	"io"
	"os"
	"os/signal"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/cobra"

	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "console (vm)",
		Short:   "Connect to a console of a virtual machine.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Console{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type Console struct {
	clientConfig clientcmd.ClientConfig
}

func usage() string {
	usage := "# Connect to the console on VM 'myvm':\n"
	usage += "virtctl console myvm"
	return usage
}

func (c *Console) Run(cmd *cobra.Command, args []string) error {
	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}

	vm := args[0]

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return err
	}

	state, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("Make raw terminal failed: %s", err)
	}
	fmt.Fprint(os.Stderr, "Escape sequence is ^]\n")

	in := os.Stdin
	out := os.Stdout

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// in -> stdinWriter | stdinReader -> console
	// out <- stdoutReader | stdoutWriter <- console

	resChan := make(chan error)
	stopChan := make(chan struct{}, 1)
	writeStop := make(chan error)
	readStop := make(chan error)

	go func() {
		err := virtCli.VM(namespace).SerialConsole(vm, stdinReader, stdoutWriter)
		resChan <- err
	}()

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		close(stopChan)
	}()

	go func() {
		_, err := io.Copy(out, stdoutReader)
		readStop <- err
	}()

	go func() {
		defer close(writeStop)
		buf := make([]byte, 1024, 1024)
		for {
			// reading from stdin
			n, err := in.Read(buf)
			if err != nil && err != io.EOF {
				writeStop <- err
				return
			}
			if n == 0 && err == io.EOF {
				return
			}

			// the escape sequence
			if buf[0] == 29 {
				return
			}
			// Writing out to the console connection
			_, err = stdinWriter.Write(buf[0:n])
			if err == io.EOF {
				return
			}
		}
	}()

	select {
	case <-stopChan:
	case err = <-readStop:
	case err = <-writeStop:
	case err = <-resChan:
	}

	terminal.Restore(int(os.Stdin.Fd()), state)

	if err != nil {
		return err
	}
	return nil
}
