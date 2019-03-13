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
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var timeout int

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "console (VMI)",
		Short:   "Connect to a console of a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Console{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 5, "The number of minutes to wait for the virtual machine instance to be ready.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type Console struct {
	clientConfig clientcmd.ClientConfig
}

func usage() string {
	usage := `  # Connect to the console on VirtualMachineInstance 'myvmi':
  virtctl console myvmi
  # Configure one minute timeout (default 5 minutes)
  virtctl console --timeout=1 myvmi`

	return usage
}

func (c *Console) Run(cmd *cobra.Command, args []string) error {
	namespace, _, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}

	vmi := args[0]

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(c.clientConfig)
	if err != nil {
		return err
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// in -> stdinWriter | stdinReader -> console
	// out <- stdoutReader | stdoutWriter <- console

	resChan := make(chan error)
	stopChan := make(chan struct{}, 1)
	writeStop := make(chan error)
	readStop := make(chan error)

	// Wait until the virtual machine is in running phase, user interrupt or timeout
	runningChan := make(chan error)
	waitInterrupt := make(chan os.Signal, 1)
	signal.Notify(waitInterrupt, os.Interrupt)

	go func() {
		con, err := virtCli.VirtualMachineInstance(namespace).SerialConsole(vmi, time.Duration(timeout)*time.Minute)
		runningChan <- err

		if err != nil {
			return
		}

		resChan <- con.Stream(kubecli.StreamOptions{
			In:  stdinReader,
			Out: stdoutWriter,
		})
	}()

	select {
	case <-waitInterrupt:
		// Make a new line in the terminal
		fmt.Println()
		return nil
	case err = <-runningChan:
		if err != nil {
			return err
		}
	}

	state, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("Make raw terminal failed: %s", err)
	}
	fmt.Fprint(os.Stderr, "Successfully connected to ", vmi, " console. The escape sequence is ^]\n")

	in := os.Stdin
	out := os.Stdout

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
