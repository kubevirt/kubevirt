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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package virtwrap

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	eventsclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/cache"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	libvirt "libvirt.org/libvirt-go"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/ignition"
	kutil "kubevirt.io/kubevirt/pkg/util"
	accesscredentials "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/access-credentials"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/sriov"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const (
	LibvirtLocalConnectionPort = 22222
	gpuEnvPrefix               = "GPU_PASSTHROUGH_DEVICES"
	vgpuEnvPrefix              = "VGPU_PASSTHROUGH_DEVICES"
	PCI_RESOURCE_PREFIX        = "PCI_RESOURCE"
	MDEV_RESOURCE_PREFIX       = "MDEV_PCI_RESOURCE"
)

type contextStore struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type DomainManager interface {
	SyncVMI(*v1.VirtualMachineInstance, bool, *cmdv1.VirtualMachineOptions) (*api.DomainSpec, error)
	PauseVMI(*v1.VirtualMachineInstance) error
	UnpauseVMI(*v1.VirtualMachineInstance) error
	KillVMI(*v1.VirtualMachineInstance) error
	DeleteVMI(*v1.VirtualMachineInstance) error
	SignalShutdownVMI(*v1.VirtualMachineInstance) error
	MarkGracefulShutdownVMI(*v1.VirtualMachineInstance) error
	ListAllDomains() ([]*api.Domain, error)
	MigrateVMI(*v1.VirtualMachineInstance, *cmdclient.MigrationOptions) error
	PrepareMigrationTarget(*v1.VirtualMachineInstance, bool) error
	GetDomainStats() ([]*stats.DomainStats, error)
	CancelVMIMigration(*v1.VirtualMachineInstance) error
	GetGuestInfo() (v1.VirtualMachineInstanceGuestAgentInfo, error)
	GetUsers() ([]v1.VirtualMachineInstanceGuestOSUser, error)
	GetFilesystems() ([]v1.VirtualMachineInstanceFileSystem, error)
	FinalizeVirtualMachineMigration(*v1.VirtualMachineInstance) error
}

type LibvirtDomainManager struct {
	virConn cli.Connection

	// Anytime a get and a set is done on the domain, this lock must be held.
	domainModifyLock sync.Mutex
	// mutex to control access to the guest time context
	setGuestTimeLock sync.Mutex

	credManager *accesscredentials.AccessCredentialManager

	virtShareDir             string
	notifier                 *eventsclient.Notifier
	lessPVCSpaceToleration   int
	paused                   pausedVMIs
	agentData                *agentpoller.AsyncAgentStore
	cloudInitDataStore       *cloudinit.CloudInitData
	setGuestTimeContextPtr   *contextStore
	ovmfPath                 string
	networkCacheStoreFactory cache.InterfaceCacheFactory
}

type hostDeviceTypePrefix struct {
	Type   converter.HostDeviceType
	Prefix string
}

type pausedVMIs struct {
	paused map[types.UID]bool
}

func (s pausedVMIs) add(uid types.UID) {
	// implicitly locked by domainModifyLock
	if _, ok := s.paused[uid]; !ok {
		s.paused[uid] = true
	}
}

func (s pausedVMIs) remove(uid types.UID) {
	// implicitly locked by domainModifyLock
	if _, ok := s.paused[uid]; ok {
		delete(s.paused, uid)
	}
}

func (s pausedVMIs) contains(uid types.UID) bool {
	_, ok := s.paused[uid]
	return ok
}

func NewLibvirtDomainManager(connection cli.Connection, virtShareDir string, notifier *eventsclient.Notifier, lessPVCSpaceToleration int, agentStore *agentpoller.AsyncAgentStore, ovmfPath string) (DomainManager, error) {
	manager := LibvirtDomainManager{
		virConn:                connection,
		virtShareDir:           virtShareDir,
		notifier:               notifier,
		lessPVCSpaceToleration: lessPVCSpaceToleration,
		paused: pausedVMIs{
			paused: make(map[types.UID]bool, 0),
		},
		agentData:                agentStore,
		ovmfPath:                 ovmfPath,
		networkCacheStoreFactory: cache.NewInterfaceCacheFactory(),
	}
	manager.credManager = accesscredentials.NewManager(connection, &manager.domainModifyLock)

	return &manager, nil
}

func getAllDomainDevices(dom cli.VirDomain) (api.Devices, error) {
	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return api.Devices{}, err
	}
	var newSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &newSpec)
	if err != nil {
		return api.Devices{}, err
	}

	return newSpec.Devices, nil
}

func getAllDomainDisks(dom cli.VirDomain) ([]api.Disk, error) {
	devices, err := getAllDomainDevices(dom)
	if err != nil {
		return nil, err
	}

	return devices.Disks, nil
}

