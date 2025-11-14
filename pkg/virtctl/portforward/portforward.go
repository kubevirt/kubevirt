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
 * Copyright The KubeVirt Authors.
 *
 */

package portforward

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	forwardToStdioFlag = "stdio"
	addressFlag        = "address"

	vm  = "vm"
	vmi = "vmi"
)

var (
	forwardToStdio bool
	address        string = "127.0.0.1"
)

func NewCommand() *cobra.Command {
	log.InitializeLogging("portforward")
	c := PortForward{}
	cmd := &cobra.Command{
		Use:     "port-forward type/name[/namespace] [protocol/]localPort[:targetPort]...",
		Short:   "Forward local ports to a virtualmachine or virtualmachineinstance.",
		Long:    usage(),
		Example: examples(),
		Args: func(cmd *cobra.Command, args []string) error {
			if n := len(args); n < 2 {
				log.Log.Errorf("fatal: Number of input parameters is incorrect, portforward requires at least 2 arg(s), received %d", n)
				// always write to stderr on failures to ensure they get printed in stdio mode
				cmd.SetOut(os.Stderr)
				cmd.Help()
				return errors.New("argument validation failed")
			}
			return nil
		},
		RunE: c.Run,
	}
	cmd.Flags().BoolVar(&forwardToStdio, forwardToStdioFlag, forwardToStdio,
		fmt.Sprintf("--%s=true: Set this to true to forward the tunnel to stdout/stdin; Only works with a single port", forwardToStdioFlag))
	cmd.Flags().StringVar(&address, addressFlag, address,
		fmt.Sprintf("--%s=: Set this to the address the local ports should be opened on", addressFlag))
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type PortForward struct {
	address  *net.IPAddr
	resource portforwardableResource
}

func (o *PortForward) Run(cmd *cobra.Command, args []string) error {
	setOutput(cmd)

	client, _, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	kind, namespace, name, ports, err := o.prepareCommand(args, namespace)
	if err != nil {
		return err
	}
	if err := o.setResource(kind, namespace, client); err != nil {
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

func (o *PortForward) prepareCommand(args []string, fallbackNamespace string) (kind string, namespace string, name string, ports []forwardedPort, err error) {
	kind, namespace, name, err = ParseTarget(args[0])
	if err != nil {
		return
	}

	ports, err = parsePorts(args[1:])
	if err != nil {
		return
	}

	if len(namespace) < 1 {
		namespace = fallbackNamespace
	}

	return
}

func (o *PortForward) setResource(kind, namespace string, client kubecli.KubevirtClient) error {
	if kind == vmi {
		o.resource = client.VirtualMachineInstance(namespace)
	} else if kind == vm {
		o.resource = client.VirtualMachine(namespace)
	} else {
		return errors.New("unsupported resource type " + kind)
	}

	return nil
}

func (o *PortForward) startStdoutStream(namespace, name string, port forwardedPort) error {
	streamer, err := o.resource.PortForward(name, port.remote, port.protocol)
	if err != nil {
		return err
	}

	log.Log.V(3).Infof("forwarding to %s/%s:%d", namespace, name, port.remote)
	if err := streamer.Stream(kvcorev1.StreamOptions{
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
	
The target argument supports the syntax target/name[/namespace] with /namespace as optional field.
Kind accepts any of vmi, vmis, vm, vms, virtualmachineinstance, virtualmachine, virtualmachineinstances, virtualmachines.

A dot in target without specifying a namespace activates legacy parsing of name and namespace using
the old type/name.namespace syntax. This is to avoid breaking existing scripts using the old syntax.
A warning will be emitted instead. This behavior will be removed in the next release.

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
  {{ProgramName}} port-forward vmi/testvmi/mynamespace udp/8080:9090

  # Forward the local port 8080 to the vm port
  {{ProgramName}} port-forward vm/testvm 8080

  # Forward the local port 8080 to the vm port in mynamespace
  {{ProgramName}} port-forward vm/testvm/mynamespace 8080

  # Note: {{ProgramName}} port-forward sends all traffic over the Kubernetes API Server. 
  # This means any traffic will add additional pressure to the control plane.
  # For continous traffic intensive connections, consider using a dedicated Kubernetes Service.`
}

// ParseTarget argument supporting the form of type/name[/namespace]
func ParseTarget(target string) (kind string, namespace string, name string, err error) {
	if target == "" {
		return "", "", "", errors.New("target cannot be empty")
	}

	parts := strings.Split(target, "/")
	switch len(parts) {
	case 1:
		return "", "", "", errors.New("target must contain type and name separated by '/'")
	case 2:
		kind = parts[0]
		name = parts[1]
	case 3:
		kind = parts[0]
		name = parts[1]
		namespace = parts[2]
		if namespace == "" {
			return "", "", "", errors.New("namespace cannot be empty")
		}
	default:
		return "", "", "", errors.New("target is not valid with more than two '/'")
	}

	kind, err = normalizeKind(kind)
	if err != nil {
		return "", "", "", err
	}

	if name == "" {
		return "", "", "", errors.New("name cannot be empty")
	}

	return kind, namespace, name, nil
}

func normalizeKind(kind string) (string, error) {
	switch strings.ToLower(kind) {
	case vm, "vms", "virtualmachine", "virtualmachines":
		return vm, nil
	case vmi, "vmis", "virtualmachineinstance", "virtualmachineinstances":
		return vmi, nil
	}
	return "", fmt.Errorf("unsupported resource type '%s'", kind)
}
