package services

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type ResourceRendererOption func(renderer *ResourceRenderer)

type ResourceRenderer struct {
	vmLimits           k8sv1.ResourceList
	vmRequests         k8sv1.ResourceList
	calculatedLimits   k8sv1.ResourceList
	calculatedRequests k8sv1.ResourceList
}

type resourcePredicate func(*v1.VirtualMachineInstance) bool

type VMIResourcePredicates struct {
	resourceRules []VMIResourceRule
	vmi           *v1.VirtualMachineInstance
}

type VMIResourceRule struct {
	predicate resourcePredicate
	option    ResourceRendererOption
}

func not(p resourcePredicate) resourcePredicate {
	return func(vmi *v1.VirtualMachineInstance) bool {
		return !p(vmi)
	}
}
func NewVMIResourceRule(p resourcePredicate, option ResourceRendererOption) VMIResourceRule {
	return VMIResourceRule{predicate: p, option: option}
}

func doesVMIRequireDedicatedCPU(vmi *v1.VirtualMachineInstance) bool {
	return vmi.IsCPUDedicated()
}

func doesVMIRequireCPUForIOThreads(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.IOThreadsPolicy != nil &&
		*vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool &&
		vmi.Spec.Domain.IOThreads != nil &&
		vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount != nil &&
		*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount > 0
}

func NewResourceRenderer(vmLimits k8sv1.ResourceList, vmRequests k8sv1.ResourceList, options ...ResourceRendererOption) *ResourceRenderer {
	limits := map[k8sv1.ResourceName]resource.Quantity{}
	requests := map[k8sv1.ResourceName]resource.Quantity{}
	copyResources(vmLimits, limits)
	copyResources(vmRequests, requests)

	resourceRenderer := &ResourceRenderer{
		vmLimits:           limits,
		vmRequests:         requests,
		calculatedLimits:   map[k8sv1.ResourceName]resource.Quantity{},
		calculatedRequests: map[k8sv1.ResourceName]resource.Quantity{},
	}

	for _, opt := range options {
		opt(resourceRenderer)
	}
	return resourceRenderer
}

func (rr *ResourceRenderer) Limits() k8sv1.ResourceList {
	podLimits := map[k8sv1.ResourceName]resource.Quantity{}
	copyResources(rr.calculatedLimits, podLimits)
	copyResources(rr.vmLimits, podLimits)
	return podLimits
}

func (rr *ResourceRenderer) Requests() k8sv1.ResourceList {
	podRequests := map[k8sv1.ResourceName]resource.Quantity{}
	copyResources(rr.calculatedRequests, podRequests)
	copyResources(rr.vmRequests, podRequests)
	return podRequests
}

func (rr *ResourceRenderer) ResourceRequirements() k8sv1.ResourceRequirements {
	return k8sv1.ResourceRequirements{
		Limits:   rr.Limits(),
		Requests: rr.Requests(),
	}
}

func WithEphemeralStorageRequest() ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		// Add ephemeral storage request to container to be used by Kubevirt. This amount of ephemeral storage
		// should be added to the user's request.
		ephemeralStorageOverhead := resource.MustParse(ephemeralStorageOverheadSize)
		ephemeralStorageRequested := renderer.vmRequests[k8sv1.ResourceEphemeralStorage]
		ephemeralStorageRequested.Add(ephemeralStorageOverhead)
		renderer.vmRequests[k8sv1.ResourceEphemeralStorage] = ephemeralStorageRequested

		if ephemeralStorageLimit, ephemeralStorageLimitDefined := renderer.vmLimits[k8sv1.ResourceEphemeralStorage]; ephemeralStorageLimitDefined {
			ephemeralStorageLimit.Add(ephemeralStorageOverhead)
			renderer.vmLimits[k8sv1.ResourceEphemeralStorage] = ephemeralStorageLimit
		}
	}
}