func (l *LibvirtDomainManager) setGuestTime(vmi *v1.VirtualMachineInstance) error {
	// Try to set VM time to the current value.  This is typically useful
	// when clock wasn't running on the VM for some time (e.g. during
	// suspension or migration), especially if the time delay exceeds NTP
	// tolerance.
	// It is not guaranteed that the time is actually set (it depends on guest
	// environment, especially QEMU agent presence) or that the set time is
	// very precise (NTP in the guest should take care of it if needed).

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to sync guest time")
		return err
	}

	go func() {
		defer dom.Free()

		ctx := l.getGuestTimeContext()
		timeout := time.After(60 * time.Second)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-timeout:
				log.Log.Object(vmi).Error("failed to sync guest time")
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				currTime := time.Now()
				secs := currTime.Unix()
				nsecs := uint(currTime.Nanosecond())
				err := dom.SetTime(secs, nsecs, 0)
				if err != nil {
					libvirtError, ok := err.(libvirt.Error)
					if !ok {
						log.Log.Object(vmi).Reason(err).Warning("failed to sync guest time")
						return
					}

					switch libvirtError.Code {
					case libvirt.ERR_AGENT_UNRESPONSIVE:
						log.Log.Object(vmi).Reason(err).Warning("failed to set time: QEMU agent unresponsive")
					case libvirt.ERR_OPERATION_UNSUPPORTED:
						// no need to retry as this opertaion is not supported
						log.Log.Object(vmi).Reason(err).Warning("failed to set time: not supported")
						return
					case libvirt.ERR_ARGUMENT_UNSUPPORTED:
						// no need to retry as the agent is not configured
						log.Log.Object(vmi).Reason(err).Warning("failed to set time: agent not configured")
						return
					default:
						log.Log.Object(vmi).Reason(err).Warning("failed to sync guest time")
					}
				} else {
					log.Log.Object(vmi).Info("guest VM time sync finished successfully")
					return
				}
			}
		}
	}()

	return nil
}

func (l *LibvirtDomainManager) getGuestTimeContext() context.Context {
	l.setGuestTimeLock.Lock()
	defer l.setGuestTimeLock.Unlock()

	// cancel the already running setGuestTime go-routine if such exist
	if l.setGuestTimeContextPtr != nil {
		l.setGuestTimeContextPtr.cancel()
	}
	// create a new context and store it
	ctx, cancel := context.WithCancel(context.Background())
	l.setGuestTimeContextPtr = &contextStore{ctx: ctx, cancel: cancel}
	return ctx
}

// PrepareMigrationTarget the target pod environment before the migration is initiated
func (l *LibvirtDomainManager) PrepareMigrationTarget(vmi *v1.VirtualMachineInstance, useEmulation bool) error {
	return l.prepareMigrationTarget(vmi, useEmulation)
}

// FinalizeVirtualMachineMigration finalized the migration after the migration has completed and vmi is running on target pod.
func (l *LibvirtDomainManager) FinalizeVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error {
	return l.finalizeMigrationTarget(vmi)
}

// hotPlugHostDevices attach host-devices to running domain
// Currently only SRIOV host-devices are supported
func (l *LibvirtDomainManager) hotPlugHostDevices(vmi *v1.VirtualMachineInstance) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domainName := api.VMINamespaceKeyFunc(vmi)
	domain, err := l.virConn.LookupDomainByName(domainName)
	if err != nil {
		return err
	}
	defer domain.Free()

	domainSpec, err := util.GetDomainSpecWithFlags(domain, 0)
	if err != nil {
		return err
	}

	sriovHostDevices, err := sriov.GetHostDevicesToAttach(vmi, domainSpec)
	if err != nil {
		return err
	}

	if err := sriov.AttachHostDevices(domain, sriovHostDevices); err != nil {
		return err
	}

	return nil
}

func getVMIEphemeralDisksTotalSize() *resource.Quantity {
	var baseDir = "/var/run/kubevirt-ephemeral-disks/"
	totalSize := int64(0)
	err := filepath.Walk(baseDir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			totalSize += f.Size()
		}
		return err
	})
	if err != nil {
		log.Log.Reason(err).Warning("failed to get VMI ephemeral disks size")
		return &resource.Quantity{Format: resource.BinarySI}
	}

	return resource.NewScaledQuantity(totalSize, 0)
}

func getVMIMigrationDataSize(vmi *v1.VirtualMachineInstance) int64 {
	var memory resource.Quantity

	// Take memory from the requested memory
	if v, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
		memory = v
	}
	// In case that guest memory is explicitly set, override it
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		memory = *vmi.Spec.Domain.Memory.Guest
	}

	//get total data Size
	if vmi.Status.MigrationMethod == v1.BlockMigration {
		disksSize := getVMIEphemeralDisksTotalSize()
		memory.Add(*disksSize)
	}
	return memory.ScaledValue(resource.Giga)
}

func (l *LibvirtDomainManager) CancelVMIMigration(vmi *v1.VirtualMachineInstance) error {
	return l.cancelMigration(vmi)
}

func (l *LibvirtDomainManager) MigrateVMI(vmi *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) error {
	return l.startMigration(vmi, options)
}

var updateHostsFile = func(entry string) (err error) {
	file, err := kutil.OpenFileWithNosec("/etc/hosts", os.O_WRONLY|os.O_APPEND)
	if err != nil {
		return fmt.Errorf("failed opening file: %s", err)
	}
	defer kutil.CloseIOAndCheckErr(file, &err)

	_, err = file.WriteString(entry)
	if err != nil {
		return fmt.Errorf("failed writing to file: %s", err)
	}
	return nil
}

func (l *LibvirtDomainManager) generateCloudInitISO(vmi *v1.VirtualMachineInstance, domPtr *cli.VirDomain) error {
	var devicesMetadata []cloudinit.DeviceData
	// this is the point where we need to build the devices metadata if it was requested.
	// This metadata maps the user provided tag to the hypervisor assigned device address.
	if domPtr != nil {
		data, err := l.buildDevicesMetadata(vmi, *domPtr)
		if err != nil {
			return err
		}
		devicesMetadata = data
	}
	// build condif drive iso file, that includes devices metadata if available
	// get stored cloud init data
	var cloudInitDataStore *cloudinit.CloudInitData
	cloudInitDataStore = l.cloudInitDataStore

	if cloudInitDataStore != nil {
		// add devices metedata
		if devicesMetadata != nil {
			cloudInitDataStore.DevicesData = &devicesMetadata
		}
		err := cloudinit.GenerateLocalData(vmi.Name, vmi.Namespace, cloudInitDataStore)
		if err != nil {
			return fmt.Errorf("generating local cloud-init data failed: %v", err)
		}
	}
	return nil
}

