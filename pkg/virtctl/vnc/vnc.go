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

package vnc

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"

	flag "github.com/spf13/pflag"
	kubev1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

const FLAG = "vnc"

type Vnc struct{}

func (o *Vnc) Run(flags *flag.FlagSet) int {
	server, _ := flags.GetString("server")
	kubeconfig, _ := flags.GetString("kubeconfig")
	namespace, _ := flags.GetString("namespace")
	if namespace == "" {
		namespace = kubev1.NamespaceDefault
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

	//                                       -> pipeInWriter  -> pipeInReader
	// remote-viewer -> unix sock connection
	//                                       <- pipeOutReader <- pipeOutWriter
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()

	k8ResChan := make(chan error)
	viewResChan := make(chan error)
	stopChan := make(chan struct{}, 1)
	writeStop := make(chan struct{})
	readStop := make(chan struct{})

	// The local tcp server is used to proxy the podExec websock connection to remote-viewer
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("Can't listen on unix socket: %s", err.Error())
		return 1
	}

	port := ln.Addr().(*net.TCPAddr).Port

	// setup connection with VM
	go func() {
		err := virtCli.VM(namespace).Vnc(vm, pipeInReader, pipeOutWriter)
		k8ResChan <- err
	}()

	// execute remote viewer
	go func() {
		cmnd := exec.Command("remote-viewer", fmt.Sprintf("vnc://127.0.0.1:%d", port))
		err := cmnd.Run()
		if err != nil {
			log.Println(err)
		}
		viewResChan <- err
	}()

	// wait for remote-viewer to connect to our local proxy server
	fd, err := ln.Accept()
	if err != nil {
		log.Printf("Failed to accept unix sock connection. %s", err.Error())
		return 1
	}
	defer fd.Close()

	log.Printf("remote-viewer connected")
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		close(stopChan)
	}()

	// write to FD <- pipeOutReader
	go func() {
		defer close(readStop)
		buf := make([]byte, 4096, 4096)
		for {
			// reading qemu vnc server
			n, err := pipeOutReader.Read(buf)
			if err != nil && err != io.EOF {
				return
			}
			if n == 0 && err == io.EOF {
				return
			}

			// Writing to remote viewer
			_, err = fd.Write(buf[0:n])
			if err == io.EOF {
				return
			}
		}
	}()

	// read from FD -> pipeInWriter
	go func() {
		defer close(writeStop)
		buf := make([]byte, 4096, 4096)
		for {
			// reading from remoteViewer
			n, err := fd.Read(buf)
			if err != nil && err != io.EOF {
				return
			}
			if n == 0 && err == io.EOF {
				return
			}

			// Writing out to the qemu vnc server
			_, err = pipeInWriter.Write(buf[0:n])
			if err == io.EOF {
				return
			}
		}
	}()

	select {
	case <-stopChan:
	case <-readStop:
	case <-writeStop:
	case err = <-k8ResChan:
	case err = <-viewResChan:
	}

	if err != nil {
		log.Printf("Error encountered: %s", err.Error())
		return 1
	}
	return 0
}

func (o *Vnc) Usage() string {
	usage := "virtctl can connect via remote-viewer to a VM\n\n"
	usage += "Examples:\n"
	usage += "# Connect to testvm via remote-viewer\n"
	usage += "./virtctl vnc testvm\n\n"
	return usage
}
func (o *Vnc) FlagSet() *flag.FlagSet {
	return flag.NewFlagSet(FLAG, flag.ExitOnError)
}
