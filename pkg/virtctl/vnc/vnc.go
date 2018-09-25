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
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const LISTEN_TIMEOUT = 60 * time.Second

const FLAG = "vnc"

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vnc (VMI)",
		Short:   "Open a vnc connection to a virtual machine instance.",
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

	vmi := args[0]

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return err
	}

	// setup connection with VM
	vnc, err := virtCli.VirtualMachineInstance(namespace).VNC(vmi)
	if err != nil {
		return fmt.Errorf("Can't access VMI %s: %s", vmi, err.Error())
	}

	lnAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("Can't resolve the address: %s", err.Error())
	}

	// The local tcp server is used to proxy the podExec websock connection to remote-viewer
	ln, err := net.ListenTCP("tcp", lnAddr)
	if err != nil {
		return fmt.Errorf("Can't listen on unix socket: %s", err.Error())
	}
	// End of pre-flight checks. Everything looks good, we can start
	// the goroutines and let the data flow

	//                                       -> pipeInWriter  -> pipeInReader
	// remote-viewer -> unix sock connection
	//                                       <- pipeOutReader <- pipeOutWriter
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()

	k8ResChan := make(chan error)
	listenResChan := make(chan error)
	viewResChan := make(chan error)
	stopChan := make(chan struct{}, 1)
	doneChan := make(chan struct{}, 1)
	writeStop := make(chan error)
	readStop := make(chan error)

	go func() {
		// transfer data from/to the VM
		k8ResChan <- vnc.Stream(kubecli.StreamOptions{
			In:  pipeInReader,
			Out: pipeOutWriter,
		})
	}()

	// wait for remote-viewer to connect to our local proxy server
	go func() {
		start := time.Now()
		glog.Infof("connection timeout: %v", LISTEN_TIMEOUT)
		// exit early if spawning remote-viewer fails
		ln.SetDeadline(time.Now().Add(LISTEN_TIMEOUT))

		fd, err := ln.Accept()
		if err != nil {
			glog.V(2).Infof("Failed to accept unix sock connection. %s", err.Error())
			listenResChan <- err
		}
		defer fd.Close()

		glog.V(2).Infof("remote-viewer connected in %v", time.Now().Sub(start))

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

		// don't terminate until remote-viewer is done
		<-doneChan
		listenResChan <- err
	}()

	// execute remote viewer
	go func() {
		port := ln.Addr().(*net.TCPAddr).Port
		args := []string{fmt.Sprintf("vnc://127.0.0.1:%d", port)}
		if glog.V(4) {
			args = append(args, "--debug")
			glog.Infof("remote-viewer commandline: %v", args)
		}

		cmnd := exec.Command("remote-viewer", args...)

		output, err := cmnd.CombinedOutput()
		if err != nil {
			glog.Errorf("remote-viewer execution failed: %v, output: %v", err, string(output))
		} else {
			glog.V(2).Infof("remote-viewer output: %v", string(output))
		}
		viewResChan <- err
		close(doneChan)
	}()

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		close(stopChan)
	}()

	select {
	case <-stopChan:
	case err = <-readStop:
	case err = <-writeStop:
	case err = <-k8ResChan:
	case err = <-viewResChan:
	case err = <-listenResChan:
	}

	if err != nil {
		return fmt.Errorf("Error encountered: %s", err.Error())
	}
	return nil
}

func usage() string {
	usage := "  # Connect to 'testvmi' via remote-viewer:\n"
	usage += "  virtctl vnc testvmi"
	return usage
}