// All local environment setup that needs to occur before VirtualMachineInstance starts
// can be done in this function. This includes things like...
//
// - storage prep
// - network prep
// - cloud-init
// - sysprep
//
// The Domain.Spec can be alterned in this function and any changes
// made to the domain will get set in libvirt after this function exits.
func (l *LibvirtDomainManager) preStartHook(vmi *v1.VirtualMachineInstance, domain *api.Domain) (*api.Domain, error) {

	logger := log.Log.Object(vmi)

	logger.Info("Executing PreStartHook on VMI pod environment")

	// generate cloud-init data
	cloudInitData, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, config.SecretSourceDir)
	if err != nil {
		return domain, fmt.Errorf("PreCloudInitIso hook failed: %v", err)
	}

	// Pass cloud-init data to PreCloudInitIso hook
	logger.Info("Starting PreCloudInitIso hook")
	hooksManager := hooks.GetManager()
	cloudInitData, err = hooksManager.PreCloudInitIso(vmi, cloudInitData)
	if err != nil {
		return domain, fmt.Errorf("PreCloudInitIso hook failed: %v", err)
	}

	if cloudInitData != nil {
		// store the generated cloud init metadata.
		// cloud init ISO will be generated after the domain definition
		l.cloudInitDataStore = cloudInitData
	}

	// generate ignition data
	ignitionData := ignition.GetIgnitionSource(vmi)
	if ignitionData != "" {

		err := ignition.GenerateIgnitionLocalData(vmi, vmi.Namespace)
		if err != nil {
			return domain, err
		}
	}

	err = network.NewVMNetworkConfigurator(vmi, l.networkCacheStoreFactory).SetupPodNetworkPhase2(domain)
	if err != nil {
		return domain, fmt.Errorf("preparing the pod network failed: %v", err)
	}

	// create disks images on the cluster lever
	// or initialize disks images for empty PVC
	hostDiskCreator := hostdisk.NewHostDiskCreator(l.notifier, l.lessPVCSpaceToleration)
	err = hostDiskCreator.Create(vmi)
	if err != nil {
		return domain, fmt.Errorf("preparing host-disks failed: %v", err)
	}

	// Create ephemeral disk for container disks
	err = containerdisk.CreateEphemeralImages(vmi)
	if err != nil {
		return domain, fmt.Errorf("preparing ephemeral container disk images failed: %v", err)
	}
	// Create images for volumes that are marked ephemeral.
	err = ephemeraldisk.CreateEphemeralImages(vmi)
	if err != nil {
		return domain, fmt.Errorf("preparing ephemeral images failed: %v", err)
	}
	// create empty disks if they exist
	if err := emptydisk.CreateTemporaryDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating empty disks failed: %v", err)
	}
	// create ConfigMap disks if they exists
	if err := config.CreateConfigMapDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating config map disks failed: %v", err)
	}
	// create Secret disks if they exists
	if err := config.CreateSecretDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating secret disks failed: %v", err)
	}

	// create Sysprep disks if they exists
	if err := config.CreateSysprepDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating sysprep disks failed: %v", err)
	}

	// create DownwardAPI disks if they exists
	if err := config.CreateDownwardAPIDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating DownwardAPI disks failed: %v", err)
	}
	// create ServiceAccount disk if exists
	if err := config.CreateServiceAccountDisk(vmi); err != nil {
		return domain, fmt.Errorf("creating service account disk failed: %v", err)
	}

	// set drivers cache mode
	for i := range domain.Spec.Devices.Disks {
		err := converter.SetDriverCacheMode(&domain.Spec.Devices.Disks[i])
		if err != nil {
			return domain, err
		}
		converter.SetOptimalIOMode(&domain.Spec.Devices.Disks[i])
	}

	if err := l.credManager.HandleQemuAgentAccessCredentials(vmi); err != nil {
		return domain, fmt.Errorf("Starting qemu agent access credential propagation failed: %v", err)
	}

	return domain, err
}

// This function parses variables that are set by SR-IOV device plugin listing
// PCI IDs for devices allocated to the pod. It also parses variables that
// virt-controller sets mapping network names to their respective resource
// names (if any).
//
// Format for PCI ID variables set by SR-IOV DP is:
// "": for no allocated devices
// PCIDEVICE_<resourceName>="0000:81:11.1": for a single device
// PCIDEVICE_<resourceName>="0000:81:11.1 0000:81:11.2[ ...]": for multiple devices
//
// Since special characters in environment variable names are not allowed,
// resourceName is mutated as follows:
// 1. All dots and slashes are replaced with underscore characters.
// 2. The result is upper cased.
//
// Example: PCIDEVICE_INTEL_COM_SRIOV_TEST=... for intel.com/sriov_test resources.
//
// Format for network to resource mapping variables is:
// KUBEVIRT_RESOURCE_NAME_<networkName>=<resourceName>
//
func updateDeviceResourcesMap(supportedDevice hostDeviceTypePrefix, resourceToAddressesMap map[string]converter.HostDevicesList, resourceName string) {
	varName := kutil.ResourceNameToEnvVar(supportedDevice.Prefix, resourceName)
	addrString, isSet := os.LookupEnv(varName)
	if isSet {
		addrs := parseDeviceAddress(addrString)
		device := converter.HostDevicesList{
			Type:     supportedDevice.Type,
			AddrList: addrs,
		}
		resourceToAddressesMap[resourceName] = device
	} else {
		log.DefaultLogger().Warningf("%s not set for device %s", varName, resourceName)
	}
}

