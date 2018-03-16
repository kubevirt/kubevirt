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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package console

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

type Console struct {
}

func (c *Console) FlagSet() *flag.FlagSet {
	cf := flag.NewFlagSet("console", flag.ExitOnError)
	cf.StringP("device", "d", "serial0", "Console to connect to")

	return cf
}

func (c *Console) Usage() string {
	usage := "Connect to a serial console on a VM:\n\n"
	usage += "Examples:\n"
	usage += "# Connect to the console on VM 'myvm':\n"
	usage += "virtctl console myvm\n\n"
	usage += "Options:\n"
	usage += c.FlagSet().FlagUsages()
	return usage
}

func (c *Console) Run(flags *flag.FlagSet) int {

	server, _ := flags.GetString("server")
	kubeconfig, _ := flags.GetString("kubeconfig")
	namespace, _ := flags.GetString("namespace")
	device := "serial0"
	if namespace == "" {
		namespace = v1.NamespaceDefault
	}
	if len(flags.Args()) != 2 {
		log.Println("VM name is missing")
		return 1
	}
	vm := flags.Arg(1)

	virtCli, err := kubecli.GetKubevirtClientFromFlags(server, kubeconfig)
	if err != nil {
		log.Println(err)
		return 1
	}

	state, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Printf("Make raw terminal failed: %s", err)
		return 1
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
		err := virtCli.VM(namespace).SerialConsole(vm, device, stdinReader, stdoutWriter)
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
		log.Println(err)
		return 1
	}
	return 0
}