func addToCPU(resource map[k8sv1.ResourceName]resource.Quantity, q resource.Quantity) {
	if r, ok := resource[k8sv1.ResourceCPU]; ok {
		r.Add(q)
		resource[k8sv1.ResourceCPU] = r
	} else {
		resource[k8sv1.ResourceCPU] = q
	}
}

func WithoutDedicatedCPU(cpu *v1.CPU, cpuAllocationRatio int, withCPULimits bool) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		vcpus := calcVCPUs(cpu)
		if vcpus != 0 && cpuAllocationRatio > 0 {
			val := float64(vcpus) / float64(cpuAllocationRatio)
			vcpusStr := fmt.Sprintf("%g", val)
			if val < 1 {
				val *= 1000
				vcpusStr = fmt.Sprintf("%gm", val)
			}
			renderer.calculatedRequests[k8sv1.ResourceCPU] = resource.MustParse(vcpusStr)

			if withCPULimits {
				renderer.calculatedLimits[k8sv1.ResourceCPU] = resource.MustParse(strconv.FormatInt(vcpus, 10))
			}
		}
	}
}

func WithIOThreads(iothreads *v1.DiskIOThreads) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		if iothreads == nil || iothreads.SupplementalPoolThreadCount == nil || *iothreads.SupplementalPoolThreadCount < 1 {
			return
		}
		q := resource.NewQuantity(int64(*iothreads.SupplementalPoolThreadCount), resource.BinarySI)
		addToCPU(renderer.vmLimits, *q)
	}
}

func WithHugePages(vmMemory *v1.Memory, memoryOverhead resource.Quantity) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		hugepageType := k8sv1.ResourceName(k8sv1.ResourceHugePagesPrefix + vmMemory.Hugepages.PageSize)
		hugepagesMemReq := renderer.vmRequests.Memory()

		// If requested, use the guest memory to allocate hugepages
		if vmMemory != nil && vmMemory.Guest != nil {
			requests := hugepagesMemReq.Value()
			guest := vmMemory.Guest.Value()
			if requests > guest {
				hugepagesMemReq = vmMemory.Guest
			}
		}
		renderer.calculatedRequests[hugepageType] = *hugepagesMemReq
		renderer.calculatedLimits[hugepageType] = *hugepagesMemReq

		reqMemDiff := resource.NewScaledQuantity(0, resource.Kilo)
		limMemDiff := resource.NewScaledQuantity(0, resource.Kilo)
		// In case the guest memory and the requested memory are different, add the difference
		// to the overhead
		if vmMemory != nil && vmMemory.Guest != nil {
			requests := renderer.vmRequests.Memory().Value()
			limits := renderer.vmLimits.Memory().Value()
			guest := vmMemory.Guest.Value()
			if requests > guest {
				reqMemDiff.Add(*renderer.vmRequests.Memory())
				reqMemDiff.Sub(*vmMemory.Guest)
			}
			if limits > guest {
				limMemDiff.Add(*renderer.vmLimits.Memory())
				limMemDiff.Sub(*vmMemory.Guest)
			}
		}
		// Set requested memory equals to overhead memory
		reqMemDiff.Add(memoryOverhead)
		renderer.vmRequests[k8sv1.ResourceMemory] = *reqMemDiff
		if _, ok := renderer.vmLimits[k8sv1.ResourceMemory]; ok {
			limMemDiff.Add(memoryOverhead)
			renderer.vmLimits[k8sv1.ResourceMemory] = *limMemDiff
		}
	}
}

func WithMemoryOverhead(guestResourceSpec v1.ResourceRequirements, memoryOverhead resource.Quantity) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		memoryRequest := renderer.vmRequests[k8sv1.ResourceMemory]
		if !guestResourceSpec.OvercommitGuestOverhead {
			memoryRequest.Add(memoryOverhead)
		}
		renderer.vmRequests[k8sv1.ResourceMemory] = memoryRequest

		if memoryLimit, ok := renderer.vmLimits[k8sv1.ResourceMemory]; ok {
			memoryLimit.Add(memoryOverhead)
			renderer.vmLimits[k8sv1.ResourceMemory] = memoryLimit
		}
	}
}

