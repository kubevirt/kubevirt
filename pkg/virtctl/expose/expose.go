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
* Copyright 2018 Red Hat, Inc.
*
 */

package expose

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_EXPOSE = "expose"
)

type Command struct {
	clientConfig clientcmd.ClientConfig
	command      string
}

// holding flag information
var ServiceName string
var ClusterIP string
var ExternalIP string
var LoadBalancerIP string
var Port int32
var NodePort int32
var Protocol string
var TargetPort int
var ServiceType string
var PortName string

// generate a new "expose" command
func NewExposeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "expose (vm)",
		Short:   "Expose a virtual machine as a new service.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_EXPOSE, clientConfig: clientConfig}
			return c.RunE(cmd, args)
		},
	}

	// flags for the "expose" command
	cmd.Flags().StringVar(&ServiceName, "name", "", "Name of the service created for the exposure of the VM")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&ClusterIP, "cluster-ip", "", "ClusterIP to be assigned to the service. Leave empty to auto-allocate, or set to 'None' to create a headless service.")
	cmd.Flags().StringVar(&ExternalIP, "external-ip", "", "Additional external IP address (not managed by the cluster) to accept for the service. If this IP is routed to a node, the service can be accessed by this IP in addition to its generated service IP. Optional.")
	cmd.Flags().StringVar(&LoadBalancerIP, "load-balancer-ip", "", "IP to assign to the Load Balancer. If empty, an ephemeral IP will be created and used.")
	cmd.Flags().Int32Var(&Port, "port", 0, "The port that the service should serve on")
	cmd.MarkFlagRequired("port")
	cmd.Flags().StringVar(&Protocol, "protocol", "TCP", "The network protocol for the service to be created.")
	cmd.Flags().IntVar(&TargetPort, "target-port", 0, "Name or number for the port on the VM that the service should direct traffic to. Optional.")
	cmd.Flags().Int32Var(&NodePort, "node-port", 0, "Port used to expose the service on each node in a cluster.")
	cmd.Flags().StringVar(&ServiceType, "type", "ClusterIP", "Type for this service: ClusterIP, NodePort, or LoadBalancer.")
	cmd.Flags().StringVar(&PortName, "port-name", "", "Name of the port. Optional.")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	usage := "  # Expose SSH to a virtual machine called 'myvm' as a node port (5555) of the cluster:\n"
	usage += fmt.Sprintf("  virtctl expose myvm --port=5555 --target-port=22 --name=myvm-ssh --type=NodePort")
	return usage
}

// executing the "expose" command
func (o *Command) RunE(cmd *cobra.Command, args []string) error {
	// first argument must be name of the VM
	vmName := args[0]

	// these are used to convert the flag values into service spec values
	var protocol v1.Protocol
	var targetPort intstr.IntOrString
	var serviceType v1.ServiceType

	// convert from integer to the IntOrString type
	targetPort = intstr.FromInt(TargetPort)

	// convert from string to the protocol enum
	switch Protocol {
	case "TCP":
		protocol = v1.ProtocolTCP
	case "UDP":
		protocol = v1.ProtocolUDP
	default:
		return fmt.Errorf("Unknown protocol: %s", Protocol)
	}

	// convert from string to the service type enum
	switch ServiceType {
	case "ClusterIP":
		serviceType = v1.ServiceTypeClusterIP
	case "NodePort":
		serviceType = v1.ServiceTypeNodePort
	case "LoadBalancer":
		serviceType = v1.ServiceTypeLoadBalancer
	case "ExternalName":
		return fmt.Errorf("Type: %s not supported", ServiceType)
	default:
		return fmt.Errorf("Unknown service type: %s", ServiceType)
	}

	// get the namespace
	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	// get the client
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	// TODO: not sure what are the options used for?
	options := k8smetav1.GetOptions{}
	// get the VM
	_, vm_err := virtClient.VM(namespace).Get(vmName, options)
	if vm_err != nil {
		// try to get offline VM this is only for better error message
		_, ovm_err := virtClient.OfflineVirtualMachine(namespace).Get(vmName, &options)
		if ovm_err != nil {
			return fmt.Errorf("Error fetching VirtualMachine: %v", vm_err)
		}
		return fmt.Errorf("OfflineVirtualMachine: %s must be started before being exposed", vmName)
	}

	// actually create the service
	service := &v1.Service{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: ServiceName,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: PortName, Protocol: protocol, Port: Port, TargetPort: targetPort, NodePort: NodePort},
			},
			Selector:       map[string]string{"kubevirt.io/domain": vmName},
			ClusterIP:      ClusterIP,
			Type:           serviceType,
			LoadBalancerIP: LoadBalancerIP,
		},
	}

	// set external IP if provided
	if len(ExternalIP) > 0 {
		service.Spec.ExternalIPs = []string{ExternalIP}
	}

	// try to create the service on the cluster
	_, err = virtClient.CoreV1().Services(namespace).Create(service)
	if err != nil {
		return fmt.Errorf("Service cretion failed: %v", err)
	} else {
		fmt.Printf("Service %s successfully exposed for VirtualMachine %s\n", ServiceName, vmName)

	}

	return nil
}
