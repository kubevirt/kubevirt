package base_defaults

import (
	"context"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// TODO: Should we make the function signatures of these setters more uniform by passing
// the clusterConfig as parameter to all of them? Order of parameters could be always (clusterConfig, vm/vmi/spec).
type BaseDefaults struct {
	defaultVMMachineTypeSetter                func(vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig)
	defaultVMIMachineTypeSetter               func(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstanceSpec)
	currentCPUTopologyStatusSetter            func(vmi *v1.VirtualMachineInstance)
	guestMemoryStatusSetter                   func(vmi *v1.VirtualMachineInstance)
	defaultHypervFeatureDependenciesSetter    func(spec *v1.VirtualMachineInstanceSpec)
	defaultEvictionStrategySetter             func(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec)
	defaultPullPoliciesOnContainerDisksSetter func(spec *v1.VirtualMachineInstanceSpec)
	guestMemorySetter                         func(spec *v1.VirtualMachineInstanceSpec)
	defaultResourceRequestsSetter             func(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec)
	defaultGuestCPUTopologySetter             func(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec)
	defaultCPUModelSetter                     func(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec)
	defaultArchitectureSetter                 func(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec)
	defaultNetworkInterfaceSetter             func(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) error
	defaultArchitectureFromDataSourceSetter   func(clusterConfig *virtconfig.ClusterConfig, vm *v1.VirtualMachine, virtClient kubecli.KubevirtClient)
	defaultVolumeDiskSetter                   func(spec *v1.VirtualMachineInstanceSpec)
}

func NewBaseDefaults() *BaseDefaults {
	return &BaseDefaults{
		defaultVMMachineTypeSetter:                setVMDefaultMachineType,
		defaultVMIMachineTypeSetter:               setDefaultMachineType,
		currentCPUTopologyStatusSetter:            setCurrentCPUTopologyStatus,
		guestMemoryStatusSetter:                   setGuestMemoryStatus,
		defaultHypervFeatureDependenciesSetter:    setDefaultHypervFeatureDependencies,
		defaultEvictionStrategySetter:             setDefaultEvictionStrategy,
		defaultPullPoliciesOnContainerDisksSetter: setDefaultPullPoliciesOnContainerDisks,
		guestMemorySetter:                         setGuestMemory,
		defaultResourceRequestsSetter:             setDefaultResourceRequests,
		defaultGuestCPUTopologySetter:             SetDefaultGuestCPUTopology,
		defaultCPUModelSetter:                     setDefaultCPUModel,
		defaultArchitectureSetter:                 setDefaultArchitecture,
		defaultArchitectureFromDataSourceSetter:   setDefaultArchitectureFromDataSource,
		defaultNetworkInterfaceSetter:             vmispec.SetDefaultNetworkInterface,
		defaultVolumeDiskSetter:                   util.SetDefaultVolumeDisk,
	}
}

func (b *BaseDefaults) SetVirtualMachineDefaults(vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig, virtClient kubecli.KubevirtClient) {
	b.defaultArchitectureFromDataSourceSetter(clusterConfig, vm, virtClient)
	b.defaultArchitectureSetter(clusterConfig, &vm.Spec.Template.Spec)
	b.defaultVMMachineTypeSetter(vm, clusterConfig)
}

func (b *BaseDefaults) SetDefaultVirtualMachineInstance(clusterConfig *virtconfig.ClusterConfig, vmi *v1.VirtualMachineInstance) error {
	if err := b.SetDefaultVirtualMachineInstanceSpec(clusterConfig, &vmi.Spec); err != nil {
		return err
	}
	// TODO setDefaultFeatures(&vmi.Spec) // TODO This is an architecture specific function
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	b.defaultHypervFeatureDependenciesSetter(&vmi.Spec)
	b.defaultCPUModelSetter(clusterConfig, &vmi.Spec)
	b.guestMemoryStatusSetter(vmi)
	b.currentCPUTopologyStatusSetter(vmi)

	// TODO Call the per-architecture defaulters here

	// Hotplug needs to be enabled on ARM yet
	// TODO Move this to architecture specific defaults
	if !IsARM64(&vmi.Spec) {
		setupHotplug(clusterConfig, vmi)
	}

	return nil
}

func (b *BaseDefaults) SetDefaultVirtualMachineInstanceSpec(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) error {
	b.defaultArchitectureSetter(clusterConfig, spec)
	b.defaultVMIMachineTypeSetter(clusterConfig, spec)
	b.defaultResourceRequestsSetter(clusterConfig, spec)
	b.guestMemorySetter(spec)
	b.defaultGuestCPUTopologySetter(clusterConfig, spec)
	b.defaultPullPoliciesOnContainerDisksSetter(spec)
	b.defaultEvictionStrategySetter(clusterConfig, spec)
	if err := b.defaultNetworkInterfaceSetter(clusterConfig, spec); err != nil {
		return err
	}
	util.SetDefaultVolumeDisk(spec)
	return nil
}

func setVMDefaultMachineType(vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) {
	// Nothing to do, let's the validating webhook fail later
	if vm.Spec.Template == nil {
		return
	}

	if machine := vm.Spec.Template.Spec.Domain.Machine; machine != nil && machine.Type != "" {
		return
	}

	if vm.Spec.Template.Spec.Domain.Machine == nil {
		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{}
	}

	if vm.Spec.Template.Spec.Domain.Machine.Type == "" {
		vm.Spec.Template.Spec.Domain.Machine.Type = clusterConfig.GetMachineType(vm.Spec.Template.Spec.Architecture)
	}
}

func setCurrentCPUTopologyStatus(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.CPU != nil && vmi.Status.CurrentCPUTopology == nil {
		vmi.Status.CurrentCPUTopology = &v1.CPUTopology{
			Sockets: vmi.Spec.Domain.CPU.Sockets,
			Cores:   vmi.Spec.Domain.CPU.Cores,
			Threads: vmi.Spec.Domain.CPU.Threads,
		}
	}
}

func setGuestMemoryStatus(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.Memory != nil &&
		vmi.Spec.Domain.Memory.Guest != nil {
		vmi.Status.Memory = &v1.MemoryStatus{
			GuestAtBoot:    vmi.Spec.Domain.Memory.Guest,
			GuestCurrent:   vmi.Spec.Domain.Memory.Guest,
			GuestRequested: vmi.Spec.Domain.Memory.Guest,
		}
	}
}

func setDefaultHypervFeatureDependencies(spec *v1.VirtualMachineInstanceSpec) {
	// In a future, yet undecided, release either libvirt or QEMU are going to check the hyperv dependencies, so we can get rid of this code.
	// Until that time, we need to handle the hyperv deps to avoid obscure rejections from QEMU later on
	log.Log.V(4).Info("Set HyperV dependencies")
	if err := SetHypervFeatureDependencies(spec); err != nil {
		// HyperV is a special case. If our best-effort attempt fails, we should leave
		// rejection to be performed later on in the validating webhook, and continue here.
		// Please note this means that partial changes may have been performed.
		// This is OK since each dependency must be atomic and independent (in ACID sense),
		// so the VMI configuration is still legal.
		log.Log.V(2).Infof("Failed to set HyperV dependencies: %s", err)
	}
}

func setDefaultEvictionStrategy(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	if spec.EvictionStrategy == nil {
		spec.EvictionStrategy = clusterConfig.GetConfig().EvictionStrategy
	}
}

func setDefaultMachineType(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	machineType := clusterConfig.GetMachineType(spec.Architecture)

	if machine := spec.Domain.Machine; machine != nil {
		if machine.Type == "" {
			machine.Type = machineType
		}
	} else {
		spec.Domain.Machine = &v1.Machine{Type: machineType}
	}

}

func setDefaultPullPoliciesOnContainerDisks(spec *v1.VirtualMachineInstanceSpec) {
	for _, volume := range spec.Volumes {
		if volume.ContainerDisk != nil && volume.ContainerDisk.ImagePullPolicy == "" {
			if strings.HasSuffix(volume.ContainerDisk.Image, ":latest") || !strings.ContainsAny(volume.ContainerDisk.Image, ":@") {
				volume.ContainerDisk.ImagePullPolicy = k8sv1.PullAlways
			} else {
				volume.ContainerDisk.ImagePullPolicy = k8sv1.PullIfNotPresent
			}
		}
	}
}

func setGuestMemory(spec *v1.VirtualMachineInstanceSpec) {
	if spec.Domain.Memory != nil &&
		spec.Domain.Memory.Guest != nil {
		return
	}

	if spec.Domain.Memory == nil {
		spec.Domain.Memory = &v1.Memory{}
	}

	switch {
	case !spec.Domain.Resources.Requests.Memory().IsZero():
		spec.Domain.Memory.Guest = spec.Domain.Resources.Requests.Memory()
	case !spec.Domain.Resources.Limits.Memory().IsZero():
		spec.Domain.Memory.Guest = spec.Domain.Resources.Limits.Memory()
	case spec.Domain.Memory.Hugepages != nil:
		if hugepagesSize, err := resource.ParseQuantity(spec.Domain.Memory.Hugepages.PageSize); err == nil {
			spec.Domain.Memory.Guest = &hugepagesSize
		}
	}

}

func setDefaultResourceRequests(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	resources := &spec.Domain.Resources

	if !resources.Limits.Cpu().IsZero() && resources.Requests.Cpu().IsZero() {
		if resources.Requests == nil {
			resources.Requests = k8sv1.ResourceList{}
		}
		resources.Requests[k8sv1.ResourceCPU] = resources.Limits[k8sv1.ResourceCPU]
	}

	if !resources.Limits.Memory().IsZero() && resources.Requests.Memory().IsZero() {
		if resources.Requests == nil {
			resources.Requests = k8sv1.ResourceList{}
		}
		resources.Requests[k8sv1.ResourceMemory] = resources.Limits[k8sv1.ResourceMemory]
	}

	if _, exists := resources.Requests[k8sv1.ResourceMemory]; !exists {
		var memory *resource.Quantity
		if spec.Domain.Memory != nil && spec.Domain.Memory.Guest != nil {
			memory = spec.Domain.Memory.Guest
		}
		if memory == nil && spec.Domain.Memory != nil && spec.Domain.Memory.Hugepages != nil {
			if hugepagesSize, err := resource.ParseQuantity(spec.Domain.Memory.Hugepages.PageSize); err == nil {
				memory = &hugepagesSize
			}
		}

		if memory != nil && memory.Value() > 0 {
			if resources.Requests == nil {
				resources.Requests = k8sv1.ResourceList{}
			}
			resources.Requests[k8sv1.ResourceMemory] = *memory
		}
	}

	if cpuRequest := clusterConfig.GetCPURequest(); !cpuRequest.Equal(resource.MustParse(virtconfig.DefaultCPURequest)) {
		if _, exists := resources.Requests[k8sv1.ResourceCPU]; !exists {
			if spec.Domain.CPU != nil && spec.Domain.CPU.DedicatedCPUPlacement {
				return
			}
			if resources.Requests == nil {
				resources.Requests = k8sv1.ResourceList{}
			}
			resources.Requests[k8sv1.ResourceCPU] = *cpuRequest
		}
	}
}

func SetDefaultGuestCPUTopology(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	cores := uint32(1)
	threads := uint32(1)
	sockets := uint32(1)
	vmiCPU := spec.Domain.CPU
	if vmiCPU == nil || (vmiCPU.Cores == 0 && vmiCPU.Sockets == 0 && vmiCPU.Threads == 0) {
		// create cpu topology struct
		if spec.Domain.CPU == nil {
			spec.Domain.CPU = &v1.CPU{}
		}
		//if cores, sockets, threads are not set, take value from domain resources request or limits and
		//set value into sockets, which have best performance (https://bugzilla.redhat.com/show_bug.cgi?id=1653453)
		resources := spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuLimit.Value())
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuRequests.Value())
		}

		spec.Domain.CPU.Sockets = sockets
		spec.Domain.CPU.Cores = cores
		spec.Domain.CPU.Threads = threads
	}
}