func WithAutoMemoryLimits(namespace string, namespaceStore cache.Store) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		requestRatio := getMemoryLimitsRatio(namespace, namespaceStore)
		memoryRequest := renderer.vmRequests[k8sv1.ResourceMemory]
		value := int64(float64(memoryRequest.Value()) * requestRatio)
		renderer.calculatedLimits[k8sv1.ResourceMemory] = *resource.NewQuantity(value, memoryRequest.Format)
	}
}

func WithCPUPinning(cpu *v1.CPU, annotations map[string]string, additionalCPUs uint32) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		vcpus := hardware.GetNumberOfVCPUs(cpu)
		if vcpus != 0 {
			renderer.calculatedLimits[k8sv1.ResourceCPU] = *resource.NewQuantity(vcpus, resource.BinarySI)
		} else {
			if cpuLimit, ok := renderer.vmLimits[k8sv1.ResourceCPU]; ok {
				renderer.vmRequests[k8sv1.ResourceCPU] = cpuLimit
			} else if cpuRequest, ok := renderer.vmRequests[k8sv1.ResourceCPU]; ok {
				renderer.vmLimits[k8sv1.ResourceCPU] = cpuRequest
			}
		}

		// allocate pcpus for emulatorThread if IsolateEmulatorThread is requested
		if cpu.IsolateEmulatorThread {
			emulatorThreadCPUs := resource.NewQuantity(1, resource.BinarySI)

			limits := renderer.calculatedLimits[k8sv1.ResourceCPU]
			_, emulatorThreadCompleteToEvenParityAnnotationExists := annotations[v1.EmulatorThreadCompleteToEvenParity]
			if emulatorThreadCompleteToEvenParityAnnotationExists &&
				(limits.Value()+int64(additionalCPUs))%2 == 0 {
				emulatorThreadCPUs = resource.NewQuantity(2, resource.BinarySI)
			}
			limits.Add(*emulatorThreadCPUs)
			renderer.vmLimits[k8sv1.ResourceCPU] = limits

			if cpuRequest, ok := renderer.vmRequests[k8sv1.ResourceCPU]; ok {
				cpuRequest.Add(*emulatorThreadCPUs)
				renderer.vmRequests[k8sv1.ResourceCPU] = cpuRequest
			}
		}

		renderer.vmLimits[k8sv1.ResourceMemory] = *renderer.vmRequests.Memory()
	}
}

func WithNetworkResources(networkToResourceMap map[string]string) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		resources := renderer.ResourceRequirements()
		for _, resourceName := range networkToResourceMap {
			if resourceName != "" {
				requestResource(&resources, resourceName)
			}
		}
		copyResources(resources.Limits, renderer.calculatedLimits)
		copyResources(resources.Requests, renderer.calculatedRequests)
	}
}

func WithGPUs(gpus []v1.GPU) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		resources := renderer.ResourceRequirements()
		for _, gpu := range gpus {
			requestResource(&resources, gpu.DeviceName)
		}
		copyResources(resources.Limits, renderer.calculatedLimits)
		copyResources(resources.Requests, renderer.calculatedRequests)
	}
}

func WithHostDevices(hostDevices []v1.HostDevice) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		resources := renderer.ResourceRequirements()
		for _, hostDev := range hostDevices {
			requestResource(&resources, hostDev.DeviceName)
		}
		copyResources(resources.Limits, renderer.calculatedLimits)
		copyResources(resources.Requests, renderer.calculatedRequests)
	}
}

func WithSEV() ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		resources := renderer.ResourceRequirements()
		requestResource(&resources, SevDevice)
		copyResources(resources.Limits, renderer.calculatedLimits)
		copyResources(resources.Requests, renderer.calculatedRequests)
	}
}