// There is an overlap between HostDevices and GPUs. Both can provide PCI devices and MDEVs
// However, both will be mapped to a hostdev struct with some differences.
func getDevicesForAssignment(devices v1.Devices) map[string]converter.HostDevicesList {
	supportedHostDeviceTypes := []hostDeviceTypePrefix{
		{
			Type:   converter.HostDevicePCI,
			Prefix: PCI_RESOURCE_PREFIX,
		},
		{
			Type:   converter.HostDeviceMDEV,
			Prefix: MDEV_RESOURCE_PREFIX,
		},
	}
	resourceToAddressesMap := make(map[string]converter.HostDevicesList)

	for _, supportedHostDeviceType := range supportedHostDeviceTypes {
		for _, hostDev := range devices.HostDevices {
			updateDeviceResourcesMap(
				supportedHostDeviceType,
				resourceToAddressesMap,
				hostDev.DeviceName,
			)
		}
		for _, gpu := range devices.GPUs {
			updateDeviceResourcesMap(
				supportedHostDeviceType,
				resourceToAddressesMap,
				gpu.DeviceName,
			)
		}
	}
	return resourceToAddressesMap

}

// This function parses all environment variables with prefix string that is set by a Device Plugin.
// Device plugin that passes GPU devices by setting these env variables is https://github.com/NVIDIA/kubevirt-gpu-device-plugin
// It returns address list for devices set in the env variable.
// The format is as follows:
// "":for no address set
// "<address_1>,": for a single address
// "<address_1>,<address_2>[,...]": for multiple addresses
func getEnvAddressListByPrefix(evnPrefix string) []string {
	var returnAddr []string
	for _, env := range os.Environ() {
		split := strings.Split(env, "=")
		if strings.HasPrefix(split[0], evnPrefix) {
			returnAddr = append(returnAddr, parseDeviceAddress(split[1])...)
		}
	}
	return returnAddr
}

func parseDeviceAddress(addrString string) []string {
	addrs := strings.Split(addrString, ",")
	naddrs := len(addrs)
	if naddrs > 0 {
		if addrs[naddrs-1] == "" {
			addrs = addrs[:naddrs-1]
		}
	}

	for index, element := range addrs {
		addrs[index] = strings.TrimSpace(element)
	}
	return addrs
}

func (l *LibvirtDomainManager) generateConverterContext(vmi *v1.VirtualMachineInstance, useEmulation bool, options *cmdv1.VirtualMachineOptions, isMigrationTarget bool) (*converter.ConverterContext, error) {

	logger := log.Log.Object(vmi)

	var emulatorThreadCpu *int
	podCPUSet, err := util.GetPodCPUSet()
	if err != nil {
		logger.Reason(err).Error("failed to read pod cpuset.")
		return nil, fmt.Errorf("failed to read pod cpuset: %v", err)
	}
	// reserve the last cpu for the emulator thread
	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		if len(podCPUSet) > 0 {
			emulatorThreadCpu = &podCPUSet[len(podCPUSet)-1]
			podCPUSet = podCPUSet[:len(podCPUSet)-1]
		}
	}

	hotplugVolumes := make(map[string]v1.VolumeStatus)
	permanentVolumes := make(map[string]v1.VolumeStatus)
	for _, status := range vmi.Status.VolumeStatus {
		if status.HotplugVolume != nil {
			hotplugVolumes[status.Name] = status
		} else {
			permanentVolumes[status.Name] = status
		}
	}

	// Check if PVC volumes are block volumes
	isBlockPVCMap := make(map[string]bool)
	isBlockDVMap := make(map[string]bool)
	diskInfo := make(map[string]*containerdisk.DiskInfo)
	for i, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			isBlockPVC := false
			if _, ok := hotplugVolumes[volume.Name]; ok {
				isBlockPVC = isHotplugBlockDeviceVolume(volume.Name)
			} else {
				isBlockPVC, _ = isBlockDeviceVolume(volume.Name)
			}
			isBlockPVCMap[volume.Name] = isBlockPVC
		} else if volume.VolumeSource.ContainerDisk != nil {
			image, err := containerdisk.GetDiskTargetPartFromLauncherView(i)
			if err != nil {
				return nil, err
			}
			info, err := converter.GetImageInfo(image)
			if err != nil {
				return nil, err
			}
			diskInfo[volume.Name] = info
		} else if volume.VolumeSource.DataVolume != nil {
			isBlockDV := false
			if _, ok := hotplugVolumes[volume.Name]; ok {
				isBlockDV = isHotplugBlockDeviceVolume(volume.Name)
			} else {
				isBlockDV, _ = isBlockDeviceVolume(volume.Name)
			}
			isBlockDVMap[volume.Name] = isBlockDV
		}
	}

	// Map the VirtualMachineInstance to the Domain
	c := &converter.ConverterContext{
		Architecture:          runtime.GOARCH,
		VirtualMachine:        vmi,
		UseEmulation:          useEmulation,
		CPUSet:                podCPUSet,
		IsBlockPVC:            isBlockPVCMap,
		IsBlockDV:             isBlockDVMap,
		DiskType:              diskInfo,
		EmulatorThreadCpu:     emulatorThreadCpu,
		OVMFPath:              l.ovmfPath,
		UseVirtioTransitional: vmi.Spec.Domain.Devices.UseVirtioTransitional != nil && *vmi.Spec.Domain.Devices.UseVirtioTransitional,
		PermanentVolumes:      permanentVolumes,
	}

	if options != nil {
		if options.VirtualMachineSMBios != nil {
			c.SMBios = options.VirtualMachineSMBios
		}
		c.MemBalloonStatsPeriod = uint(options.MemBalloonStatsPeriod)
		// Add preallocated and thick-provisioned volumes for which we need to avoid the discard=unmap option
		c.VolumesDiscardIgnore = options.PreallocatedVolumes
		// Disk iotune configuration
		c.DiskIoTune = options.DiskIoTune
	}

	if !isMigrationTarget {
		sriovDevices, err := sriov.CreateHostDevices(vmi)
		if err != nil {
			return nil, err
		}

		c.HotplugVolumes = hotplugVolumes
		c.SRIOVDevices = sriovDevices
		c.GpuDevices = getEnvAddressListByPrefix(gpuEnvPrefix)
		c.VgpuDevices = getEnvAddressListByPrefix(vgpuEnvPrefix)
		c.HostDevices = getDevicesForAssignment(vmi.Spec.Domain.Devices)
	}

	return c, nil
}

