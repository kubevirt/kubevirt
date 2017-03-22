package virthandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/ipam"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/coreos/rkt/networking/tuntap"
	"github.com/jeevatkm/go-model"
	. "github.com/projectcalico/cni-plugin/utils"
	"github.com/projectcalico/libcalico-go/lib/api"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/vishvananda/netlink"
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

func NewVMController(lw cache.ListerWatcher, domainManager virtwrap.DomainManager, recorder record.EventRecorder, restClient rest.RESTClient, clientset *kubernetes.Clientset, host string) (cache.Store, workqueue.RateLimitingInterface, *kubecli.Controller) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	dispatch := NewVMHandlerDispatch(domainManager, recorder, &restClient, clientset, host)

	indexer, informer := kubecli.NewController(lw, queue, &v1.VM{}, dispatch)
	return indexer, queue, informer

}
func NewVMHandlerDispatch(domainManager virtwrap.DomainManager, recorder record.EventRecorder, restClient *rest.RESTClient, clientset *kubernetes.Clientset, host string) kubecli.ControllerDispatch {
	return &VMHandlerDispatch{
		domainManager: domainManager,
		recorder:      recorder,
		restClient:    *restClient,
		clientset:     clientset,
		host:          host,
	}
}

type VMHandlerDispatch struct {
	domainManager virtwrap.DomainManager
	recorder      record.EventRecorder
	restClient    rest.RESTClient
	clientset     *kubernetes.Clientset
	host          string
}

func (d *VMHandlerDispatch) Execute(store cache.Store, queue workqueue.RateLimitingInterface, key interface{}) {

	// Fetch the latest Vm state from cache
	obj, exists, err := store.GetByKey(key.(string))

	if err != nil {
		queue.AddRateLimited(key)
		return
	}

	// Retrieve the VM
	var vm *v1.VM
	if !exists {
		_, name, err := cache.SplitMetaNamespaceKey(key.(string))
		if err != nil {
			// TODO do something more smart here
			queue.AddRateLimited(key)
			return
		}
		vm = v1.NewVMReferenceFromName(name)

		// If we don't have the VM in the cache, it could be that it is currently migrating to us
		result := d.restClient.Get().Name(vm.GetObjectMeta().GetName()).Resource("vms").Namespace(kubeapi.NamespaceDefault).Do()
		if result.Error() == nil {
			// So the VM still seems to exist
			fetchedVM, err := result.Get()
			if err != nil {
				// Since there was no fetch error, this should have worked, let's back off
				queue.AddRateLimited(key)
				return
			}
			if fetchedVM.(*v1.VM).Status.MigrationNodeName == d.host {
				// OK, this VM is migrating to us, don't interrupt it
				queue.Forget(key)
				return
			}
		} else if result.Error().(*errors.StatusError).Status().Code != int32(http.StatusNotFound) {
			// Something went wrong, let's try again later
			queue.AddRateLimited(key)
			return
		}
<<<<<<< 8f5c4f8e474e665696dd1a0f5d9d7b4dfbbb4ca4
		// The VM is deleted on the cluster, let's go on with the deletion on the host
	} else {
		vm = obj.(*v1.VM)
	}
	logging.DefaultLogger().V(3).Info().Object(vm).Msg("Processing VM update.")

	// Process the VM
	if !exists {
		// Since the VM was not in the cache, we delete it
		DeleteInterfaces(vm, kubeapi.NamespaceDefault)
		err = d.domainManager.KillVM(vm)
	} else if isWorthSyncing(vm) {
		// Synchronize the VM state
		vm, err = MapPersistentVolumes(vm, d.clientset.CoreV1().RESTClient(), kubeapi.NamespaceDefault)

		if err == nil {
				vm, err = MapInterfaces(vm, kubeapi.NamespaceDefault)
		}

		if err == nil {
			// TODO check if found VM has the same UID like the domain, if not, delete the Domain first

			// Only sync if the VM is not marked as migrating. Everything except shutting down the VM is not permitted when it is migrating.
			// TODO MigrationNodeName should be a pointer
			if vm.Status.MigrationNodeName == "" {
				err = d.domainManager.SyncVM(vm)
			} else {
				queue.Forget(key)
				return
			}
		}

		// Update VM status to running
		if err == nil && vm.Status.Phase != v1.Running {
			obj, err = kubeapi.Scheme.Copy(vm)
			if err == nil {
				vm = obj.(*v1.VM)
				vm.Status.Phase = v1.Running
				err = d.restClient.Put().Resource("vms").Body(vm).
					Name(vm.ObjectMeta.Name).Namespace(kubeapi.NamespaceDefault).Do().Error()
			}
		}
	}

	if err != nil {
		// Something went wrong, reenqueue the item with a delay
		logging.DefaultLogger().Error().Object(vm).Reason(err).Msg("Synchronizing the VM failed.")
		d.recorder.Event(vm, kubev1.EventTypeWarning, v1.SyncFailed.String(), err.Error())
		queue.AddRateLimited(key)
		return
	}

	logging.DefaultLogger().V(3).Info().Object(vm).Msg("Synchronizing the VM succeeded.")
	queue.Forget(key)
	return
}

