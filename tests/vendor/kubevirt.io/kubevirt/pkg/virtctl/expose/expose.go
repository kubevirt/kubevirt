package expose

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"

	v12 "kubevirt.io/api/core/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

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
var strIPFamily string
var strIPFamilyPolicy string

// NewExposeCommand generates a new "expose" command
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
		Args:    templates.ExactArgs("expose", 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{command: COMMAND_EXPOSE, clientConfig: clientConfig}
			return c.RunE(args)
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
	cmd.Flags().StringVar(&strIPFamily, "ip-family", "", "IP family over which the service will be exposed. Valid values are 'IPv4', 'IPv6', 'IPv4,IPv6' or 'IPv6,IPv4'")
	cmd.Flags().StringVar(&strIPFamilyPolicy, "ip-family-policy", "", "IP family policy defines whether the service can use IPv4, IPv6, or both. Valid values are 'SingleStack', 'PreferDualStack' or 'RequireDualStack'")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	usage := `  # Expose SSH to a virtual machine instance called 'myvm' on each node via a NodePort service:
  {{ProgramName}} expose vmi myvm --port=22 --name=myvm-ssh --type=NodePort

  # Expose all defined pod-network ports of a virtual machine instance replicaset on a service:
  {{ProgramName}} expose vmirs myvmirs --name=vmirs-service

  # Expose port 8080 as port 80 from a virtual machine instance replicaset on a service:
  {{ProgramName}} expose vmirs myvmirs --port=80 --target-port=8080 --name=vmirs-service`
	return usage
}

// executing the "expose" command
func (o *Command) RunE(args []string) error {
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

	ipFamilies, err := convertIPFamily(strIPFamily)
	if err != nil {
		return err
	}

	ipFamilyPolicy, err := convertIPFamilyPolicy(strIPFamilyPolicy, ipFamilies)
	if err != nil {
		return err
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
		vmi, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vmName, &options)
		if err != nil {
			return fmt.Errorf("error fetching VirtualMachineInstance: %v", err)
		}
		serviceSelector = vmi.ObjectMeta.Labels
		ports = podNetworkPorts(&vmi.Spec)
		// remove unwanted labels
		delete(serviceSelector, virtv1.NodeNameLabel)
		delete(serviceSelector, virtv1.VirtualMachinePoolRevisionName)
		delete(serviceSelector, virtv1.MigrationTargetNodeNameLabel)
	case "vm", "vms", "virtualmachine", "virtualmachines":
		// get the VM
		vm, err := virtClient.VirtualMachine(namespace).Get(context.Background(), vmName, &options)
		if err != nil {
			return fmt.Errorf("error fetching Virtual Machine: %v", err)
		}
		if vm.Spec.Template != nil {
			ports = podNetworkPorts(&vm.Spec.Template.Spec)
		}
		serviceSelector = vm.Spec.Template.ObjectMeta.Labels
		delete(serviceSelector, virtv1.VirtualMachinePoolRevisionName)
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
		return fmt.Errorf("cannot expose %s without any label: %s", vmType, vmName)
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
			IPFamilies:     ipFamilies,
		},
	}

	// set external IP if provided
	if len(externalIP) > 0 {
		service.Spec.ExternalIPs = []string{externalIP}
	}

	if ipFamilyPolicy != "" {
		service.Spec.IPFamilyPolicy = &ipFamilyPolicy
	}

	major, minor, err := serverVersion(virtClient)
	if err != nil {
		return err
	}

	if major > 1 || (major == 1 && minor >= 20) {
		_, err = virtClient.CoreV1().Services(namespace).Create(context.Background(), service, k8smetav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("service creation failed for k8s >= 1.20: %v", err)
		}
		// For k8s < 1.20 we have to "migrate" the "ipFamilies" field to
		// "ipFamily" we do this using an unstructured approach
	} else {
		if len(ipFamilies) > 1 {
			return fmt.Errorf("k8s < 1.20 doesn't support multiple ip families")
		}

		if ipFamilyPolicy != "" {
			return fmt.Errorf("k8s < 1.20 doesn't support 'ipFamilyPolicy'")
		}
		// convert the Service to unstructured.Unstructured
		unstructuredService, err := runtime.DefaultUnstructuredConverter.ToUnstructured(service)
		if err != nil {
			return err
		}

		// Add ipFamily field with proper content
		err = unstructured.SetNestedField(unstructuredService, string(ipFamilies[0]), "spec", "ipFamily")
		if err != nil {
			return err
		}

		// try to create the service on the cluster
		_, err = virtClient.DynamicClient().Resource(schema.GroupVersionResource{Version: "v1", Resource: "services"}).Namespace(namespace).Create(context.Background(), &unstructured.Unstructured{Object: unstructuredService}, k8smetav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("service creation failed for k8s < 1.20: %v", err)
		}
	}
	fmt.Printf("Service %s successfully exposed for %s %s\n", serviceName, vmType, vmName)
	return nil
}

func convertIPFamily(strIPFamily string) ([]v1.IPFamily, error) {
	switch strings.ToLower(strIPFamily) {
	case "":
		return []v1.IPFamily{}, nil
	case "ipv4":
		return []v1.IPFamily{v1.IPv4Protocol}, nil
	case "ipv6":
		return []v1.IPFamily{v1.IPv6Protocol}, nil
	case "ipv4,ipv6":
		return []v1.IPFamily{v1.IPv4Protocol, v1.IPv6Protocol}, nil
	case "ipv6,ipv4":
		return []v1.IPFamily{v1.IPv6Protocol, v1.IPv4Protocol}, nil
	default:
		return nil, fmt.Errorf("unknown IPFamily/s: %s", strIPFamily)
	}
}

func convertIPFamilyPolicy(strIPFamilyPolicy string, ipFamilies []v1.IPFamily) (v1.IPFamilyPolicyType, error) {
	switch strings.ToLower(strIPFamilyPolicy) {
	case "":
		if len(ipFamilies) > 1 {
			return v1.IPFamilyPolicyPreferDualStack, nil
		}
		return "", nil
	case "singlestack":
		return v1.IPFamilyPolicySingleStack, nil
	case "preferdualstack":
		return v1.IPFamilyPolicyPreferDualStack, nil
	case "requiredualstack":
		return v1.IPFamilyPolicyRequireDualStack, nil
	default:
		return "", fmt.Errorf("unknown IPFamilyPolicy/s: %s", strIPFamilyPolicy)
	}
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

func serverVersion(virtClient kubecli.KubevirtClient) (major int, minor int, err error) {
	serverVersion, err := virtClient.DiscoveryClient().ServerVersion()
	if err != nil {
		return 0, 0, err
	}
	// Make a Regex to say we only want numbers
	reg, err := regexp.Compile("[^0-9]+")
	if err != nil {
		return 0, 0, err
	}
	major, err = strconv.Atoi(reg.ReplaceAllString(serverVersion.Major, ""))
	if err != nil {
		return 0, 0, err
	}
	minor, err = strconv.Atoi(reg.ReplaceAllString(serverVersion.Minor, ""))
	if err != nil {
		return 0, 0, err
	}
	return
}
