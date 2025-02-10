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
	"slices"
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
)

var (
	forwardToStdio bool
	address        string = "127.0.0.1"
)

func NewCommand() *cobra.Command {
	log.InitializeLogging("portforward")
	c := PortForward{}
	cmd := &cobra.Command{
		Use:     "port-forward [kind/]name[.namespace] [protocol/]localPort[:targetPort]...",
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
	address *net.IPAddr
	client  kubecli.KubevirtClient
}

func (o *PortForward) Run(cmd *cobra.Command, args []string) error {
	setOutput(cmd)

	client, namespace, changed, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	o.client = client

	namespace, name, ports, err := o.prepareCommand(args, namespace, changed)
	if err != nil {
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

	if err := o.startPortForwards(namespace, name, ports); err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	return nil
}

func (o *PortForward) prepareCommand(args []string, clientNamespace string, namespaceChanged bool) (string, string, []forwardedPort, error) {
	namespace, name, err := ParseTarget(args[0])
	if err != nil {
		return "", "", nil, err
	}

	ports, err := parsePorts(args[1:])
	if err != nil {
		return "", "", nil, err
	}

	if namespace == "" {
		namespace = clientNamespace
	} else if namespaceChanged {
		log.Log.Infof("Overriding target namespace '%s' with namespace '%s' from commandline", namespace, clientNamespace)
		namespace = clientNamespace
	}

	return namespace, name, ports, nil
}

func (o *PortForward) startStdoutStream(namespace, name string, port forwardedPort) error {
	streamer, err := o.client.VirtualMachineInstance(namespace).PortForward(name, port.remote, port.protocol)
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

func (o *PortForward) startPortForwards(namespace, name string, ports []forwardedPort) error {
	for _, port := range ports {
		forwarder := portForwarder{
			namespace: namespace,
			name:      name,
			client:    o.client,
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
  {{ProgramName}} port-forward testvmi 8080

  # Forward the local port 8080 to the vmi port 9090:
  {{ProgramName}} port-forward testvmi 8080:9090

  # Forward the local port 8080 to the vmi port 9090 as a UDP connection:
  {{ProgramName}} port-forward mynamespace/testvmi udp/8080:9090

  # Forward the local port 8080 to the vm port
  {{ProgramName}} port-forward testvm 8080

  # Forward the local port 8080 to the vm port in mynamespace
  {{ProgramName}} port-forward mynamespace/testvm 8080

  # Note: {{ProgramName}} port-forward sends all traffic over the Kubernetes API Server. 
  # This means any traffic will add additional pressure to the control plane.
  # For continous traffic intensive connections, consider using a dedicated Kubernetes Service.

  # Open an SSH connection using PortForward and ProxyCommand:
  ssh -o 'ProxyCommand={{ProgramName}} port-forward --stdio=true mynamespace/testvmi 22' user@mynamespace/testvmi

  # Use as SCP ProxyCommand:
  scp -o 'ProxyCommand={{ProgramName}} port-forward --stdio=true mynamespace/testvmi 22' local.file user@mynamespace/testvmi`
}

// ParseTarget argument supporting the form of $namespace/$name or vmi/$name.$namespace (legacy)
func ParseTarget(target string) (string, string, error) {
	namespace := ""
	name := target

	parts := strings.Split(name, "/")
	if len(parts) > 2 {
		return "", "", errors.New("target is not valid with more than one '/'")
	}
	if len(parts) == 2 {
		namespace = parts[0]
		name = parts[1]

		if namespace == "" {
			return "", "", errors.New("namespace cannot be empty")
		}
	}

	if name == "" {
		return "", "", errors.New("name cannot be empty or expected name after '/'")
	}

	if namespaceReserved(namespace) {
		log.Log.Warningf("Parsing target '%s' in legacy mode, support for this syntax will go away in a future release", target)
		return parseTargetLegacy(name)
	}

	return namespace, name, nil
}

func namespaceReserved(namespace string) bool {
	reserved := []string{
		"vmi", "vmis", "virtualmachineinstance", "virtualmachineinstances",
		"vm", "vms", "virtualmachine", "virtualmachines",
	}
	return slices.Contains(reserved, namespace)
}

func parseTargetLegacy(name string) (string, string, error) {
	if name[0] == '.' {
		return "", "", errors.New("expected name before '.'")
	}

	if name[len(name)-1] == '.' {
		return "", "", errors.New("expected namespace after '.'")
	}

	if lastDot := strings.LastIndex(name, "."); lastDot != -1 {
		return name[lastDot+1:], name[:lastDot], nil
	}

	return "", name, nil
}