func (l *LibvirtDomainManager) SyncVMI(vmi *v1.VirtualMachineInstance, useEmulation bool, options *cmdv1.VirtualMachineOptions) (*api.DomainSpec, error) {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	logger := log.Log.Object(vmi)

	domain := &api.Domain{}

	c, err := l.generateConverterContext(vmi, useEmulation, options, false)
	if err != nil {
		logger.Reason(err).Error("failed to generate libvirt domain from VMI spec")
		return nil, err
	}

	if err := converter.CheckEFI_OVMFRoms(vmi, c); err != nil {
		logger.Error("EFI OVMF roms missing")
		return nil, err
	}

	if err := converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c); err != nil {
		logger.Error("Conversion failed.")
		return nil, err
	}

	// Set defaults which are not coming from the cluster
	api.NewDefaulter(c.Architecture).SetObjectDefaults_Domain(domain)

	dom, err := l.virConn.LookupDomainByName(domain.Spec.Name)
	if err != nil {
		// We need the domain but it does not exist, so create it
		if domainerrors.IsNotFound(err) {
			domain, err = l.preStartHook(vmi, domain)
			if err != nil {
				logger.Reason(err).Error("pre start setup for VirtualMachineInstance failed.")
				return nil, err
			}
			dom, err = l.setDomainSpecWithHooks(vmi, &domain.Spec)
			if err != nil {
				return nil, err
			}
			logger.Info("Domain defined.")
		} else {
			logger.Reason(err).Error("Getting the domain failed.")
			return nil, err
		}
	}
	defer dom.Free()
	domState, _, err := dom.GetState()
	if err != nil {
		logger.Reason(err).Error("Getting the domain state failed.")
		return nil, err
	}

	// TODO Suspend, Pause, ..., for now we only support reaching the running state
	// TODO for migration and error detection we also need the state change reason
	// TODO blocked state
	if cli.IsDown(domState) && !vmi.IsRunning() && !vmi.IsFinal() {
		err = l.generateCloudInitISO(vmi, &dom)
		if err != nil {
			return nil, err
		}
		createFlags := getDomainCreateFlags(vmi)
		err = dom.CreateWithFlags(createFlags)
		if err != nil {
			logger.Reason(err).
				Errorf("Failed to start VirtualMachineInstance with flags %v.", createFlags)
			return nil, err
		}
		logger.Info("Domain started.")
		if vmi.ShouldStartPaused() {
			l.paused.add(vmi.UID)
		}
	} else if cli.IsPaused(domState) && !l.paused.contains(vmi.UID) {
		// TODO: if state change reason indicates a system error, we could try something smarter
		err := dom.Resume()
		if err != nil {
			logger.Reason(err).Error("unpausing the VirtualMachineInstance failed.")
			return nil, err
		}
		logger.Info("Domain unpaused.")
	} else {
		// Nothing to do
	}

	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	var oldSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &oldSpec)
	if err != nil {
		logger.Reason(err).Error("Parsing domain XML failed.")
		return nil, err
	}

	//Look up all the disks to detach
	for _, detachDisk := range getDetachedDisks(oldSpec.Devices.Disks, domain.Spec.Devices.Disks) {
		logger.V(1).Infof("Detaching disk %s, target %s", detachDisk.Alias.GetName(), detachDisk.Target.Device)
		detachBytes, err := xml.Marshal(detachDisk)
		if err != nil {
			logger.Reason(err).Error("marshalling detached disk failed")
			return nil, err
		}
		err = dom.DetachDevice(strings.ToLower(string(detachBytes)))
		if err != nil {
			logger.Reason(err).Error("detaching device")
			return nil, err
		}
	}
	//Look up all the disks to attach
	for _, attachDisk := range getAttachedDisks(oldSpec.Devices.Disks, domain.Spec.Devices.Disks) {
		allowAttach, err := checkIfDiskReadyToUse(getSourceFile(attachDisk))
		if err != nil {
			return nil, err
		}
		if !allowAttach {
			continue
		}
		logger.V(1).Infof("Attaching disk %s, target %s", attachDisk.Alias.GetName(), attachDisk.Target.Device)
		attachBytes, err := xml.Marshal(attachDisk)
		if err != nil {
			logger.Reason(err).Error("marshalling attached disk failed")
			return nil, err
		}
		err = dom.AttachDevice(strings.ToLower(string(attachBytes)))
		if err != nil {
			logger.Reason(err).Error("attaching device")
			return nil, err
		}
	}

	// TODO: check if VirtualMachineInstance Spec and Domain Spec are equal or if we have to sync
	return &oldSpec, nil
}

