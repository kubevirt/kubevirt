package expose

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"

	v12 "kubevirt.io/kubevirt/pkg/api/v1"

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
var strProtocol string
var strTargetPort string
var strServiceType string
var portName string
var namespace string

// generate a new "expose" command
func NewExposeCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "expose (TYPE NAME)",
		Short: "Expose a virtual machine instance, virtual machine, or virtual machine instance replica set as a new service.",
		Long: `Looks up a virtual machine instance, virtual machine or virtual machine instance replica set by name and use its selector as the selector for a new service on the specified port.
A virtual machine instance replica set will be exposed as a service only if its selector is convertible to a selector that service supports, i.e. when the selector contains only the matchLabels component.
Note that if no port is specified via --port and the exposed resource has multiple ports, all will be re-used by the new service. 
Also if no labels are specified, the new service will re-use the labels from the resource it exposes.
        
Possible types are (case insensitive, both single and plurant forms):
        
virtualmachineinstance (vmi), virtualmachine (vm), virtualmachineinstancereplicaset (vmirs)`,
		Example: usage(),
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_EXPOSE, clientConfig: clientConfig}
			return c.RunE(cmd, args)
		},
	}

	// flags for the "expose" command
	cmd.Flags().StringVar(&serviceName, "name", "", "Name of the service created for the exposure of the VM.")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&clusterIP, "cluster-ip", "", "ClusterIP to be assigned to the service. Leave empty to auto-allocate, or set to 'None' to create a headless service.")
	cmd.Flags().StringVar(&externalIP, "external-ip", "", "Additional external IP address (not managed by the cluster) to accept for the service. If this IP is routed to a node, the service can be accessed by this IP in addition to its generated service IP. Optional.")
	cmd.Flags().StringVar(&loadBalancerIP, "load-balancer-ip", "", "IP to assign to the Load Balancer. If empty, an ephemeral IP will be created and used.")
	cmd.Flags().Int32Var(&port, "port", 0, "The port that the service should serve on.")
	cmd.Flags().StringVar(&strProtocol, "protocol", "TCP", "The network protocol for the service to be created.")
	cmd.Flags().StringVar(&strTargetPort, "target-port", "", "Name or number for the port on the VM that the service should direct traffic to. Optional.")
	cmd.Flags().StringVar(&strServiceType, "type", "ClusterIP", "Type for this service: ClusterIP, NodePort, or LoadBalancer.")
	cmd.Flags().StringVar(&portName, "port-name", "", "Name of the port. Optional.")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	usage := `  # Expose SSH to a virtual machine instance called 'myvm' on each node via a NodePort service:
  virtctl expose vmi myvm --port=22 --name=myvm-ssh --type=NodePort

  # Expose all defined pod-network ports of a virtual machine instance replicaset on a service:
  virtctl expose vmirs myvmirs --name=vmirs-service

  # Expose port 8080 as port 80 from a virtual machine instance replicaset on a service:
  virtctl expose vmirs myvmirs --port=80 --target-port=8080 --name=vmirs-service`
	return usage
}

// executing the "expose" command
func (o *Command) RunE(cmd *cobra.Command, args []string) error {
	// first argument is type of VM: VMI, VM or VMIRS
	vmType := strings.ToLower(args[0])
	// second argument must be name of the VM
	vmName := args[1]

	// these are used to convert the flag values into service spec values
	var protocol v1.Protocol
	var targetPort intstr.IntOrString
	var serviceType v1.ServiceType

	// convert from integer to the IntOrString type
	targetPort = intstr.Parse(strTargetPort)

	// convert from string to the protocol enum
	switch strProtocol {
	case "TCP":
		protocol = v1.ProtocolTCP
	case "UDP":
		protocol = v1.ProtocolUDP
	default:
		return fmt.Errorf("unknown protocol: %s", strProtocol)
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
		return fmt.Errorf("type: %s not supported", strServiceType)
	default:
		return fmt.Errorf("unknown service type: %s", strServiceType)
	}

	// get the namespace
	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	// get the client
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	// does a plain quorum read from the apiserver
	options := k8smetav1.GetOptions{}
	var serviceSelector map[string]string
	ports := []v1.ServicePort{}

	switch vmType {
	case "vmi", "vmis", "virtualmachineinstance", "virtualmachineinstances":
		// get the VM
		vmi, err := virtClient.VirtualMachineInstance(namespace).Get(vmName, &options)
		if err != nil {
			return fmt.Errorf("error fetching VirtualMachineInstance: %v", err)
		}
		serviceSelector = vmi.ObjectMeta.Labels
		ports = podNetworkPorts(&vmi.Spec)
		// remove unwanted labels
		delete(serviceSelector, "kubevirt.io/nodeName")
	case "vm", "vms", "virtualmachine", "virtualmachines":
		// get the VM
		vm, err := virtClient.VirtualMachine(namespace).Get(vmName, &options)
		if err != nil {
			return fmt.Errorf("error fetching Virtual Machine: %v", err)
		}
		if vm.Spec.Template != nil {
			ports = podNetworkPorts(&vm.Spec.Template.Spec)
		}
		serviceSelector = vm.Spec.Template.ObjectMeta.Labels
	case "vmirs", "vmirss", "virtualmachineinstancereplicaset", "virtualmachineinstancereplicasets":
		// get the VM replica set
		vmirs, err := virtClient.ReplicaSet(namespace).Get(vmName, options)
		if err != nil {
			return fmt.Errorf("error fetching VirtualMachineInstance ReplicaSet: %v", err)
		}
		if len(vmirs.Spec.Selector.MatchExpressions) > 0 {
			return fmt.Errorf("cannot expose VirtualMachineInstance ReplicaSet with match expressions")
		}
		if vmirs.Spec.Template != nil {
			ports = podNetworkPorts(&vmirs.Spec.Template.Spec)
		}
		serviceSelector = vmirs.Spec.Selector.MatchLabels
	default:
		return fmt.Errorf("unsupported resource type: %s", vmType)
	}

	if len(serviceSelector) == 0 {
		return fmt.Errorf("missing label information for %s: %s", vmType, vmName)
	}

	if port == 0 && len(ports) == 0 {
		return fmt.Errorf("couldn't find port via --port flag or introspection")
	} else if port != 0 {
		ports = []v1.ServicePort{{Name: portName, Protocol: protocol, Port: port, TargetPort: targetPort}}
	}

	// actually create the service
	service := &v1.Service{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports:          ports,
			Selector:       serviceSelector,
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
		return fmt.Errorf("service creation failed: %v", err)
	}
	fmt.Printf("Service %s successfully exposed for %s %s\n", serviceName, vmType, vmName)
	return nil
}

func podNetworkPorts(vmiSpec *v12.VirtualMachineInstanceSpec) []v1.ServicePort {
	podNetworkName := ""
	for _, network := range vmiSpec.Networks {
		if network.Pod != nil {
			podNetworkName = network.Name
			break
		}
	}
	if podNetworkName != "" {
		for _, device := range vmiSpec.Domain.Devices.Interfaces {
			if device.Name == podNetworkName {
				ports := []v1.ServicePort{}
				for i, port := range device.Ports {
					ports = append(ports, v1.ServicePort{Name: fmt.Sprintf("port-%d", i+1), Protocol: v1.Protocol(port.Protocol), Port: port.Port})
				}
				return ports
			}
		}
	}
	return nil
}