func WithPersistentReservation() ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		resources := renderer.ResourceRequirements()
		requestResource(&resources, PrDevice)
		copyResources(resources.Limits, renderer.calculatedLimits)
		copyResources(resources.Requests, renderer.calculatedRequests)
	}
}

func copyResources(srcResources, dstResources k8sv1.ResourceList) {
	for key, value := range srcResources {
		dstResources[key] = value
	}
}

// GetMemoryOverhead computes the estimation of total
// memory needed for the domain to operate properly.
// This includes the memory needed for the guest and memory
// for Qemu and OS overhead.
// The return value is overhead memory quantity
//
// Note: This is the best estimation we were able to come up with
//
//	and is still not 100% accurate
func GetMemoryOverhead(vmi *v1.VirtualMachineInstance, cpuArch string, additionalOverheadRatio *string) resource.Quantity {
	domain := vmi.Spec.Domain
	vmiMemoryReq := domain.Resources.Requests.Memory()

	overhead := *resource.NewScaledQuantity(0, resource.Kilo)

	// Add the memory needed for pagetables (one bit for every 512b of RAM size)
	pagetableMemory := resource.NewScaledQuantity(vmiMemoryReq.ScaledValue(resource.Kilo), resource.Kilo)
	pagetableMemory.Set(pagetableMemory.Value() / 512)
	overhead.Add(*pagetableMemory)

	// Add fixed overhead for KubeVirt components, as seen in a random run, rounded up to the nearest MiB
	// Note: shared libraries are included in the size, so every library is counted (wrongly) as many times as there are
	//   processes using it. However, the extra memory is only in the order of 10MiB and makes for a nice safety margin.
	overhead.Add(resource.MustParse(VirtLauncherMonitorOverhead))
	overhead.Add(resource.MustParse(VirtLauncherOverhead))
	overhead.Add(resource.MustParse(VirtlogdOverhead))
	overhead.Add(resource.MustParse(VirtqemudOverhead))
	overhead.Add(resource.MustParse(QemuOverhead))

	// Add CPU table overhead (8 MiB per vCPU and 8 MiB per IO thread)
	// overhead per vcpu in MiB
	coresMemory := resource.MustParse("8Mi")
	var vcpus int64
	if domain.CPU != nil {
		vcpus = hardware.GetNumberOfVCPUs(domain.CPU)
	} else {
		// Currently, a default guest CPU topology is set by the API webhook mutator, if not set by a user.
		// However, this wasn't always the case.
		// In case when the guest topology isn't set, take value from resources request or limits.
		resources := vmi.Spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			vcpus = cpuLimit.Value()
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			vcpus = cpuRequests.Value()
		}
	}

	// if neither CPU topology nor request or limits provided, set vcpus to 1
	if vcpus < 1 {
		vcpus = 1
	}
	value := coresMemory.Value() * vcpus
	coresMemory = *resource.NewQuantity(value, coresMemory.Format)
	overhead.Add(coresMemory)

	// static overhead for IOThread
	overhead.Add(resource.MustParse("8Mi"))

	// Add video RAM overhead
	if domain.Devices.AutoattachGraphicsDevice == nil || *domain.Devices.AutoattachGraphicsDevice == true {
		overhead.Add(resource.MustParse("16Mi"))
	}

	// When use uefi boot on aarch64 with edk2 package, qemu will create 2 pflash(64Mi each, 128Mi in total)
	// it should be considered for memory overhead
	// Additional information can be found here: https://github.com/qemu/qemu/blob/master/hw/arm/virt.c#L120
	if cpuArch == "arm64" {
		overhead.Add(resource.MustParse("128Mi"))
	}

	// Additional overhead of 1G for VFIO devices. VFIO requires all guest RAM to be locked
	// in addition to MMIO memory space to allow DMA. 1G is often the size of reserved MMIO space on x86 systems.
	// Additial information can be found here: https://www.redhat.com/archives/libvir-list/2015-November/msg00329.html
	if util.IsVFIOVMI(vmi) {
		overhead.Add(resource.MustParse("1Gi"))
	}

	// DownardMetrics volumes are using emptyDirs backed by memory.
	// the max. disk size is only 256Ki.
	if downwardmetrics.HasDownwardMetricDisk(vmi) {
		overhead.Add(resource.MustParse("1Mi"))
	}

	addProbeOverheads(vmi, &overhead)

	// Consider memory overhead for SEV guests.
	// Additional information can be found here: https://libvirt.org/kbase/launch_security_sev.html#memory
	if util.IsSEVVMI(vmi) {
		overhead.Add(resource.MustParse("256Mi"))
	}

	// Having a TPM device will spawn a swtpm process
	// In `ps`, swtpm has VSZ of 53808 and RSS of 3496, so 53Mi should do
	if vmi.Spec.Domain.Devices.TPM != nil {
		overhead.Add(resource.MustParse("53Mi"))
	}

	// Multiplying the ratio is expected to be the last calculation before returning overhead
	if additionalOverheadRatio != nil && *additionalOverheadRatio != "" {
		ratio, err := strconv.ParseFloat(*additionalOverheadRatio, 64)
		if err != nil {
			// This error should never happen as it's already validated by webhooks
			log.Log.Warningf("cannot add additional overhead to virt infra overhead calculation: %v", err)
			return overhead
		}

		overhead = multiplyMemory(overhead, ratio)
	}

	if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
		overhead.Add(resource.MustParse("100Mi"))
	}

	return overhead
}

