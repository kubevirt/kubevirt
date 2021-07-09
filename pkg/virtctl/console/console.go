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
 * Copyright 2017 - 2019 Red Hat, Inc.
 *
 */

package console

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
	"kubevirt.io/kubevirt/pkg/virtctl/utils"
)

var timeout int

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "console (VMI)",
		Short:   "Connect to a console of a virtual machine instance.",
		Example: usage(),
		Args:    templates.ExactArgs("console", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Console{clientConfig: clientConfig}
			return c.Run(args)
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
  {{ProgramName}} console myvmi
  # Configure one minute timeout (default 5 minutes)
  {{ProgramName}} console --timeout=1 myvmi`

	return usage
}

func (c *Console) Run(args []string) error {
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
	// Wait until the virtual machine is in running phase, user interrupt or timeout
	resChan := make(chan error)
	runningChan := make(chan error)
	waitInterrupt := make(chan os.Signal, 1)
	signal.Notify(waitInterrupt, os.Interrupt)

	go func() {
		con, err := virtCli.VirtualMachineInstance(namespace).SerialConsole(vmi, &kubecli.SerialConsoleOptions{ConnectionTimeout: time.Duration(timeout) * time.Minute})
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
	err = utils.AttachConsole(stdinReader, stdoutReader, stdinWriter, stdoutWriter,
		fmt.Sprint("Successfully connected to ", vmi, " console. The escape sequence is ^]\n"),
		resChan)

	if err != nil {
		if e, ok := err.(*websocket.CloseError); ok && e.Code == websocket.CloseAbnormalClosure {
			fmt.Fprint(os.Stderr, "\nYou were disconnected from the console. This has one of the following reasons:"+
				"\n - another user connected to the console of the target vm"+
				"\n - network issues\n")
		}
		return err
	}
	return nil
}