func getSourceFile(disk api.Disk) string {
	file := disk.Source.File
	if disk.Source.File == "" {
		file = disk.Source.Dev
	}
	return file
}

var checkIfDiskReadyToUse = checkIfDiskReadyToUseFunc

func checkIfDiskReadyToUseFunc(filename string) (bool, error) {
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		log.DefaultLogger().V(1).Infof("stat error: %v", err)
		return false, err
	}
	if (info.Mode() & os.ModeDevice) != 0 {
		file, err := os.OpenFile(filename, os.O_RDONLY, 0600)
		if err != nil {
			log.DefaultLogger().V(1).Infof("Unable to open file: %v", err)
			return false, nil
		}
		if err := file.Close(); err != nil {
			return false, fmt.Errorf("Unable to close file: %s", file.Name())
		}
		return true, nil
	}
	// Before attempting to attach, ensure we can open the file
	file, err := os.OpenFile(filename, os.O_RDWR, 0600)
	if err != nil {
		return false, nil
	}
	if err := file.Close(); err != nil {
		return false, fmt.Errorf("Unable to close file: %s", file.Name())
	}
	return true, nil
}

func getDetachedDisks(oldDisks, newDisks []api.Disk) []api.Disk {
	newDiskMap := make(map[string]api.Disk)
	for _, disk := range newDisks {
		file := getSourceFile(disk)
		if file != "" {
			newDiskMap[file] = disk
		}
	}
	res := make([]api.Disk, 0)
	for _, oldDisk := range oldDisks {
		if _, ok := newDiskMap[getSourceFile(oldDisk)]; !ok {
			// This disk got detached, add it to the list
			res = append(res, oldDisk)
		}
	}
	return res
}

func getAttachedDisks(oldDisks, newDisks []api.Disk) []api.Disk {
	oldDiskMap := make(map[string]api.Disk)
	for _, disk := range oldDisks {
		file := getSourceFile(disk)
		if file != "" {
			oldDiskMap[file] = disk
		}
	}
	res := make([]api.Disk, 0)
	for _, newDisk := range newDisks {
		if _, ok := oldDiskMap[getSourceFile(newDisk)]; !ok {
			// This disk got attached, add it to the list
			res = append(res, newDisk)
		}
	}
	return res
}

var isHotplugBlockDeviceVolume = isHotplugBlockDeviceVolumeFunc

func isHotplugBlockDeviceVolumeFunc(volumeName string) bool {
	path := converter.GetHotplugBlockDeviceVolumePath(volumeName)
	fileInfo, err := os.Stat(path)
	if err == nil {
		if !fileInfo.IsDir() && (fileInfo.Mode()&os.ModeDevice) != 0 {
			return true
		}
		return false
	}
	return false
}

var isBlockDeviceVolume = isBlockDeviceVolumeFunc

func isBlockDeviceVolumeFunc(volumeName string) (bool, error) {
	path := converter.GetBlockDeviceVolumePath(volumeName)
	fileInfo, err := os.Stat(path)
	if err == nil {
		if (fileInfo.Mode() & os.ModeDevice) != 0 {
			return true, nil
		}
		return false, fmt.Errorf("found %v, but it's not a block device", path)
	}
	if os.IsNotExist(err) {
		// cross check: is it a filesystem volume
		path = converter.GetFilesystemVolumePath(volumeName)
		fileInfo, err := os.Stat(path)
		if err == nil {
			if fileInfo.Mode().IsRegular() {
				return false, nil
			}
			return false, fmt.Errorf("found %v, but it's not a regular file", path)
		}
		if os.IsNotExist(err) {
			return false, fmt.Errorf("neither found block device nor regular file for volume %v", volumeName)
		}
	}
	return false, fmt.Errorf("error checking for block device: %v", err)
}

func (l *LibvirtDomainManager) getDomainSpec(dom cli.VirDomain) (*api.DomainSpec, error) {
	domainSpec, err := util.GetDomainSpecWithRuntimeInfo(dom)
	if err != nil {
		// Return without runtime info only for cases we know for sure it's not supposed to be there
		if domainerrors.IsNotFound(err) || domainerrors.IsInvalidOperation(err) {
			state, _, err := dom.GetState()
			if err != nil {
				return nil, err
			}
			return util.GetDomainSpec(state, dom)
		}
	}

	return domainSpec, err
}

func (l *LibvirtDomainManager) PauseVMI(vmi *v1.VirtualMachineInstance) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	logger := log.Log.Object(vmi)

	domName := util.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VirtualMachineInstance does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return fmt.Errorf("Domain not found.")
		} else {
			logger.Reason(err).Error("Getting the domain failed during pause.")
			return err
		}
	}
	defer dom.Free()

	domState, _, err := dom.GetState()
	if err != nil {
		logger.Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING {
		err = dom.Suspend()
		if err != nil {
			logger.Reason(err).Error("Signalling suspension failed.")
			return err
		}
		logger.Infof("Signaled pause for %s", vmi.GetObjectMeta().GetName())
		l.paused.add(vmi.UID)
	} else {
		logger.Infof("Domain is not running for %s", vmi.GetObjectMeta().GetName())
	}

	return nil
}