// Request a resource by name. This function bumps the number of resources,
// both its limits and requests attributes.
//
// If we were operating with a regular resource (CPU, memory, network
// bandwidth), we would need to take care of QoS. For example,
// https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed
// explains that when Limits are set but Requests are not then scheduler
// assumes that Requests are the same as Limits for a particular resource.
//
// But this function is not called for this standard resources but for
// resources managed by device plugins. The device plugin design document says
// the following on the matter:
// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/resource-management/device-plugin.md#end-user-story
//
// ```
// Devices can be selected using the same process as for OIRs in the pod spec.
// Devices have no impact on QOS. However, for the alpha, we expect the request
// to have limits == requests.
// ```
//
// Which suggests that, for resources managed by device plugins, 1) limits
// should be equal to requests; and 2) QoS rules do not apVFIO//
// Hence we don't copy Limits value to Requests if the latter is missing.
func requestResource(resources *k8sv1.ResourceRequirements, resourceName string) {
	name := k8sv1.ResourceName(resourceName)
	bumpResources(resources.Limits, name)
	bumpResources(resources.Requests, name)
}

func bumpResources(resources k8sv1.ResourceList, name k8sv1.ResourceName) {
	unitQuantity := *resource.NewQuantity(1, resource.DecimalSI)

	val, ok := resources[name]
	if ok {
		val.Add(unitQuantity)
		resources[name] = val
	} else {
		resources[name] = unitQuantity
	}
}

func calcVCPUs(cpu *v1.CPU) int64 {
	if cpu != nil {
		return hardware.GetNumberOfVCPUs(cpu)
	}
	return int64(1)
}

func getRequiredResources(vmi *v1.VirtualMachineInstance, allowEmulation bool) k8sv1.ResourceList {
	res := k8sv1.ResourceList{}
	if util.NeedTunDevice(vmi) {
		res[TunDevice] = resource.MustParse("1")
	}
	if needVirtioNetDevice(vmi, allowEmulation) {
		// Note that about network interface, allowEmulation does not make
		// any difference on eventual Domain xml, but uniformly making
		// /dev/vhost-net unavailable and libvirt implicitly fallback
		// to use QEMU userland NIC emulation.
		res[VhostNetDevice] = resource.MustParse("1")
	}
	if !allowEmulation {
		res[KvmDevice] = resource.MustParse("1")
	}
	if util.IsAutoAttachVSOCK(vmi) {
		res[VhostVsockDevice] = resource.MustParse("1")
	}
	return res
}