// Almost everything in the VM object maps exactly to its domain counterpart
// One exception is persistent volume claims. This function looks up each PV
// and inserts a corrected disk entry into the VM's device map.
func MapPersistentVolumes(vm *v1.VM, restClient cache.Getter, namespace string) (*v1.VM, error) {
	vmCopy := &v1.VM{}
	model.Copy(vmCopy, vm)

	for idx, disk := range vmCopy.Spec.Domain.Devices.Disks {
		if disk.Type == "PersistentVolumeClaim" {
			logging.DefaultLogger().V(3).Info().Object(vm).Msgf("Mapping PersistentVolumeClaim: %s", disk.Source.Name)

			// Look up existing persistent volume
			obj, err := restClient.Get().Namespace(namespace).Resource("persistentvolumeclaims").Name(disk.Source.Name).Do().Get()

			if err != nil {
				logging.DefaultLogger().Error().Reason(err).Msg("unable to look up persistent volume claim")
				return vm, fmt.Errorf("unable to look up persistent volume claim: %v", err)
			}

			pvc := obj.(*kubev1.PersistentVolumeClaim)
			if pvc.Status.Phase != kubev1.ClaimBound {
				logging.DefaultLogger().Error().Msg("attempted use of unbound persistent volume")
				return vm, fmt.Errorf("attempted use of unbound persistent volume claim: %s", pvc.Name)
			}

			// Look up the PersistentVolume this PVC is bound to
			// Note: This call is not namespaced!
			obj, err = restClient.Get().Resource("persistentvolumes").Name(pvc.Spec.VolumeName).Do().Get()

			if err != nil {
				logging.DefaultLogger().Error().Reason(err).Msg("unable to access persistent volume record")
				return vm, fmt.Errorf("unable to access persistent volume record: %v", err)
			}
			pv := obj.(*kubev1.PersistentVolume)

			if pv.Spec.ISCSI != nil {
				logging.DefaultLogger().Object(vm).Info().Msg("Mapping iSCSI PVC")
				newDisk := v1.Disk{}

				newDisk.Type = "network"
				newDisk.Device = "disk"
				newDisk.Target = disk.Target
				newDisk.Driver = new(v1.DiskDriver)
				newDisk.Driver.Type = "raw"
				newDisk.Driver.Name = "qemu"

				newDisk.Source.Name = fmt.Sprintf("%s/%d", pv.Spec.ISCSI.IQN, pv.Spec.ISCSI.Lun)
				newDisk.Source.Protocol = "iscsi"

				hostPort := strings.Split(pv.Spec.ISCSI.TargetPortal, ":")
				newDisk.Source.Host = &v1.DiskSourceHost{}
				newDisk.Source.Host.Name = hostPort[0]
				if len(hostPort) > 1 {
					newDisk.Source.Host.Port = hostPort[1]
				}

				vmCopy.Spec.Domain.Devices.Disks[idx] = newDisk
			} else {
				logging.DefaultLogger().Object(vm).Error().Msg(fmt.Sprintf("Referenced PV %v is backed by an unsupported storage type", pv))
			}
		}
	}

	return vmCopy, nil
}


func isWorthSyncing(vm *v1.VM) bool {
	return vm.Status.Phase != v1.Succeeded && vm.Status.Phase != v1.Failed
}

// This function creates a virtual interface on the Kubernetes cluster network
// and binds the VM to it.
func MapInterfaces(vm *v1.VM, namespace string) (*v1.VM, error) {
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
				nameTemplate := fmt.Sprintf("tap-%s-%%d", vmCopy.Spec.Domain.UUID[0:8])
				ifName, err := tuntap.CreatePersistentIface(nameTemplate, tuntap.Tap)
				if err != nil {
					return vm, fmt.Errorf("Failed to create a tap device: %v", err)
				}

				link, err := netlink.LinkByName(ifName)
				if err != nil {
					return vm, fmt.Errorf("cannot find link %q", ifName)
				}
				logging.DefaultLogger().Object(vm).Info().Msgf("Link(tap) %v created", link)

				if err = netlink.LinkSetUp(link); err != nil {
					return nil, fmt.Errorf("cannot set link up %q", ifName)
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
			return vm, fmt.Errorf("No %s network provider found in %s", iface.Source.Network, pluginDir)
		}
	}
	return vmCopy, nil
}

func DeleteInterfaces(vm *v1.VM, namespace string) error {
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
			tuntap.RemovePersistentIface(endpoint.Spec.InterfaceName, tuntap.Tap)
		}

		// Delete endpoint in calico data store
		if err := calicoClient.WorkloadEndpoints().Delete(api.WorkloadEndpointMetadata{
			Name:         "eth0",
			Node:         hostname,
			Orchestrator: orchestrator,
			Workload:     workload}); err != nil {
			return err
		}
	}
	return nil
}
