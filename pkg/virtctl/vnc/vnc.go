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

package vnc

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const FLAG = "vnc"

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vnc (vm)",
		Short:   "Open a vnc connection to a virtual machine.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := VNC{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type VNC struct {
	clientConfig clientcmd.ClientConfig
}

func (o *VNC) Run(cmd *cobra.Command, args []string) error {
	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	vm := args[0]

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return err
	}

	//                                       -> pipeInWriter  -> pipeInReader
	// remote-viewer -> unix sock connection
	//                                       <- pipeOutReader <- pipeOutWriter
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()

	k8ResChan := make(chan error)
	viewResChan := make(chan error)
	stopChan := make(chan struct{}, 1)
	writeStop := make(chan error)
	readStop := make(chan error)

	// The local tcp server is used to proxy the podExec websock connection to remote-viewer
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("Can't listen on unix socket: %s", err.Error())
	}

	port := ln.Addr().(*net.TCPAddr).Port

	// setup connection with VM
	go func() {
		err := virtCli.VM(namespace).VNC(vm, pipeInReader, pipeOutWriter)
		k8ResChan <- err
	}()

	// execute remote viewer
	go func() {
		cmnd := exec.Command("remote-viewer", fmt.Sprintf("vnc://127.0.0.1:%d", port))
		err := cmnd.Run()
		if err != nil {
			glog.Error(err)
		}
		viewResChan <- err
	}()

	// wait for remote-viewer to connect to our local proxy server
	fd, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("Failed to accept unix sock connection. %s", err.Error())
	}
	defer fd.Close()

	glog.V(2).Infof("remote-viewer connected")
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		close(stopChan)
	}()

	// write to FD <- pipeOutReader
	go func() {
		_, err := io.Copy(fd, pipeOutReader)
		readStop <- err
	}()

	// read from FD -> pipeInWriter
	go func() {
		_, err := io.Copy(pipeInWriter, fd)
		writeStop <- err
	}()

	select {
	case <-stopChan:
	case err = <-readStop:
	case err = <-writeStop:
	case err = <-k8ResChan:
	case err = <-viewResChan:
	}

	if err != nil {
		return fmt.Errorf("Error encountered: %s", err.Error())
	}
	return nil
}

func usage() string {
	usage := "# Connect to testvm via remote-viewer:\n"
	usage += "./virtctl vnc testvm"
	return usage
}