func WithVirtualizationResources(virtResources k8sv1.ResourceList) ResourceRendererOption {
	return func(renderer *ResourceRenderer) {
		copyResources(virtResources, renderer.vmLimits)
	}
}

func validatePermittedHostDevices(spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) error {
	errors := make([]string, 0)

	if hostDevs := config.GetPermittedHostDevices(); hostDevs != nil {
		// build a map of all permitted host devices
		supportedHostDevicesMap := make(map[string]bool)
		for _, dev := range hostDevs.PciHostDevices {
			supportedHostDevicesMap[dev.ResourceName] = true
		}
		for _, dev := range hostDevs.MediatedDevices {
			supportedHostDevicesMap[dev.ResourceName] = true
		}
		for _, dev := range hostDevs.USB {
			supportedHostDevicesMap[dev.ResourceName] = true
		}
		for _, hostDev := range spec.Domain.Devices.GPUs {
			if _, exist := supportedHostDevicesMap[hostDev.DeviceName]; !exist {
				errors = append(errors, fmt.Sprintf("GPU %s is not permitted in permittedHostDevices configuration", hostDev.DeviceName))
			}
		}
		for _, hostDev := range spec.Domain.Devices.HostDevices {
			if _, exist := supportedHostDevicesMap[hostDev.DeviceName]; !exist {
				errors = append(errors, fmt.Sprintf("HostDevice %s is not permitted in permittedHostDevices configuration", hostDev.DeviceName))
			}
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf(strings.Join(errors, " "))
	}

	return nil
}

func sidecarResources(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig) k8sv1.ResourceRequirements {
	resources := k8sv1.ResourceRequirements{
		Requests: k8sv1.ResourceList{},
		Limits:   k8sv1.ResourceList{},
	}
	if reqCpu := config.GetSupportContainerRequest(v1.SideCar, k8sv1.ResourceCPU); reqCpu != nil {
		resources.Requests[k8sv1.ResourceCPU] = *reqCpu
	}
	if reqMem := config.GetSupportContainerRequest(v1.SideCar, k8sv1.ResourceMemory); reqMem != nil {
		resources.Requests[k8sv1.ResourceMemory] = *reqMem
	}

	// add default cpu and memory limits to enable cpu pinning if requested
	// TODO(vladikr): make the hookSidecar express resources
	if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
		resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("200m")
		if limCpu := config.GetSupportContainerLimit(v1.SideCar, k8sv1.ResourceCPU); limCpu != nil {
			resources.Limits[k8sv1.ResourceCPU] = *limCpu
		}
		resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("64M")
		if limMem := config.GetSupportContainerLimit(v1.SideCar, k8sv1.ResourceMemory); limMem != nil {
			resources.Limits[k8sv1.ResourceMemory] = *limMem
		}
		resources.Requests[k8sv1.ResourceCPU] = resources.Limits[k8sv1.ResourceCPU]
		resources.Requests[k8sv1.ResourceMemory] = resources.Limits[k8sv1.ResourceMemory]
	} else {
		if limCpu := config.GetSupportContainerLimit(v1.SideCar, k8sv1.ResourceCPU); limCpu != nil {
			resources.Limits[k8sv1.ResourceCPU] = *limCpu
		}
		if limMem := config.GetSupportContainerLimit(v1.SideCar, k8sv1.ResourceMemory); limMem != nil {
			resources.Limits[k8sv1.ResourceMemory] = *limMem
		}
	}
	return resources
}

func initContainerResourceRequirementsForVMI(vmi *v1.VirtualMachineInstance, containerType v1.SupportContainerType, config *virtconfig.ClusterConfig) k8sv1.ResourceRequirements {
	if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
		return k8sv1.ResourceRequirements{
			Limits:   initContainerDedicatedCPURequiredResources(containerType, config),
			Requests: initContainerDedicatedCPURequiredResources(containerType, config),
		}
	} else {
		return k8sv1.ResourceRequirements{
			Limits:   initContainerMinimalLimits(containerType, config),
			Requests: initContainerMinimalRequests(containerType, config),
		}
	}
}