func (l *LibvirtDomainManager) UnpauseVMI(vmi *v1.VirtualMachineInstance) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	logger := log.Log.Object(vmi)

	domName := util.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VirtualMachineInstance does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return fmt.Errorf("Domain not found.")
		} else {
			logger.Reason(err).Error("Getting the domain failed during unpause.")
			return err
		}
	}
	defer dom.Free()

	domState, _, err := dom.GetState()
	if err != nil {
		logger.Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_PAUSED {
		err = dom.Resume()
		if err != nil {
			logger.Reason(err).Error("Signalling unpause failed.")
			return err
		}
		logger.Infof("Signaled unpause for %s", vmi.GetObjectMeta().GetName())
		l.paused.remove(vmi.UID)
		// Try to set guest time after this commands execution.
		// This operation is not disruptive.
		if err := l.setGuestTime(vmi); err != nil {
			return err
		}

	} else {
		logger.Infof("Domain is not paused for %s", vmi.GetObjectMeta().GetName())
	}

	return nil
}

func (l *LibvirtDomainManager) MarkGracefulShutdownVMI(vmi *v1.VirtualMachineInstance) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Getting the domain for shutdown failed.")
		return err
	}

	defer dom.Free()
	domainSpec, err := l.getDomainSpec(dom)
	if err != nil {
		return err
	}

	t := true

	if domainSpec.Metadata.KubeVirt.GracePeriod == nil {
		domainSpec.Metadata.KubeVirt.GracePeriod = &api.GracePeriodMetadata{
			MarkedForGracefulShutdown: &t,
		}
	} else if domainSpec.Metadata.KubeVirt.GracePeriod.MarkedForGracefulShutdown != nil &&
		*domainSpec.Metadata.KubeVirt.GracePeriod.MarkedForGracefulShutdown == true {
		// already marked, nothing to do
		return nil
	} else {
		domainSpec.Metadata.KubeVirt.GracePeriod.MarkedForGracefulShutdown = &t
	}

	d, err := l.setDomainSpecWithHooks(vmi, domainSpec)
	if err != nil {
		return err
	}
	defer d.Free()
	return nil

}

func (l *LibvirtDomainManager) SignalShutdownVMI(vmi *v1.VirtualMachineInstance) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	domName := util.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VirtualMachineInstance does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed during graceful shutdown.")
			return err
		}
	}
	defer dom.Free()

	domState, _, err := dom.GetState()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		domSpec, err := l.getDomainSpec(dom)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Unable to retrieve domain xml")
			return err
		}

		if domSpec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp == nil {
			err = dom.ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_ACPI_POWER_BTN)
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("Signalling graceful shutdown failed.")
				return err
			}
			log.Log.Object(vmi).Infof("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())

			now := metav1.Now()
			domSpec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp = &now
			d, err := l.setDomainSpecWithHooks(vmi, domSpec)
			if err != nil {
				log.Log.Object(vmi).Reason(err).Error("Unable to update grace period start time on domain xml")
				return err
			}
			defer d.Free()
		}
	}

	return nil
}

func (l *LibvirtDomainManager) KillVMI(vmi *v1.VirtualMachineInstance) error {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the VirtualMachineInstance does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()
	// TODO: Graceful shutdown
	domState, _, err := dom.GetState()
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		log.Log.Object(vmi).Reason(err).Error("Getting the domain state failed.")
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED || domState == libvirt.DOMAIN_SHUTDOWN {
		err = dom.DestroyFlags(libvirt.DOMAIN_DESTROY_GRACEFUL)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				return nil
			}
			log.Log.Object(vmi).Reason(err).Error("Destroying the domain state failed.")
			return err
		}
		log.Log.Object(vmi).Info("Domain stopped.")
		return nil
	}

	log.Log.Object(vmi).Info("Domain not running or paused, nothing to do.")
	return nil
}

func (l *LibvirtDomainManager) DeleteVMI(vmi *v1.VirtualMachineInstance) error {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		// If the domain does not exist, we are done
		if domainerrors.IsNotFound(err) {
			return nil
		} else {
			log.Log.Object(vmi).Reason(err).Error("Getting the domain failed.")
			return err
		}
	}
	defer dom.Free()

	err = dom.UndefineFlags(libvirt.DOMAIN_UNDEFINE_NVRAM)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Undefining the domain failed.")
		return err
	}
	log.Log.Object(vmi).Info("Domain undefined.")
	return nil
}

func (l *LibvirtDomainManager) ListAllDomains() ([]*api.Domain, error) {

	doms, err := l.virConn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, err
	}
	// Free memory allocated for domains
	defer func() {
		for i := range doms {
			err := doms[i].Free()
			if err != nil {
				log.Log.Reason(err).Warning("Error freeing a domain")
			}
		}
	}()

	var list []*api.Domain
	for _, dom := range doms {
		domain, err := util.NewDomain(dom)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		spec, err := l.getDomainSpec(dom)
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		domain.Spec = *spec
		status, reason, err := dom.GetState()
		if err != nil {
			if domainerrors.IsNotFound(err) {
				continue
			}
			return list, err
		}
		domain.SetState(util.ConvState(status), util.ConvReason(status, reason))
		list = append(list, domain)
	}

	return list, nil
}

func (l *LibvirtDomainManager) setDomainSpecWithHooks(vmi *v1.VirtualMachineInstance, origSpec *api.DomainSpec) (cli.VirDomain, error) {
	return util.SetDomainSpecStrWithHooks(l.virConn, vmi, origSpec)
}

func (l *LibvirtDomainManager) GetDomainStats() ([]*stats.DomainStats, error) {
	statsTypes := libvirt.DOMAIN_STATS_BALLOON | libvirt.DOMAIN_STATS_CPU_TOTAL | libvirt.DOMAIN_STATS_VCPU | libvirt.DOMAIN_STATS_INTERFACE | libvirt.DOMAIN_STATS_BLOCK
	flags := libvirt.CONNECT_GET_ALL_DOMAINS_STATS_RUNNING

	return l.virConn.GetDomainStats(statsTypes, flags)
}