func setDefaultCPUModel(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	// create cpu topology struct
	if spec.Domain.CPU == nil {
		spec.Domain.CPU = &v1.CPU{}
	}

	// if vmi doesn't have cpu model set
	if spec.Domain.CPU.Model == "" {
		if clusterConfigCPUModel := clusterConfig.GetCPUModel(); clusterConfigCPUModel != "" {
			//set is as vmi cpu model
			spec.Domain.CPU.Model = clusterConfigCPUModel
		} else {
			spec.Domain.CPU.Model = v1.DefaultCPUModel
		}
	}
}

func setDefaultArchitecture(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) {
	if spec.Architecture == "" {
		spec.Architecture = clusterConfig.GetDefaultArchitecture()
	}
}

func setDefaultArchitectureFromDataSource(clusterConfig *virtconfig.ClusterConfig, vm *v1.VirtualMachine, virtClient kubecli.KubevirtClient) {
	const (
		dataSourceKind        = "datasource"
		templateArchLabel     = "template.kubevirt.io/architecture"
		ignoreFailureErrorFmt = "ignoring failure to find datasource during vm mutation: %v"
		ignoreUnknownArchFmt  = "ignoring unknown architecture %s provided by DataSource %s in namespace %s"
	)
	if vm.Spec.Template.Spec.Architecture != "" {
		return
	}
	for _, template := range vm.Spec.DataVolumeTemplates {
		if template.Spec.SourceRef == nil || !strings.EqualFold(template.Spec.SourceRef.Kind, dataSourceKind) {
			continue
		}
		namespace := vm.Namespace
		templateNamespace := template.Spec.SourceRef.Namespace
		if templateNamespace != nil && *templateNamespace != "" {
			namespace = *templateNamespace
		}
		ds, err := virtClient.CdiClient().CdiV1beta1().DataSources(namespace).Get(
			context.Background(), template.Spec.SourceRef.Name, metav1.GetOptions{})
		if err != nil {
			log.Log.Errorf(ignoreFailureErrorFmt, err)
			continue
		}
		if ds.Spec.Source.DataSource != nil {
			ds, err = virtClient.CdiClient().CdiV1beta1().DataSources(ds.Spec.Source.DataSource.Namespace).Get(
				context.Background(), ds.Spec.Source.DataSource.Name, metav1.GetOptions{})
			if err != nil {
				log.Log.Errorf(ignoreFailureErrorFmt, err)
				continue
			}
		}
		arch, ok := ds.Labels[templateArchLabel]
		if !ok {
			continue
		}
		switch arch {
		case "amd64", "arm64", "s390x":
			vm.Spec.Template.Spec.Architecture = arch
		default:
			log.Log.Warningf(ignoreUnknownArchFmt, arch, ds.Name, ds.Namespace)
			continue
		}
		return
	}
}