func initContainerDedicatedCPURequiredResources(containerType v1.SupportContainerType, config *virtconfig.ClusterConfig) k8sv1.ResourceList {
	res := k8sv1.ResourceList{
		k8sv1.ResourceCPU:    resource.MustParse("10m"),
		k8sv1.ResourceMemory: resource.MustParse("40M"),
	}
	if cpuLim := config.GetSupportContainerLimit(containerType, k8sv1.ResourceCPU); cpuLim != nil {
		res[k8sv1.ResourceCPU] = *cpuLim
	}
	if memLim := config.GetSupportContainerLimit(containerType, k8sv1.ResourceMemory); memLim != nil {
		res[k8sv1.ResourceMemory] = *memLim
	}
	return res
}

func initContainerMinimalLimits(containerType v1.SupportContainerType, config *virtconfig.ClusterConfig) k8sv1.ResourceList {
	res := k8sv1.ResourceList{
		k8sv1.ResourceCPU:    resource.MustParse("100m"),
		k8sv1.ResourceMemory: resource.MustParse("40M"),
	}
	if cpuLim := config.GetSupportContainerLimit(containerType, k8sv1.ResourceCPU); cpuLim != nil {
		res[k8sv1.ResourceCPU] = *cpuLim
	}
	if memLim := config.GetSupportContainerLimit(containerType, k8sv1.ResourceMemory); memLim != nil {
		res[k8sv1.ResourceMemory] = *memLim
	}
	return res
}

func initContainerMinimalRequests(containerType v1.SupportContainerType, config *virtconfig.ClusterConfig) k8sv1.ResourceList {
	res := k8sv1.ResourceList{
		k8sv1.ResourceCPU:    resource.MustParse("10m"),
		k8sv1.ResourceMemory: resource.MustParse("1M"),
	}
	if cpuReq := config.GetSupportContainerRequest(containerType, k8sv1.ResourceCPU); cpuReq != nil {
		res[k8sv1.ResourceCPU] = *cpuReq
	}
	if memReq := config.GetSupportContainerRequest(containerType, k8sv1.ResourceMemory); memReq != nil {
		res[k8sv1.ResourceMemory] = *memReq
	}
	return res
}

func hotplugContainerResourceRequirementsForVMI(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig) k8sv1.ResourceRequirements {
	return k8sv1.ResourceRequirements{
		Limits:   hotplugContainerLimits(config),
		Requests: hotplugContainerRequests(config),
	}
}

func hotplugContainerLimits(config *virtconfig.ClusterConfig) k8sv1.ResourceList {
	cpuQuantity := resource.MustParse("100m")
	if cpu := config.GetSupportContainerLimit(v1.HotplugAttachment, k8sv1.ResourceCPU); cpu != nil {
		cpuQuantity = *cpu
	}
	memQuantity := resource.MustParse("80M")
	if mem := config.GetSupportContainerLimit(v1.HotplugAttachment, k8sv1.ResourceMemory); mem != nil {
		memQuantity = *mem
	}
	return k8sv1.ResourceList{
		k8sv1.ResourceCPU:    cpuQuantity,
		k8sv1.ResourceMemory: memQuantity,
	}
}

func hotplugContainerRequests(config *virtconfig.ClusterConfig) k8sv1.ResourceList {
	cpuQuantity := resource.MustParse("10m")
	if cpu := config.GetSupportContainerRequest(v1.HotplugAttachment, k8sv1.ResourceCPU); cpu != nil {
		cpuQuantity = *cpu
	}
	memQuantity := resource.MustParse("2M")
	if mem := config.GetSupportContainerRequest(v1.HotplugAttachment, k8sv1.ResourceMemory); mem != nil {
		memQuantity = *mem
	}
	return k8sv1.ResourceList{
		k8sv1.ResourceCPU:    cpuQuantity,
		k8sv1.ResourceMemory: memQuantity,
	}
}

