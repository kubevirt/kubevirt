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
	"golang.org/x/term"

	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

type consoleCommand struct {
	timeout int
}

func NewCommand() *cobra.Command {
	c := consoleCommand{}
	cmd := &cobra.Command{
		Use:     "console (VMI)",
		Short:   "Connect to a console of a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE:    c.run,
	}
	cmd.Flags().IntVar(&c.timeout, "timeout", 5, "The number of minutes to wait for the virtual machine instance to be ready.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	usage := `  # Connect to the console on VirtualMachineInstance 'myvmi':
  {{ProgramName}} console myvmi
  # Configure one minute timeout (default 5 minutes)
  {{ProgramName}} console --timeout=1 myvmi`

	return usage
}

func (c *consoleCommand) run(cmd *cobra.Command, args []string) error {
	vmi := args[0]

	client, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	return c.handleConsoleConnection(client, namespace, vmi)
}

func (c *consoleCommand) handleConsoleConnection(client kubecli.KubevirtClient, namespace, vmi string) error {
	// in -> stdinWriter | stdinReader -> console
	// out <- stdoutReader | stdoutWriter <- console
	// Wait until the virtual machine is in running phase, user interrupt or timeout
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	resChan := make(chan error)
	runningChan := make(chan error)
	waitInterrupt := make(chan os.Signal, 1)
	signal.Notify(waitInterrupt, os.Interrupt)

	go func() {
		con, err := client.VirtualMachineInstance(namespace).SerialConsole(vmi, &kvcorev1.SerialConsoleOptions{ConnectionTimeout: time.Duration(c.timeout) * time.Minute})
		runningChan <- err

		if err != nil {
			return
		}

		resChan <- con.Stream(kvcorev1.StreamOptions{
			In:  stdinReader,
			Out: stdoutWriter,
		})
	}()

	select {
	case <-waitInterrupt:
		// Make a new line in the terminal
		fmt.Println()
		return nil
	case err := <-runningChan:
		if err != nil {
			return err
		}
	}
	err := Attach(stdinReader, stdoutReader, stdinWriter, stdoutWriter,
		fmt.Sprintf("Successfully connected to %s console. Press Ctrl+] or Ctrl+5 to exit console.\n", vmi),
		resChan)
	if err != nil {
		if e, ok := err.(*websocket.CloseError); ok && e.Code == websocket.CloseAbnormalClosure {
			fmt.Fprint(os.Stderr, "\n"+
				"You were disconnected from the console. This could be caused by one of the following:"+
				"\n - the target VM was powered off"+
				"\n - another user connected to the console of the target VM"+
				"\n - network issues\n")
		}
		return err
	}
	return nil
}

// Attach attaches stdin and stdout to the console
// in -> stdinWriter | stdinReader -> console
// out <- stdoutReader | stdoutWriter <- console
func Attach(stdinReader, stdoutReader *io.PipeReader, stdinWriter, stdoutWriter *io.PipeWriter, message string, resChan <-chan error) (err error) {
	stopChan := make(chan struct{}, 1)
	writeStop := make(chan error)
	readStop := make(chan error)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		state, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("Make raw terminal failed: %s", err)
		}
		defer term.Restore(int(os.Stdin.Fd()), state)
	}
	fmt.Fprint(os.Stderr, message)

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

	return err
}
