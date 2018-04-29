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
	"strings"

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
var serviceName string
var clusterIP string
var externalIP string
var loadBalancerIP string
var port int32
var nodePort int32
var strProtocol string
var intTargetPort int
var strServiceType string
var portName string

// generate a new "expose" command
func NewExposeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "expose TYPE NAME",
		Short: "Expose a virtual machine as a new service.",
		Long: `Looks up a virtual machine, offline virtual machine or virtual machine replica set by name and use its selector as the selector for a new service on the specified port.
        A virtual machine replica set will be exposed as a service only if its selector is convertible to a selector that service supports, i.e. when the selector contains only the matchLabels component.
        Note that if no port is specified via --port and the exposed resource has multiple ports, all will be re-used by the new service. 
        Also if no labels are specified, the new service will re-use the labels from the resource it exposes.
        
        Possible types are (case insensitive, both single and plurant forms):
        
        virtualmachine (vm), offlinevirtualmachine (ovm), virtualmachinereplicaset (vmrs)`,
		Example: usage(),
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_EXPOSE, clientConfig: clientConfig}
			return c.RunE(cmd, args)
		},
	}

	// flags for the "expose" command
	cmd.Flags().StringVar(&serviceName, "name", "", "Name of the service created for the exposure of the VM")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&clusterIP, "cluster-ip", "", "ClusterIP to be assigned to the service. Leave empty to auto-allocate, or set to 'None' to create a headless service.")
	cmd.Flags().StringVar(&externalIP, "external-ip", "", "Additional external IP address (not managed by the cluster) to accept for the service. If this IP is routed to a node, the service can be accessed by this IP in addition to its generated service IP. Optional.")
	cmd.Flags().StringVar(&loadBalancerIP, "load-balancer-ip", "", "IP to assign to the Load Balancer. If empty, an ephemeral IP will be created and used.")
	cmd.Flags().Int32Var(&port, "port", 0, "The port that the service should serve on")
	cmd.MarkFlagRequired("port")
	cmd.Flags().StringVar(&strProtocol, "protocol", "TCP", "The network protocol for the service to be created.")
	cmd.Flags().IntVar(&intTargetPort, "target-port", 0, "Name or number for the port on the VM that the service should direct traffic to. Optional.")
	cmd.Flags().Int32Var(&nodePort, "node-port", 0, "Port used to expose the service on each node in a cluster.")
	cmd.Flags().StringVar(&strServiceType, "type", "ClusterIP", "Type for this service: ClusterIP, NodePort, or LoadBalancer.")
	cmd.Flags().StringVar(&portName, "port-name", "", "Name of the port. Optional.")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	usage := "  # Expose SSH to a virtual machine called 'myvm' as a node port (5555) of the cluster:\n"
	usage += fmt.Sprintf("  virtctl expose vm myvm --port=5555 --target-port=22 --name=myvm-ssh --type=NodePort")
	return usage
}

// executing the "expose" command
func (o *Command) RunE(cmd *cobra.Command, args []string) error {
	// first argument is type of VM: VM, offline VM or replica set VM
	vmType := strings.ToLower(args[0])
	// second argument must be name of the VM
	vmName := args[1]

	// these are used to convert the flag values into service spec values
	var protocol v1.Protocol
	var targetPort intstr.IntOrString
	var serviceType v1.ServiceType

	// convert from integer to the IntOrString type
	targetPort = intstr.FromInt(intTargetPort)

	// convert from string to the protocol enum
	switch strProtocol {
	case "TCP":
		protocol = v1.ProtocolTCP
	case "UDP":
		protocol = v1.ProtocolUDP
	default:
		return fmt.Errorf("Unknown protocol: %s", strProtocol)
	}

	// convert from string to the service type enum
	switch strServiceType {
	case "ClusterIP":
		serviceType = v1.ServiceTypeClusterIP
	case "NodePort":
		serviceType = v1.ServiceTypeNodePort
	case "LoadBalancer":
		serviceType = v1.ServiceTypeLoadBalancer
	case "ExternalName":
		return fmt.Errorf("Type: %s not supported", strServiceType)
	default:
		return fmt.Errorf("Unknown service type: %s", strServiceType)
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

	// does a plain quorum read from the apiserver
	options := k8smetav1.GetOptions{}

	switch vmType {
	case "vm", "vms", "virtualmachine", "virtualmachines":
		// get the VM
		_, err := virtClient.VM(namespace).Get(vmName, options)
		if err != nil {
			return fmt.Errorf("Error fetching VirtualMachine: %v", err)
		}
	case "ovm", "ovms", "offlinevirtualmachine", "offlinevirtualmachines":
		// get the offline VM
		_, err := virtClient.OfflineVirtualMachine(namespace).Get(vmName, &options)
		if err != nil {
			return fmt.Errorf("Error fetching OfflineVirtualMachine: %v", err)
		}
	case "vmrs", "vmrss", "virtualmachinereplicaset", "virtualmachinereplicasets":
		// get the VM replica set
		_, err := virtClient.ReplicaSet(namespace).Get(vmName, options)
		if err != nil {
			return fmt.Errorf("Error fetching VirtualMachine ReplicaSet: %v", err)
		}
		// in case of replica set we take the label from the replica set template
        // same as the original vmName
		//vmName = vmrs.Spec.Template.ObjectMeta.Labels["kubevirt.io/vmReplicaSet"]
	default:
		return fmt.Errorf("Unsupported resource type: %s", vmType)
	}

	// actually create the service
	service := &v1.Service{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: serviceName,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: portName, Protocol: protocol, Port: port, TargetPort: targetPort, NodePort: nodePort},
			},
			Selector:       map[string]string{"kubevirt.io/domain": vmName},
			ClusterIP:      clusterIP,
			Type:           serviceType,
			LoadBalancerIP: loadBalancerIP,
		},
	}

	// set external IP if provided
	if len(externalIP) > 0 {
		service.Spec.ExternalIPs = []string{externalIP}
	}

	// try to create the service on the cluster
	_, err = virtClient.CoreV1().Services(namespace).Create(service)
	if err != nil {
		return fmt.Errorf("Service cretion failed: %v", err)
	} else {
		fmt.Printf("Service %s successfully exposed for VirtualMachine %s\n", serviceName, vmName)

	}

	return nil
}