func vmExportContainerResourceRequirements(config *virtconfig.ClusterConfig) k8sv1.ResourceRequirements {
	return k8sv1.ResourceRequirements{
		Limits:   vmExportContainerLimits(config),
		Requests: vmExportContainerRequests(config),
	}
}

func vmExportContainerLimits(config *virtconfig.ClusterConfig) k8sv1.ResourceList {
	cpuQuantity := resource.MustParse("1")
	if cpu := config.GetSupportContainerLimit(v1.VMExport, k8sv1.ResourceCPU); cpu != nil {
		cpuQuantity = *cpu
	}
	memQuantity := resource.MustParse("1024Mi")
	if mem := config.GetSupportContainerLimit(v1.VMExport, k8sv1.ResourceMemory); mem != nil {
		memQuantity = *mem
	}
	return k8sv1.ResourceList{
		k8sv1.ResourceCPU:    cpuQuantity,
		k8sv1.ResourceMemory: memQuantity,
	}
}

func vmExportContainerRequests(config *virtconfig.ClusterConfig) k8sv1.ResourceList {
	cpuQuantity := resource.MustParse("100m")
	if cpu := config.GetSupportContainerRequest(v1.VMExport, k8sv1.ResourceCPU); cpu != nil {
		cpuQuantity = *cpu
	}
	memQuantity := resource.MustParse("200Mi")
	if mem := config.GetSupportContainerRequest(v1.VMExport, k8sv1.ResourceMemory); mem != nil {
		memQuantity = *mem
	}
	return k8sv1.ResourceList{
		k8sv1.ResourceCPU:    cpuQuantity,
		k8sv1.ResourceMemory: memQuantity,
	}
}

func multiplyMemory(mem resource.Quantity, multiplication float64) resource.Quantity {
	overheadAddition := float64(mem.ScaledValue(resource.Kilo)) * (multiplication - 1.0)
	additionalOverhead := resource.NewScaledQuantity(int64(overheadAddition), resource.Kilo)

	mem.Add(*additionalOverhead)
	return mem
}

func getMemoryLimitsRatio(namespace string, namespaceStore cache.Store) float64 {
	if namespaceStore == nil {
		return DefaultMemoryLimitOverheadRatio
	}

	obj, exists, err := namespaceStore.GetByKey(namespace)
	if err != nil {
		log.Log.Warningf("Error retrieving namespace from informer. Using the default memory limits ratio. %s", err.Error())
		return DefaultMemoryLimitOverheadRatio
	} else if !exists {
		log.Log.Warningf("namespace %s does not exist. Using the default memory limits ratio.", namespace)
		return DefaultMemoryLimitOverheadRatio
	}

	ns, ok := obj.(*k8sv1.Namespace)
	if !ok {
		log.Log.Errorf("couldn't cast object to Namespace: %+v", obj)
		return DefaultMemoryLimitOverheadRatio
	}

	value, ok := ns.GetLabels()[v1.AutoMemoryLimitsRatioLabel]
	if !ok {
		return DefaultMemoryLimitOverheadRatio
	}

	limitRatioValue, err := strconv.ParseFloat(value, 64)
	if err != nil || limitRatioValue < 1.0 {
		log.Log.Warningf("%s is an invalid value for %s label in namespace %s. Using the default one: %f", value, v1.AutoMemoryLimitsRatioLabel, namespace, DefaultMemoryLimitOverheadRatio)
		return DefaultMemoryLimitOverheadRatio
	}

	return limitRatioValue
}

// needVirtioNetDevice checks whether a VMI requires the presence of the "virtio" net device.
// This happens when the VMI wants to use a "virtio" network interface, and software emulation is disallowed.
func needVirtioNetDevice(vmi *v1.VirtualMachineInstance, allowEmulation bool) bool {
	return util.WantVirtioNetDevice(vmi) && !allowEmulation
}
