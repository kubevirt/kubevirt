//go:build includenovnc

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

package novnc

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var listenAddress = "127.0.0.1"

var novncServer bool
var customPort = 0

var serveOnly bool

func init() {
	virtctl.CommandRegistrationCollback = append(virtctl.CommandRegistrationCollback, NewCommand)
}

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "novnc (VMI)",
		Short:   "Open a vnc connection to a virtual machine instance and access it via an embedded novnc instance",
		Example: usage(),
		Args:    templates.ExactArgs("novnc", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := NOVNC{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.Flags().StringVar(&listenAddress, "address", listenAddress, "--address=127.0.0.1: Setting this will change the listening address of the NOVNC server. Example: --address=0.0.0.0 will make the server listen on all interfaces.")
	cmd.Flags().IntVar(&customPort, "port", customPort,
		"--port=0: Assigning a port value to this will try to run the proxy on the given port if the port is accessible; If unassigned, the proxy will run on a random port")
	cmd.Flags().BoolVar(&serveOnly, "serve-only", serveOnly, "--serve-only=false: Setting this true will run only serve novnc on the specified address and port and print the connection url, but will not open a browser")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type NOVNC struct {
	clientConfig clientcmd.ClientConfig
}

func (o *NOVNC) Run(cmd *cobra.Command, args []string) error {
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

	stopChan := make(chan struct{}, 1)

	go func() {
		defer close(stopChan)
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
	}()

	go func() {
		defer close(stopChan)
		err = RunNOVNCWebserver(listenAddress, strconv.Itoa(customPort), serveOnly, vnc.AsConn())
		if err != nil {
			fmt.Println(err)
		}
	}()

	select {
	case <-stopChan:
	}

	if err != nil {
		return fmt.Errorf("Error encountered: %s", err.Error())
	}
	return nil
}

func usage() string {
	return `  # Connect to the testvmi and open the printed URL to connect via novnc:
   {{ProgramName}} vnc novnc --serve-only testvmi
   Open http://127.0.0.1:39985?autoconnect=true to connect to the virtual machine.

   # Connect to the testvmi and open a browser session automatically:
   {{ProgramName}} vnc novnc testvmi
`
}
