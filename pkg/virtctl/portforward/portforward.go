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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package portforward

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	forwardToStdioFlag = "stdio"
	addressFlag        = "address"
)

var (
	forwardToStdio bool
	address        string = "127.0.0.1"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "port-forward [kind/]name[.namespace] [protocol/]localPort[:targetPort]...",
		Short:   "Forward local ports to a virtualmachine or virtualmachineinstance.",
		Long:    usage(),
		Example: examples(),
		Args: func(cmd *cobra.Command, args []string) error {
			if n := len(args); n < 2 {
				glog.Errorf("fatal: Number of input parameters is incorrect, portforward requires at least 2 arg(s), received %d", n)
				// always write to stderr on failures to ensure they get printed in stdio mode
				cmd.SetOut(os.Stderr)
				cmd.Help()
				return errors.New("argument validation failed")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c := PortForward{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.Flags().BoolVar(&forwardToStdio, forwardToStdioFlag, forwardToStdio,
		fmt.Sprintf("--%s=true: Set this to true to forward the tunnel to stdout/stdin; Only works with a single port", forwardToStdioFlag))
	cmd.Flags().StringVar(&address, addressFlag, address,
		fmt.Sprintf("--%s=: Set this to the address the local ports should be opened on", addressFlag))
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type PortForward struct {
	address      *net.IPAddr
	clientConfig clientcmd.ClientConfig
	resource     portforwardableResource
}

func (o *PortForward) Run(cmd *cobra.Command, args []string) error {
	setOutput(cmd)
	kind, namespace, name, ports, err := o.prepareCommand(args)
	if err != nil {
		return err
	}

	if err := o.setResource(kind, namespace); err != nil {
		return err
	}

	if forwardToStdio {
		if len(ports) != 1 {
			return errors.New("only one port supported when forwarding to stdout")
		}
		return o.startStdoutStream(namespace, name, ports[0])
	}

	o.address, err = net.ResolveIPAddr("", address)
	if err != nil {
		return err
	}

	if err := o.startPortForwards(kind, namespace, name, ports); err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	return nil
}

func (o *PortForward) prepareCommand(args []string) (kind string, namespace string, name string, ports []forwardedPort, err error) {
	kind, namespace, name, err = templates.ParseTarget(args[0])
	if err != nil {
		return
	}

	ports, err = parsePorts(args[1:])
	if err != nil {
		return
	}

	if len(namespace) < 1 {
		namespace, _, err = o.clientConfig.Namespace()
		if err != nil {
			return
		}
	}

	return
}

func (o *PortForward) setResource(kind, namespace string) error {
	client, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return err
	}

	if templates.KindIsVMI(kind) {
		o.resource = client.VirtualMachineInstance(namespace)
	} else if templates.KindIsVM(kind) {
		o.resource = client.VirtualMachine(namespace)
	} else {
		return errors.New("unsupported resource kind " + kind)
	}

	return nil
}

func (o *PortForward) startStdoutStream(namespace, name string, port forwardedPort) error {
	streamer, err := o.resource.PortForward(name, port.remote, port.protocol)
	if err != nil {
		return err
	}

	glog.Infof("forwarding to %s/%s:%d", namespace, name, port.remote)
	if err := streamer.Stream(kubecli.StreamOptions{
		In:  os.Stdin,
		Out: os.Stdout,
	}); err != nil {
		return err
	}

	return nil
}

func (o *PortForward) startPortForwards(kind, namespace, name string, ports []forwardedPort) error {
	for _, port := range ports {
		forwarder := portForwarder{
			kind:      kind,
			namespace: namespace,
			name:      name,
			resource:  o.resource,
		}
		if err := forwarder.startForwarding(o.address, port); err != nil {
			return err
		}
	}
	return nil
}

// setOutput to stderr if we're using stdout for traffic
func setOutput(cmd *cobra.Command) {
	if forwardToStdio {
		cmd.SetOut(os.Stderr)
		cmd.Root().SetOut(os.Stderr)
	} else {
		cmd.SetOut(os.Stdout)
	}
}

func usage() string {
	return `Forward local ports to a virtualmachine or virtualmachineinstance.
	
The target argument supports the syntax kind/name.namespace with kind/ and .namespace as optional fields.
Kind accepts any of vmi (default), vmis, vm, vms, virtualmachineinstance, virtualmachine, virtualmachineinstances, virtualmachines.

The port argument supports the syntax protocol/localPort:targetPort with protocol/ and :targetPort as optional fields.
Protocol supports TCP (default) and UDP.

Portforwards get established over the Kubernetes control-plane using websocket streams.
Usage can be restricted by the cluster administrator through the /portforward subresource.
`
}

func examples() string {
	return `  # Forward the local port 8080 to the vmi port:
  {{ProgramName}} port-forward vmi/testvmi 8080

  # Forward the local port 8080 to the vmi port 9090:
  {{ProgramName}} port-forward vmi/testvmi 8080:9090

  # Forward the local port 8080 to the vmi port 9090 as a UDP connection:
  {{ProgramName}} port-forward vmi/testvmi.mynamespace udp/8080:9090

  # Forward the local port 8080 to the vm port
  {{ProgramName}} port-forward vm/testvm 8080

  # Forward the local port 8080 to the vm port in mynamespace
  {{ProgramName}} port-forward vm/testvm.mynamespace 8080

  # Note: {{ProgramName}} port-forward sends all traffic over the Kubernetes API Server. 
  # This means any traffic will add additional pressure to the control plane.
  # For continous traffic intensive connections, consider using a dedicated Kubernetes Service.

  # Open an SSH connection using PortForward and ProxyCommand:
  ssh -o 'ProxyCommand={{ProgramName}} port-forward --stdio=true testvmi.mynamespace 22' user@testvmi.mynamespace

  # Use as SCP ProxyCommand:
  scp -o 'ProxyCommand={{ProgramName}} port-forward --stdio=true testvmi.mynamespace 22' local.file user@testvmi.mynamespace`
}