func (l *LibvirtDomainManager) buildDevicesMetadata(vmi *v1.VirtualMachineInstance, dom cli.VirDomain) ([]cloudinit.DeviceData, error) {
	taggedInterfaces := make(map[string]v1.Interface)
	var devicesMetadata []cloudinit.DeviceData

	// Get all tagged interfaces for lookup
	for _, vif := range vmi.Spec.Domain.Devices.Interfaces {
		if vif.Tag != "" {
			taggedInterfaces[vif.Name] = vif
		}
	}

	devices, err := getAllDomainDevices(dom)
	if err != nil {
		return nil, err
	}
	interfaces := devices.Interfaces
	for _, nic := range interfaces {
		if data, exist := taggedInterfaces[nic.Alias.GetName()]; exist {
			address := nic.Address
			var mac string
			if nic.MAC != nil {
				mac = nic.MAC.MAC
			}
			pciAddrStr := fmt.Sprintf("%s:%s:%s:%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
			deviceData := cloudinit.DeviceData{
				Type:    cloudinit.NICMetadataType,
				Bus:     nic.Address.Type,
				Address: pciAddrStr,
				MAC:     mac,
				Tags:    []string{data.Tag},
			}
			devicesMetadata = append(devicesMetadata, deviceData)
		}
	}
	return devicesMetadata, nil

}

// GetGuestInfo queries the agent store and return the aggregated data from Guest agent
func (l *LibvirtDomainManager) GetGuestInfo() (v1.VirtualMachineInstanceGuestAgentInfo, error) {
	sysInfo := l.agentData.GetSysInfo()
	fsInfo := l.agentData.GetFS(10)
	userInfo := l.agentData.GetUsers(10)

	gaInfo := l.agentData.GetGA()

	guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{
		GAVersion:         gaInfo.Version,
		SupportedCommands: gaInfo.SupportedCommands,
		Hostname:          sysInfo.Hostname,
		OS: v1.VirtualMachineInstanceGuestOSInfo{
			Name:          sysInfo.OSInfo.Name,
			KernelRelease: sysInfo.OSInfo.KernelRelease,
			Version:       sysInfo.OSInfo.Version,
			PrettyName:    sysInfo.OSInfo.PrettyName,
			VersionID:     sysInfo.OSInfo.VersionId,
			KernelVersion: sysInfo.OSInfo.KernelVersion,
			Machine:       sysInfo.OSInfo.Machine,
			ID:            sysInfo.OSInfo.Id,
		},
		Timezone: fmt.Sprintf("%s, %d", sysInfo.Timezone.Zone, sysInfo.Timezone.Offset),
	}

	for _, user := range userInfo {
		guestInfo.UserList = append(guestInfo.UserList, v1.VirtualMachineInstanceGuestOSUser{
			UserName:  user.Name,
			Domain:    user.Domain,
			LoginTime: user.LoginTime,
		})
	}

	for _, fs := range fsInfo {
		guestInfo.FSInfo.Filesystems = append(guestInfo.FSInfo.Filesystems, v1.VirtualMachineInstanceFileSystem{
			DiskName:       fs.Name,
			MountPoint:     fs.Mountpoint,
			FileSystemType: fs.Type,
			UsedBytes:      fs.UsedBytes,
			TotalBytes:     fs.TotalBytes,
		})
	}

	return guestInfo, nil
}

// GetUsers return the full list of users on the guest machine
func (l *LibvirtDomainManager) GetUsers() ([]v1.VirtualMachineInstanceGuestOSUser, error) {
	userInfo := l.agentData.GetUsers(-1)
	userList := []v1.VirtualMachineInstanceGuestOSUser{}

	for _, user := range userInfo {
		userList = append(userList, v1.VirtualMachineInstanceGuestOSUser{
			UserName:  user.Name,
			Domain:    user.Domain,
			LoginTime: user.LoginTime,
		})
	}

	return userList, nil
}

// GetFilesystems return the full list of filesystems on the guest machine
func (l *LibvirtDomainManager) GetFilesystems() ([]v1.VirtualMachineInstanceFileSystem, error) {
	fsInfo := l.agentData.GetFS(-1)
	fsList := []v1.VirtualMachineInstanceFileSystem{}

	for _, fs := range fsInfo {
		fsList = append(fsList, v1.VirtualMachineInstanceFileSystem{
			DiskName:       fs.Name,
			MountPoint:     fs.Mountpoint,
			FileSystemType: fs.Type,
			UsedBytes:      fs.UsedBytes,
			TotalBytes:     fs.TotalBytes,
		})
	}

	return fsList, nil
}

// check whether VMI has a certain condition
func vmiHasCondition(vmi *v1.VirtualMachineInstance, cond v1.VirtualMachineInstanceConditionType) bool {
	if vmi == nil {
		return false
	}

	for _, c := range vmi.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func isDomainPaused(dom cli.VirDomain) (bool, error) {
	status, reason, err := dom.GetState()
	if err != nil {
		return false, err
	}
	return util.ConvState(status) == api.Paused &&
		util.ConvReason(status, reason) == api.ReasonPausedUser, nil
}

func getDomainCreateFlags(vmi *v1.VirtualMachineInstance) libvirt.DomainCreateFlags {
	flags := libvirt.DOMAIN_NONE

	if vmi.ShouldStartPaused() {
		flags |= libvirt.DOMAIN_START_PAUSED
	}
	return flags
}
