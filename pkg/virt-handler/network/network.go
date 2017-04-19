package network

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/ipam"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/jeevatkm/go-model"
	. "github.com/projectcalico/cni-plugin/utils"
	"github.com/projectcalico/libcalico-go/lib/api"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
)

// This function creates a virtual interface on the Kubernetes cluster network
// and binds the VM to it.
func AddToNetwork(vm *v1.VM, namespace string) (*v1.VM, error) {
	vmCopy := &v1.VM{}
	model.Copy(vmCopy, vm)

	for idx, iface := range vmCopy.Spec.Domain.Devices.Interfaces {
		if iface.Type == "cni" {
			logging.DefaultLogger().V(3).Info().Object(vm).Msg("Mapping Interface")

			// Load a network configuration file
			netConfDir := os.Getenv("CNI_CONF")
			pluginDir := os.Getenv("CNI_PATH")

			files, err := libcni.ConfFiles(netConfDir)
			switch {
			case err != nil:
				return vm, err
			case len(files) == 0:
				return vm, fmt.Errorf("No networks found in %s", netConfDir)
			}
			sort.Strings(files)
			for _, file := range files {
				conf, err := libcni.ConfFromFile(file)
				if err != nil {
					return vm, fmt.Errorf("Error loading CNI config file %s: %v", file, err)
				}

				logging.DefaultLogger().Object(vm).Info().Msgf("Found %s network provider", conf.Network.Type)

				// network plugin specific code
				switch conf.Network.Type {
				case "calico":
					workload := vmCopy.Spec.Domain.Name
					orchestrator := "k8s"
					hostname, _ := os.Hostname()

					logger := CreateContextLogger(workload)

					// Get an IP address from calico-ipam
					// Collect the result in this variable - this is ultimately what gets "returned" by this function by printing
					// it to stdout.
					var result *types.Result

					// Set env variables to call calico-ipam
					os.Setenv("CNI_COMMAND", "ADD")
					os.Setenv("CNI_NETNS", "nil")
					os.Setenv("CNI_IFNAME", "nil")
					cniArgs := fmt.Sprintf("IgnoreUnknown=1;K8S_POD_NAMESPACE=%s;K8S_POD_NAME=%s", namespace, workload)
					os.Setenv("CNI_ARGS", cniArgs)

					result, err = ipam.ExecAdd(conf.Network.IPAM.Type, conf.Bytes)
					if err != nil {
						return vm, err
					}
					logging.DefaultLogger().Object(vm).Info().Msgf("Got result from IPAM plugin: %v", result)

					cniConf := NetConf{}
					if err := json.Unmarshal(conf.Bytes, &cniConf); err != nil {
						return vm, fmt.Errorf("failed to load netconf: %v", err)
					}

					calicoClient, err := CreateClient(cniConf)
					if err != nil {
						return vm, err
					}

					// Create the endpoint object and configure it.
					var endpoint *api.WorkloadEndpoint
					endpoint = api.NewWorkloadEndpoint()
					endpoint.Metadata.Name = "eth0"
					endpoint.Metadata.Node = hostname
					endpoint.Metadata.Orchestrator = orchestrator
					endpoint.Metadata.Workload = workload
					//endpoint.Metadata.Labels = labels

					// Set the profileID according to whether Kubernetes policy is required.
					// If it's not, then just use the network name (which is the normal behavior)
					// otherwise use one based on the Kubernetes pod's Namespace.
					if cniConf.Policy.PolicyType == "k8s" {
						endpoint.Spec.Profiles = []string{fmt.Sprintf("k8s_ns.%s", namespace)}
					} else {
						endpoint.Spec.Profiles = []string{cniConf.Name}
					}

					// Populate the endpoint with the output from the IPAM plugin.
					if err = PopulateEndpointNets(endpoint, result); err != nil {
						// Cleanup IP allocation and return the error.
						ReleaseIPAllocation(logger, cniConf.IPAM.Type, conf.Bytes)
						return vm, err
					}
					logging.DefaultLogger().Object(vm).Info().Msgf("Populated endpoint: %v", endpoint)

					// create a tap device
					//nameTemplate := fmt.Sprintf("tap-%s-%%d", vmCopy.Spec.Domain.UUID[0:8])
					//ifName, err := tuntap.CreatePersistentIface(nameTemplate, tuntap.Tap)
					ifName := fmt.Sprintf("tap-%s", vmCopy.Spec.Domain.UUID[0:8])
					tap := &netlink.Tuntap{
						LinkAttrs: netlink.LinkAttrs{Name: ifName},
						Mode:      netlink.TUNTAP_MODE_TAP,
					}
					if err := netlink.LinkAdd(tap); err != nil {
						return vm, fmt.Errorf("Failed to create a tap device: %v", err)
					}
					//if err != nil {
					//	return vm, fmt.Errorf("Failed to create a tap device: %v", err)
					//}

					link, err := netlink.LinkByName(ifName)
					if err != nil {
						return vm, fmt.Errorf("cannot find link %q", ifName)
					}
					logging.DefaultLogger().Object(vm).Info().Msgf("Link(tap) %v created", link)

					if err = netlink.LinkSetUp(link); err != nil {
						return nil, fmt.Errorf("cannot set link up %q", ifName)
					}

					// From calico endpoint manager
					//(https://github.com/projectcalico/felix/blob/master/intdataplane/endpoint_mgr.go)
					// Enable strict reverse-path filtering.  This prevents a workload from spoofing its
					// IP address.  Non-privileged containers have additional anti-spoofing protection
					// but VM workloads, for example, can easily spoof their IP.
					err = writeProcSys(fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/rp_filter", ifName), "1")
					if err != nil {
						return vm, err
					}
					// Enable routing to localhost.  This is required to allow for NAT to the local
					// host.
					err = writeProcSys(fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/route_localnet", ifName), "1")
					if err != nil {
						return vm, err
					}
					// Enable proxy ARP, this makes the host respond to all ARP requests with its own
					// MAC.  This has a couple of advantages:
					//
					// - In OpenStack, we're forced to configure the guest's networking using DHCP.
					//   Since DHCP requires a subnet and gateway, representing the Calico network
					//   in the natural way would lose a lot of IP addresses.  For IPv4, we'd have to
					//   advertise a distinct /30 to each guest, which would use up 4 IPs per guest.
					//   Using proxy ARP, we can advertise the whole pool to each guest as its subnet
					//   but have the host respond to all ARP requests and route all the traffic whether
					//   it is on or off subnet.
					//
					// - For containers, we install explicit routes into the containers network
					//   namespace and we use a link-local address for the gateway.  Turing on proxy ARP
					//   means that we don't need to assign the link local address explicitly to each
					//   host side of the veth, which is one fewer thing to maintain and one fewer
					//   thing we may clash over.
					err = writeProcSys(fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/proxy_arp", ifName), "1")
					if err != nil {
						return vm, err
					}
					// Normally, the kernel has a delay before responding to proxy ARP but we know
					// that's not needed in a Calico network so we disable it.
					err = writeProcSys(fmt.Sprintf("/proc/sys/net/ipv4/neigh/%s/proxy_delay", ifName), "0")
					if err != nil {
						return vm, err
					}
					// Enable IP forwarding of packets coming _from_ this interface.  For packets to
					// be forwarded in both directions we need this flag to be set on the fabric-facing
					// interface too (or for the global default to be set).
					err = writeProcSys(fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/forwarding", ifName), "1")
					if err != nil {
						return vm, err
					}

					mac := link.Attrs().HardwareAddr
					endpoint.Spec.MAC = &cnet.MAC{HardwareAddr: mac}
					endpoint.Spec.InterfaceName = ifName
					logger.WithField("endpoint", endpoint).Info("Added Mac and interface name to endpoint")

					// Write the endpoint object (either the newly created one, or the updated one)
					if _, err := calicoClient.WorkloadEndpoints().Apply(endpoint); err != nil {
						// Cleanup IP allocation and return the error.
						ReleaseIPAllocation(logger, cniConf.IPAM.Type, conf.Bytes)
						return vm, err
					}
					logger.Info("Wrote updated endpoint to datastore")

					logging.DefaultLogger().Object(vm).Info().Msgf("Mapping %s", ifName)
					newIface := v1.Interface{}

					newIface.Type = "ethernet"
					newIface.MAC = new(v1.MAC)
					newIface.MAC.MAC = mac.String()
					newIface.Target = new(v1.InterfaceTarget)
					newIface.Target.Device = ifName
					newIface.Model = new(v1.Model)
					newIface.Model.Type = "virtio"

					vmCopy.Spec.Domain.Devices.Interfaces[idx] = newIface
					return vmCopy, nil
				}
			}
			return vm, fmt.Errorf("No %s network provider found in %s", iface.Source.Network, pluginDir)
		}
	}
	return vmCopy, nil
}

func DeleteFromNetwork(vm *v1.VM, namespace string) error {
	netConfDir := os.Getenv("CNI_CONF")
	files, err := libcni.ConfFiles(netConfDir)
	switch {
	case err != nil:
		return err
	case len(files) == 0:
		return fmt.Errorf("No networks found in %s", netConfDir)
	}
	sort.Strings(files)
	for _, file := range files {
		conf, err := libcni.ConfFromFile(file)
		if err != nil {
			return fmt.Errorf("Error loading CNI config file %s: %v", file, err)
		}

		logging.DefaultLogger().Object(vm).Info().Msgf("Found %s network provider", conf.Network.Type)

		// network plugin specific code
		switch conf.Network.Type {
		case "calico":
			// Release IP address
			workload := vm.GetObjectMeta().GetName()
			logging.DefaultLogger().Object(vm).Info().Msgf("VM spec when del %+v", vm.GetObjectMeta())
			orchestrator := "k8s"
			hostname, _ := os.Hostname()
			os.Setenv("CNI_COMMAND", "DEL")
			os.Setenv("CNI_NETNS", "nil")
			os.Setenv("CNI_IFNAME", "nil")
			cniArgs := fmt.Sprintf("IgnoreUnknown=1;K8S_POD_NAMESPACE=%s;K8S_POD_NAME=%s", namespace, workload)
			os.Setenv("CNI_ARGS", cniArgs)

			ipamErr := ipam.ExecDel(conf.Network.IPAM.Type, conf.Bytes)

			if ipamErr != nil {
				logging.DefaultLogger().Object(vm).Info().Msgf("IPAM error %v", ipamErr)
			}

			cniConf := NetConf{}
			if err := json.Unmarshal(conf.Bytes, &cniConf); err != nil {
				return fmt.Errorf("failed to load netconf: %v", err)
			}
			calicoClient, err := CreateClient(cniConf)
			if err != nil {
				return err
			}

			// Get tap interface name
			endpoints, err := calicoClient.WorkloadEndpoints().List(api.WorkloadEndpointMetadata{
				Node:         hostname,
				Orchestrator: orchestrator,
				Workload:     workload})
			if err != nil {
				return err
			}

			if len(endpoints.Items) == 1 {
				var endpoint *api.WorkloadEndpoint
				endpoint = &endpoints.Items[0]
				// Delete tap interface
				//tuntap.RemovePersistentIface(endpoint.Spec.InterfaceName, tuntap.Tap)
				link, err := netlink.LinkByName(endpoint.Spec.InterfaceName)
				if err != nil {
					return fmt.Errorf("cannot find link %q", endpoint.Spec.InterfaceName)
				}
				netlink.LinkDel(link)
			}

			// Delete endpoint in calico data store
			if err := calicoClient.WorkloadEndpoints().Delete(api.WorkloadEndpointMetadata{
				Name:         "eth0",
				Node:         hostname,
				Orchestrator: orchestrator,
				Workload:     workload}); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func writeProcSys(path, value string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	n, err := f.Write([]byte(value))
	if err == nil && n < len(value) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}
