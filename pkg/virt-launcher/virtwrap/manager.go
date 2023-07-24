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
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/generic"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/gpu"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/efi"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/ignition"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	netsriov "kubevirt.io/kubevirt/pkg/network/sriov"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	kutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	accesscredentials "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/access-credentials"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
	domainerrors "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const (
	failedSyncGuestTime                       = "failed to sync guest time"
	failedGetDomain                           = "Getting the domain failed."
	failedGetDomainState                      = "Getting the domain state failed."
	failedDomainMemoryDump                    = "Domain memory dump failed"
	affectDeviceLiveAndConfigLibvirtFlags     = libvirt.DOMAIN_DEVICE_MODIFY_LIVE | libvirt.DOMAIN_DEVICE_MODIFY_CONFIG
	affectDomainLiveAndConfigLibvirtFlags     = libvirt.DOMAIN_AFFECT_LIVE | libvirt.DOMAIN_AFFECT_CONFIG
	affectDomainVCPULiveAndConfigLibvirtFlags = libvirt.DOMAIN_VCPU_LIVE | libvirt.DOMAIN_VCPU_CONFIG
)

const maxConcurrentHotplugHostDevices = 1
const maxConcurrentMemoryDumps = 1

type contextStore struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type DomainManager interface {
	SyncVMI(*v1.VirtualMachineInstance, bool, *cmdv1.VirtualMachineOptions) (*api.DomainSpec, error)
	PauseVMI(*v1.VirtualMachineInstance) error
	UnpauseVMI(*v1.VirtualMachineInstance) error
	FreezeVMI(*v1.VirtualMachineInstance, int32) error
	UnfreezeVMI(*v1.VirtualMachineInstance) error
	SoftRebootVMI(*v1.VirtualMachineInstance) error
	KillVMI(*v1.VirtualMachineInstance) error
	DeleteVMI(*v1.VirtualMachineInstance) error
	SignalShutdownVMI(*v1.VirtualMachineInstance) error
	MarkGracefulShutdownVMI()
	ListAllDomains() ([]*api.Domain, error)
	MigrateVMI(*v1.VirtualMachineInstance, *cmdclient.MigrationOptions) error
	PrepareMigrationTarget(*v1.VirtualMachineInstance, bool, *cmdv1.VirtualMachineOptions) error
	GetDomainStats() ([]*stats.DomainStats, error)
	CancelVMIMigration(*v1.VirtualMachineInstance) error
	GetGuestInfo() v1.VirtualMachineInstanceGuestAgentInfo
	GetUsers() []v1.VirtualMachineInstanceGuestOSUser
	GetFilesystems() []v1.VirtualMachineInstanceFileSystem
	FinalizeVirtualMachineMigration(*v1.VirtualMachineInstance) error
	HotplugHostDevices(vmi *v1.VirtualMachineInstance) error
	InterfacesStatus() []api.InterfaceStatus
	GetGuestOSInfo() *api.GuestOSInfo
	Exec(string, string, []string, int32) (string, error)
	GuestPing(string) error
	MemoryDump(vmi *v1.VirtualMachineInstance, dumpPath string) error
	GetQemuVersion() (string, error)
	UpdateVCPUs(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error
	GetSEVInfo() (*v1.SEVPlatformInfo, error)
	GetLaunchMeasurement(*v1.VirtualMachineInstance) (*v1.SEVMeasurementInfo, error)
	InjectLaunchSecret(*v1.VirtualMachineInstance, *v1.SEVSecretOptions) error
	UpdateGuestMemory(vmi *v1.VirtualMachineInstance) error
}

type LibvirtDomainManager struct {
	virConn cli.Connection

	// Anytime a get and a set is done on the domain, this lock must be held.
	domainModifyLock sync.Mutex
	// mutex to control access to the guest time context
	setGuestTimeLock sync.Mutex

	credManager *accesscredentials.AccessCredentialManager

	hotplugHostDevicesInProgress chan struct{}
	memoryDumpInProgress         chan struct{}

	virtShareDir             string
	ephemeralDiskDir         string
	paused                   pausedVMIs
	agentData                *agentpoller.AsyncAgentStore
	cloudInitDataStore       *cloudinit.CloudInitData
	setGuestTimeContextPtr   *contextStore
	efiEnvironment           *efi.EFIEnvironment
	ovmfPath                 string
	ephemeralDiskCreator     ephemeraldisk.EphemeralDiskCreatorInterface
	directIOChecker          converter.DirectIOChecker
	disksInfo                map[string]*cmdv1.DiskInfo
	cancelSafetyUnfreezeChan chan struct{}
	migrateInfoStats         *stats.DomainJobInfo

	metadataCache *metadata.Cache
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

func NewLibvirtDomainManager(connection cli.Connection, virtShareDir, ephemeralDiskDir string, agentStore *agentpoller.AsyncAgentStore, ovmfPath string, ephemeralDiskCreator ephemeraldisk.EphemeralDiskCreatorInterface, metadataCache *metadata.Cache) (DomainManager, error) {
	directIOChecker := converter.NewDirectIOChecker()
	return newLibvirtDomainManager(connection, virtShareDir, ephemeralDiskDir, agentStore, ovmfPath, ephemeralDiskCreator, directIOChecker, metadataCache)
}

func newLibvirtDomainManager(connection cli.Connection, virtShareDir, ephemeralDiskDir string, agentStore *agentpoller.AsyncAgentStore, ovmfPath string, ephemeralDiskCreator ephemeraldisk.EphemeralDiskCreatorInterface, directIOChecker converter.DirectIOChecker, metadataCache *metadata.Cache) (DomainManager, error) {
	manager := LibvirtDomainManager{
		virConn:          connection,
		virtShareDir:     virtShareDir,
		ephemeralDiskDir: ephemeralDiskDir,
		paused: pausedVMIs{
			paused: make(map[types.UID]bool, 0),
		},
		agentData:                agentStore,
		efiEnvironment:           efi.DetectEFIEnvironment(runtime.GOARCH, ovmfPath),
		ephemeralDiskCreator:     ephemeralDiskCreator,
		directIOChecker:          directIOChecker,
		disksInfo:                map[string]*cmdv1.DiskInfo{},
		cancelSafetyUnfreezeChan: make(chan struct{}),
		migrateInfoStats:         &stats.DomainJobInfo{},
		metadataCache:            metadataCache,
	}

	manager.hotplugHostDevicesInProgress = make(chan struct{}, maxConcurrentHotplugHostDevices)
	manager.memoryDumpInProgress = make(chan struct{}, maxConcurrentMemoryDumps)
	manager.credManager = accesscredentials.NewManager(connection, &manager.domainModifyLock, metadataCache)

	return &manager, nil
}

func getDomainSpec(dom cli.VirDomain) (*api.DomainSpec, error) {
	var newSpec api.DomainSpec
	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return &newSpec, err
	}
	err = xml.Unmarshal([]byte(xmlstr), &newSpec)
	if err != nil {
		return &newSpec, err
	}

	return &newSpec, nil
}

func getAllDomainDevices(dom cli.VirDomain) (api.Devices, error) {
	domSpec, err := getDomainSpec(dom)
	if err != nil {
		return domSpec.Devices, err
	}
	return domSpec.Devices, nil
}

func getAllDomainDisks(dom cli.VirDomain) ([]api.Disk, error) {
	devices, err := getAllDomainDevices(dom)
	if err != nil {
		return nil, err
	}

	return devices.Disks, nil
}

func (l *LibvirtDomainManager) UpdateGuestMemory(vmi *v1.VirtualMachineInstance) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	const errMsgPrefix = "failed to update Guest Memory"

	domainName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	defer dom.Free()

	requestedHotPlugMemory := vmi.Spec.Domain.Memory.Guest.DeepCopy()
	requestedHotPlugMemory.Sub(*vmi.Status.Memory.GuestAtBoot)
	pluggableMemoryRequested, err := vcpu.QuantityToByte(requestedHotPlugMemory)
	if err != nil {
		return err
	}

	spec, err := getDomainSpec(dom)
	if err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, "Parsing domain XML failed.")
	}

	spec.Devices.Memory.Target.Requested = pluggableMemoryRequested
	memoryDevice, err := xml.Marshal(spec.Devices.Memory)
	if err != nil {
		log.Log.Reason(err).Error("marshalling target virtio-mem failed")
		return err
	}

	err = dom.UpdateDeviceFlags(strings.ToLower(string(memoryDevice)), libvirt.DOMAIN_DEVICE_MODIFY_LIVE)
	if err != nil {
		log.Log.Reason(err).Error("updating device")
		return err
	}

	log.Log.V(2).Infof("hotplugging guest memory to %v", vmi.Spec.Domain.Memory.Guest.Value())
	return nil
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
		log.Log.Object(vmi).Reason(err).Error(failedSyncGuestTime)
		return err
	}

	go func() {
		defer dom.Free()

		// Syncing the guest time is a best-effort. Therefore
		// don't flood the logs
		var latestErr error
		defer func() {
			if latestErr != nil {
				log.Log.Object(vmi).Warning(latestErr.Error())
			}
		}()

		ctx := l.getGuestTimeContext()
		timeout := time.After(60 * time.Second)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-timeout:
				log.Log.Object(vmi).Error(failedSyncGuestTime)
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
						log.Log.Object(vmi).Reason(err).Warning(failedSyncGuestTime)
						return
					}

					switch libvirtError.Code {
					case libvirt.ERR_AGENT_UNRESPONSIVE:
						const unresponsive = "failed to set time: QEMU agent unresponsive"
						latestErr = fmt.Errorf("%s, %s", unresponsive, err)
						log.Log.Object(vmi).Reason(err).V(9).Info(unresponsive)
					case libvirt.ERR_OPERATION_UNSUPPORTED:
						// no need to retry as this opertaion is not supported
						log.Log.Object(vmi).Reason(err).Warning("failed to set time: not supported")
						return
					case libvirt.ERR_ARGUMENT_UNSUPPORTED:
						// no need to retry as the agent is not configured
						log.Log.Object(vmi).Reason(err).Warning("failed to set time: agent not configured")
						return
					default:
						latestErr = fmt.Errorf("%s, %s", failedSyncGuestTime, err)
						log.Log.Object(vmi).Reason(err).V(9).Info(failedSyncGuestTime)
					}
				} else {
					latestErr = nil
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
func (l *LibvirtDomainManager) PrepareMigrationTarget(
	vmi *v1.VirtualMachineInstance,
	allowEmulation bool,
	options *cmdv1.VirtualMachineOptions,
) error {
	return l.prepareMigrationTarget(vmi, allowEmulation, options)
}

// FinalizeVirtualMachineMigration finalized the migration after the migration has completed and vmi is running on target pod.
func (l *LibvirtDomainManager) FinalizeVirtualMachineMigration(vmi *v1.VirtualMachineInstance) error {
	return l.finalizeMigrationTarget(vmi)
}

// UpdateVCPUs plugs or unplugs vCPUs on a running domain
func (l *LibvirtDomainManager) UpdateVCPUs(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	const errMsgPrefix = "failed to update vCPUs"

	domainName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	defer dom.Free()
	var topology *cmdv1.Topology

	logger := log.Log.Object(vmi)

	vcpuTopology := vcpu.GetCPUTopology(vmi)
	vcpuCount := vcpu.CalculateRequestedVCPUs(vcpuTopology)
	// hot plug/unplug vCPUs
	if err := dom.SetVcpusFlags(uint(vcpuCount),
		affectDomainVCPULiveAndConfigLibvirtFlags); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	// Adjust guest vcpu config. Currently will handle vCPUs to pCPUs pinning
	if vmi.IsCPUDedicated() {
		useIOThreads := false
		if options != nil && options.Topology != nil {
			topology = options.Topology
		}

		podCPUSet, err := util.GetPodCPUSet()
		if err != nil {
			logger.Reason(err).Error("failed to read pod cpuset.")
			return fmt.Errorf("failed to read pod cpuset: %v", err)
		}

		domain, err := util.NewDomain(dom)
		if err != nil {
			return fmt.Errorf("%s: %v", errMsgPrefix, err)
		}

		spec, err := getDomainSpec(dom)
		if err != nil {
			return fmt.Errorf("%s: %v", errMsgPrefix, err)
		}

		domain.Spec = *spec

		if domain.Spec.CPUTune != nil && len(domain.Spec.CPUTune.IOThreadPin) > 0 {
			useIOThreads = true
		}

		err = vcpu.AdjustDomainForTopologyAndCPUSet(domain, vmi, topology, podCPUSet, useIOThreads)
		if err != nil {
			return fmt.Errorf("%s: %v", errMsgPrefix, err)
		}

		for _, vcpupin := range domain.Spec.CPUTune.VCPUPin {
			vcpu := vcpupin.VCPU
			cpuSet := vcpupin.CPUSet
			pcpu, err := strconv.Atoi(cpuSet)
			if err != nil {
				return fmt.Errorf("%s: %v", errMsgPrefix, err)
			}
			cpuMap := make([]bool, int(pcpu)+1)
			cpuMap[pcpu] = true
			err = dom.PinVcpuFlags(uint(vcpu), cpuMap, affectDomainLiveAndConfigLibvirtFlags)
			if err != nil {
				return fmt.Errorf("%s: %v", errMsgPrefix, err)
			}
		}
		if domain.Spec.CPUTune.EmulatorPin != nil {
			isolCpu, _ := strconv.Atoi(domain.Spec.CPUTune.EmulatorPin.CPUSet)
			cpuMap := make([]bool, isolCpu+1)
			cpuMap[int(isolCpu)] = true
			err = dom.PinEmulator(cpuMap, affectDomainLiveAndConfigLibvirtFlags)
			if err != nil {
				return fmt.Errorf("%s: %v", errMsgPrefix, err)
			}
		}

	}
	return nil
}

// HotplugHostDevices attach host-devices to running domain, currently only SRIOV host-devices are supported.
// This operation runs in the background, only one hotplug operation can occur at a time.
func (l *LibvirtDomainManager) HotplugHostDevices(vmi *v1.VirtualMachineInstance) error {
	select {
	case l.hotplugHostDevicesInProgress <- struct{}{}:
	default:
		return fmt.Errorf("hot-plug host-devices is in progress")
	}

	go func() {
		defer func() { <-l.hotplugHostDevicesInProgress }()

		if err := l.hotPlugHostDevices(vmi); err != nil {
			log.Log.Object(vmi).Error(err.Error())
		}
	}()
	return nil
}

func (l *LibvirtDomainManager) hotPlugHostDevices(vmi *v1.VirtualMachineInstance) error {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	const errMsgPrefix = "failed to hot-plug host-devices"

	domainName := api.VMINamespaceKeyFunc(vmi)
	domain, err := l.virConn.LookupDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	defer domain.Free()

	domainSpec, err := util.GetDomainSpecWithFlags(domain, 0)
	if err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	sriovHostDevices, err := sriov.GetHostDevicesToAttach(vmi, domainSpec)
	if err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := hostdevice.AttachHostDevices(domain, sriovHostDevices); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, hostdevice.AttachHostDevices(domain, sriovHostDevices))
	}

	return nil
}

func (l *LibvirtDomainManager) Exec(domainName, command string, args []string, timeoutSeconds int32) (string, error) {
	return agent.GuestExec(l.virConn, domainName, command, args, timeoutSeconds)
}

func (l *LibvirtDomainManager) GuestPing(domainName string) error {
	pingCmd := `{"execute":"guest-ping"}`
	_, err := l.virConn.QemuAgentCommand(pingCmd, domainName)
	return err
}

func getVMIEphemeralDisksTotalSize(ephemeralDiskDir string) *resource.Quantity {
	totalSize := int64(0)
	err := filepath.Walk(ephemeralDiskDir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() {
			totalSize += f.Size()
		}
		return nil
	})
	if err != nil {
		log.Log.Reason(err).Warning("failed to get VMI ephemeral disks size")
		return resource.NewScaledQuantity(0, 0)
	}

	return resource.NewScaledQuantity(totalSize, 0)
}

func getVMIMigrationDataSize(vmi *v1.VirtualMachineInstance, ephemeralDiskDir string) int64 {
	var memory resource.Quantity

	// Take memory from the requested memory
	if v, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
		memory = v
	}
	// In case that guest memory is explicitly set, override it
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		memory = *vmi.Spec.Domain.Memory.Guest
	}

	// get total data Size
	if vmi.Status.MigrationMethod == v1.BlockMigration {
		disksSize := getVMIEphemeralDisksTotalSize(ephemeralDiskDir)
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

func (l *LibvirtDomainManager) generateSomeCloudInitISO(vmi *v1.VirtualMachineInstance, domPtr *cli.VirDomain, size int64) error {
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
		var err error
		if size != 0 {
			err = cloudinit.GenerateEmptyIso(vmi.Name, vmi.Namespace, cloudInitDataStore, size)
		} else {
			// ClusterInstancetype will take precedence over a namespaced Instancetype
			// for setting instance_type in the metadata
			instancetype := vmi.Annotations[v1.ClusterInstancetypeAnnotation]
			if instancetype == "" {
				instancetype = vmi.Annotations[v1.InstancetypeAnnotation]
			}

			err = cloudinit.GenerateLocalData(vmi, instancetype, cloudInitDataStore)
		}
		if err != nil {
			return fmt.Errorf("generating local cloud-init data failed: %v", err)
		}
	}
	return nil
}

func (l *LibvirtDomainManager) generateCloudInitISO(vmi *v1.VirtualMachineInstance, domPtr *cli.VirDomain) error {
	return l.generateSomeCloudInitISO(vmi, domPtr, 0)
}

func (l *LibvirtDomainManager) generateCloudInitEmptyISO(vmi *v1.VirtualMachineInstance, domPtr *cli.VirDomain) error {
	if l.cloudInitDataStore == nil {
		return nil
	}
	for _, vs := range vmi.Status.VolumeStatus {
		if vs.Name == l.cloudInitDataStore.VolumeName {
			return l.generateSomeCloudInitISO(vmi, domPtr, vs.Size)
		}
	}
	return fmt.Errorf("failed to find the status of volume %s", l.cloudInitDataStore.VolumeName)
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
func (l *LibvirtDomainManager) preStartHook(vmi *v1.VirtualMachineInstance, domain *api.Domain, generateEmptyIsos bool) (*api.Domain, error) {
	logger := log.Log.Object(vmi)

	logger.Info("Executing PreStartHook on VMI pod environment")

	disksInfo := map[string]*containerdisk.DiskInfo{}
	for k, v := range l.disksInfo {
		if v != nil {
			disksInfo[k] = &containerdisk.DiskInfo{
				Format:      v.Format,
				BackingFile: v.BackingFile,
				ActualSize:  int64(v.ActualSize),
				VirtualSize: int64(v.VirtualSize),
			}
		}
	}

	// generate cloud-init data
	cloudInitData, err := cloudinit.ReadCloudInitVolumeDataSource(vmi, config.SecretSourceDir)
	if err != nil {
		return domain, fmt.Errorf("ReadCloudInitVolumeDataSource failed: %v", err)
	}

	// Pass cloud-init data to PreCloudInitIso hook
	logger.Info("Starting PreCloudInitIso hook")
	hooksManager := hooks.GetManager()
	cloudInitData, err = hooksManager.PreCloudInitIso(vmi, cloudInitData)
	if err != nil {
		return domain, fmt.Errorf("PreCloudInitIso hook failed: %v", err)
	}

	if cloudInitData != nil {
		// need to prepare the local path for cloud-init in advance for proper
		// detection of the disk driver cache mode
		if err := cloudinit.PrepareLocalPath(vmi.Name, vmi.Namespace); err != nil {
			return domain, fmt.Errorf("PrepareLocalPath failed: %v", err)
		}
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

	nonAbsentIfaces := netvmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	nonAbsentNets := netvmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, nonAbsentIfaces)
	err = netsetup.NewVMNetworkConfigurator(vmi, cache.CacheCreator{}).SetupPodNetworkPhase2(domain, nonAbsentNets)
	if err != nil {
		return domain, fmt.Errorf("preparing the pod network failed: %v", err)
	}

	// Create ephemeral disk for container disks
	err = containerdisk.CreateEphemeralImages(vmi, l.ephemeralDiskCreator, disksInfo)
	if err != nil {
		return domain, fmt.Errorf("preparing ephemeral container disk images failed: %v", err)
	}
	// Create images for volumes that are marked ephemeral.
	err = l.ephemeralDiskCreator.CreateEphemeralImages(vmi, domain)
	if err != nil {
		return domain, fmt.Errorf("preparing ephemeral images failed: %v", err)
	}
	// create empty disks if they exist
	if err := emptydisk.NewEmptyDiskCreator().CreateTemporaryDisks(vmi); err != nil {
		return domain, fmt.Errorf("creating empty disks failed: %v", err)
	}
	// create ConfigMap disks if they exists
	if err := config.CreateConfigMapDisks(vmi, generateEmptyIsos); err != nil {
		return domain, fmt.Errorf("creating config map disks failed: %v", err)
	}
	// create Secret disks if they exists
	if err := config.CreateSecretDisks(vmi, generateEmptyIsos); err != nil {
		return domain, fmt.Errorf("creating secret disks failed: %v", err)
	}

	// create Sysprep disks if they exists
	if err := config.CreateSysprepDisks(vmi, generateEmptyIsos); err != nil {
		return domain, fmt.Errorf("creating sysprep disks failed: %v", err)
	}

	// create DownwardAPI disks if they exists
	if err := config.CreateDownwardAPIDisks(vmi, generateEmptyIsos); err != nil {
		return domain, fmt.Errorf("creating DownwardAPI disks failed: %v", err)
	}
	// create ServiceAccount disk if exists
	if err := config.CreateServiceAccountDisk(vmi, generateEmptyIsos); err != nil {
		return domain, fmt.Errorf("creating service account disk failed: %v", err)
	}
	// create downwardMetric disk if exists
	if err := downwardmetrics.CreateDownwardMetricDisk(vmi); err != nil {
		return domain, fmt.Errorf("failed to craete downwardMetric disk: %v", err)
	}

	// set drivers cache mode
	for i := range domain.Spec.Devices.Disks {
		err := converter.SetDriverCacheMode(&domain.Spec.Devices.Disks[i], l.directIOChecker)
		if err != nil {
			return domain, err
		}
		converter.SetOptimalIOMode(&domain.Spec.Devices.Disks[i])
	}

	if err := l.credManager.HandleQemuAgentAccessCredentials(vmi); err != nil {
		return domain, fmt.Errorf("Starting qemu agent access credential propagation failed: %v", err)
	}

	// expand disk image files if they're too small
	expandDiskImagesOffline(vmi, domain)

	return domain, err
}

func expandDiskImagesOffline(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	logger := log.Log.Object(vmi)
	for _, disk := range domain.Spec.Devices.Disks {
		if shouldExpandOffline(disk) {
			possibleGuestSize, ok := possibleGuestSize(disk)
			if !ok {
				logger.Errorf("Failed to get possible guest size from disk")
				break
			}
			err := expandDiskImageOffline(getSourceFile(disk), possibleGuestSize)
			if err != nil {
				logger.Reason(err).Errorf("failed to expand disk image %v at boot", disk)
			}
		}
	}
}

func expandDiskImageOffline(imagePath string, size int64) error {
	log.Log.Infof("pre-start expansion of image %s to size %d", imagePath, size)
	var preallocateFlag string
	if converter.IsPreAllocated(imagePath) {
		preallocateFlag = "--preallocation=falloc"
	} else {
		preallocateFlag = "--preallocation=off"
	}
	size = kutil.AlignImageSizeTo1MiB(size, log.Log.With("image", imagePath))
	if size == 0 {
		return fmt.Errorf("%s must be at least 1MiB", imagePath)
	}
	cmd := exec.Command("/usr/bin/qemu-img", "resize", preallocateFlag, imagePath, strconv.FormatInt(size, 10))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("expanding image failed with error: %v, output: %s", err, out)
	}
	return nil
}

func possibleGuestSize(disk api.Disk) (int64, bool) {
	if disk.Capacity == nil {
		log.DefaultLogger().Error("No disk capacity")
		return 0, false
	}
	preferredSize := *disk.Capacity
	if disk.FilesystemOverhead == nil {
		log.DefaultLogger().Errorf("No filesystem overhead found for disk %v", disk)
		return 0, false
	}
	filesystemOverhead, err := strconv.ParseFloat(string(*disk.FilesystemOverhead), 64)
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to parse filesystem overhead as float")
		return 0, false
	}
	size := int64((1 - filesystemOverhead) * float64(preferredSize))
	size = kutil.AlignImageSizeTo1MiB(size, log.DefaultLogger())
	if size == 0 {
		return 0, false
	}
	return size, true
}

func shouldExpandOffline(disk api.Disk) bool {
	if !disk.ExpandDisksEnabled {
		return false
	}
	if disk.Source.Dev != "" {
		// Block devices don't need to be expanded
		return false
	}
	diskInfo, err := converter.GetImageInfo(getSourceFile(disk))
	if err != nil {
		log.DefaultLogger().Reason(err).Warning("Failed to get image info")
		return false
	}
	possibleGuestSize, ok := possibleGuestSize(disk)
	if !ok || possibleGuestSize <= diskInfo.VirtualSize {
		return false
	}
	return true
}

func (l *LibvirtDomainManager) generateConverterContext(vmi *v1.VirtualMachineInstance, allowEmulation bool, options *cmdv1.VirtualMachineOptions, isMigrationTarget bool) (*converter.ConverterContext, error) {

	logger := log.Log.Object(vmi)

	podCPUSet, err := util.GetPodCPUSet()
	if err != nil {
		logger.Reason(err).Error("failed to read pod cpuset.")
		return nil, fmt.Errorf("failed to read pod cpuset: %v", err)
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
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil || volume.VolumeSource.Ephemeral != nil {
			isBlockPVC := false
			if _, ok := hotplugVolumes[volume.Name]; ok {
				isBlockPVC = isHotplugBlockDeviceVolume(volume.Name)
			} else {
				isBlockPVC, _ = isBlockDeviceVolume(volume.Name)
			}
			isBlockPVCMap[volume.Name] = isBlockPVC
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

	var efiConf *converter.EFIConfiguration
	if vmi.IsBootloaderEFI() {
		secureBoot := vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot == nil || *vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot
		sev := kutil.IsSEVVMI(vmi)

		if !l.efiEnvironment.Bootable(secureBoot, sev) {
			log.Log.Errorf("EFI OVMF roms missing for booting in EFI mode with SecureBoot=%v, SEV=%v", secureBoot, sev)
			return nil, fmt.Errorf("EFI OVMF roms missing for booting in EFI mode with SecureBoot=%v, SEV=%v", secureBoot, sev)
		}

		efiConf = &converter.EFIConfiguration{
			EFICode:      l.efiEnvironment.EFICode(secureBoot, sev),
			EFIVars:      l.efiEnvironment.EFIVars(secureBoot, sev),
			SecureLoader: secureBoot,
		}
	}

	// Map the VirtualMachineInstance to the Domain
	c := &converter.ConverterContext{
		Architecture:          runtime.GOARCH,
		VirtualMachine:        vmi,
		AllowEmulation:        allowEmulation,
		CPUSet:                podCPUSet,
		IsBlockPVC:            isBlockPVCMap,
		IsBlockDV:             isBlockDVMap,
		EFIConfiguration:      efiConf,
		UseVirtioTransitional: vmi.Spec.Domain.Devices.UseVirtioTransitional != nil && *vmi.Spec.Domain.Devices.UseVirtioTransitional,
		PermanentVolumes:      permanentVolumes,
		EphemeraldiskCreator:  l.ephemeralDiskCreator,
		UseLaunchSecurity:     kutil.IsSEVVMI(vmi),
		FreePageReporting:     isFreePageReportingEnabled(false, vmi),
		SerialConsoleLog:      isSerialConsoleLogEnabled(false, vmi),
	}

	if options != nil {
		c.ExpandDisksEnabled = options.ExpandDisksEnabled
		if options.VirtualMachineSMBios != nil {
			c.SMBios = options.VirtualMachineSMBios
		}
		if options.Topology != nil {
			c.Topology = options.Topology
		}
		c.MemBalloonStatsPeriod = uint(options.MemBalloonStatsPeriod)
		// Add preallocated and thick-provisioned volumes for which we need to avoid the discard=unmap option
		c.VolumesDiscardIgnore = options.PreallocatedVolumes

		if len(options.DisksInfo) > 0 {
			l.disksInfo = options.DisksInfo
		}

		if options.GetClusterConfig() != nil {
			c.ExpandDisksEnabled = options.GetClusterConfig().GetExpandDisksEnabled()
			c.FreePageReporting = isFreePageReportingEnabled(options.GetClusterConfig().GetFreePageReportingDisabled(), vmi)
			c.BochsForEFIGuests = options.GetClusterConfig().GetBochsDisplayForEFIGuests()
			c.SerialConsoleLog = isSerialConsoleLogEnabled(options.GetClusterConfig().GetSerialConsoleLogDisabled(), vmi)
		}
	}
	c.DisksInfo = l.disksInfo

	if !isMigrationTarget {
		sriovDevices, err := sriov.CreateHostDevices(vmi)
		if err != nil {
			return nil, err
		}

		c.HotplugVolumes = hotplugVolumes
		c.SRIOVDevices = sriovDevices

		genericHostDevices, err := generic.CreateHostDevices(vmi.Spec.Domain.Devices.HostDevices)
		if err != nil {
			return nil, err
		}
		c.GenericHostDevices = genericHostDevices

		gpuHostDevices, err := gpu.CreateHostDevices(vmi.Spec.Domain.Devices.GPUs)
		if err != nil {
			return nil, err
		}
		c.GPUHostDevices = gpuHostDevices
	}

	return c, nil
}

func isFreePageReportingEnabled(clusterFreePageReportingDisabled bool, vmi *v1.VirtualMachineInstance) bool {
	if clusterFreePageReportingDisabled ||
		(vmi.Spec.Domain.Devices.AutoattachMemBalloon != nil && *vmi.Spec.Domain.Devices.AutoattachMemBalloon == false) ||
		vmi.IsHighPerformanceVMI() ||
		vmi.GetAnnotations()[v1.FreePageReportingDisabledAnnotation] == "true" {
		return false
	}

	return true
}

func isSerialConsoleLogEnabled(clusterSerialConsoleLogDisabled bool, vmi *v1.VirtualMachineInstance) bool {
	return (vmi.Spec.Domain.Devices.LogSerialConsole != nil && *vmi.Spec.Domain.Devices.LogSerialConsole) || (vmi.Spec.Domain.Devices.LogSerialConsole == nil && !clusterSerialConsoleLogDisabled)
}

func (l *LibvirtDomainManager) SyncVMI(vmi *v1.VirtualMachineInstance, allowEmulation bool, options *cmdv1.VirtualMachineOptions) (*api.DomainSpec, error) {
	l.domainModifyLock.Lock()
	defer l.domainModifyLock.Unlock()

	logger := log.Log.Object(vmi)

	domain := &api.Domain{}

	c, err := l.generateConverterContext(vmi, allowEmulation, options, false)
	if err != nil {
		logger.Reason(err).Error("failed to generate libvirt domain from VMI spec")
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
			domain, err = l.preStartHook(vmi, domain, false)
			if err != nil {
				logger.Reason(err).Error("pre start setup for VirtualMachineInstance failed.")
				return nil, err
			}

			dom, err = withNetworkIfacesResources(
				vmi, &domain.Spec,
				func(v *v1.VirtualMachineInstance, s *api.DomainSpec) (cli.VirDomain, error) {
					return l.setDomainSpecWithHooks(v, s)
				},
			)
			if err != nil {
				return nil, err
			}

			l.metadataCache.UID.Set(vmi.UID)
			l.metadataCache.GracePeriod.Set(
				api.GracePeriodMetadata{DeletionGracePeriodSeconds: converter.GracePeriodSeconds(vmi)},
			)
			logger.Info("Domain defined.")
		} else {
			logger.Reason(err).Error(failedGetDomain)
			return nil, err
		}
	}
	defer dom.Free()
	domState, _, err := dom.GetState()
	if err != nil {
		logger.Reason(err).Error(failedGetDomainState)
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

	oldSpec, err := getDomainSpec(dom)
	if err != nil {
		logger.Reason(err).Error("Parsing domain XML failed.")
		return nil, err
	}

	// Look up all the disks to detach
	for _, detachDisk := range getDetachedDisks(oldSpec.Devices.Disks, domain.Spec.Devices.Disks) {
		logger.V(1).Infof("Detaching disk %s, target %s", detachDisk.Alias.GetName(), detachDisk.Target.Device)
		detachBytes, err := xml.Marshal(detachDisk)
		if err != nil {
			logger.Reason(err).Error("marshalling detached disk failed")
			return nil, err
		}
		err = dom.DetachDeviceFlags(strings.ToLower(string(detachBytes)), affectDeviceLiveAndConfigLibvirtFlags)
		if err != nil {
			logger.Reason(err).Error("detaching device")
			return nil, err
		}
	}
	// Look up all the disks to attach
	for _, attachDisk := range getAttachedDisks(oldSpec.Devices.Disks, domain.Spec.Devices.Disks) {
		allowAttach, err := checkIfDiskReadyToUse(getSourceFile(attachDisk))
		if err != nil {
			return nil, err
		}
		if !allowAttach {
			continue
		}
		logger.V(1).Infof("Attaching disk %s, target %s", attachDisk.Alias.GetName(), attachDisk.Target.Device)
		// set drivers cache mode
		err = converter.SetDriverCacheMode(&attachDisk, l.directIOChecker)
		if err != nil {
			return nil, err
		}
		err = converter.SetOptimalIOMode(&attachDisk)
		if err != nil {
			return nil, err
		}

		attachBytes, err := xml.Marshal(attachDisk)
		if err != nil {
			logger.Reason(err).Error("marshalling attached disk failed")
			return nil, err
		}
		err = dom.AttachDeviceFlags(strings.ToLower(string(attachBytes)), affectDeviceLiveAndConfigLibvirtFlags)
		if err != nil {
			logger.Reason(err).Error("attaching device")
			return nil, err
		}
	}

	// Resize and notify the VM about changed disks
	for _, disk := range domain.Spec.Devices.Disks {
		if shouldExpandOnline(dom, disk) {
			possibleGuestSize, ok := possibleGuestSize(disk)
			if !ok {
				logger.Reason(err).Warningf("Failed to get possible guest size from disk %v", disk)
				break
			}
			err := dom.BlockResize(getSourceFile(disk), uint64(possibleGuestSize), libvirt.DOMAIN_BLOCK_RESIZE_BYTES)
			if err != nil {
				logger.Reason(err).Errorf("libvirt failed to expand disk image %v", disk)
			}
		}
	}

	if vmi.IsRunning() {
		networkInterfaceManager := newVirtIOInterfaceManager(
			dom, netsetup.NewVMNetworkConfigurator(vmi, cache.CacheCreator{}))
		if err := networkInterfaceManager.hotplugVirtioInterface(vmi, &api.Domain{Spec: *oldSpec}, domain); err != nil {
			return nil, err
		}
		if err := networkInterfaceManager.hotUnplugVirtioInterface(vmi, &api.Domain{Spec: *oldSpec}); err != nil {
			return nil, err
		}
	}

	// TODO: check if VirtualMachineInstance Spec and Domain Spec are equal or if we have to sync
	return oldSpec, nil
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
		if errors.Is(err, os.ErrNotExist) {
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

func isHotplugDisk(disk api.Disk) bool {
	return strings.HasPrefix(getSourceFile(disk), v1.HotplugDiskDir)
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
		if !isHotplugDisk(oldDisk) {
			continue
		}
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
		if !isHotplugDisk(newDisk) {
			continue
		}
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
		if (fileInfo.Mode() & os.ModeDevice) != 0 {
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
	if errors.Is(err, os.ErrNotExist) {
		// cross check: is it a filesystem volume
		path = converter.GetFilesystemVolumePath(volumeName)
		fileInfo, err := os.Stat(path)
		if err == nil {
			if fileInfo.Mode().IsRegular() {
				return false, nil
			}
			return false, fmt.Errorf("found %v, but it's not a regular file", path)
		}
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("neither found block device nor regular file for volume %v", volumeName)
		}
	}
	return false, fmt.Errorf("error checking for block device: %v", err)
}

func shouldExpandOnline(dom cli.VirDomain, disk api.Disk) bool {
	if !disk.ExpandDisksEnabled {
		log.DefaultLogger().V(3).Infof("Not expanding disks, ExpandDisks featuregate disabled")
		return false
	}
	blockInfo, err := dom.GetBlockInfo(getSourceFile(disk), 0)
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to get block info")
		return false
	}
	guestSize := blockInfo.Capacity
	possibleGuestSize, ok := possibleGuestSize(disk)
	if !ok || possibleGuestSize <= int64(guestSize) {
		return false
	}
	return true
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

func removePreviousMemoryDump(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to remove older memory dumps")
		return
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "memory.dump") {
			err = os.Remove(filepath.Join(dir, file.Name()))
			if err != nil {
				log.Log.Reason(err).Errorf("failed to remove older memory dumps")
			}
		}
	}
}

func (l *LibvirtDomainManager) MemoryDump(vmi *v1.VirtualMachineInstance, dumpPath string) error {
	select {
	case l.memoryDumpInProgress <- struct{}{}:
	default:
		log.Log.Object(vmi).Infof("memory-dump is in progress")
		return nil
	}

	go func() {
		defer func() { <-l.memoryDumpInProgress }()
		if err := l.memoryDump(vmi, dumpPath); err != nil {
			log.Log.Object(vmi).Reason(err).Error(failedDomainMemoryDump)
		}
	}()
	return nil
}

func (l *LibvirtDomainManager) memoryDump(vmi *v1.VirtualMachineInstance, dumpPath string) error {
	logger := log.Log.Object(vmi)

	if l.shouldSkipMemoryDump(dumpPath) {
		return nil
	}
	l.initializeMemoryDumpMetadata(dumpPath)

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if dom == nil || err != nil {
		return err
	}
	defer dom.Free()
	// keep trying to do memory dump even if remove previous one failed
	removePreviousMemoryDump(filepath.Dir(dumpPath))

	logger.Infof("Starting memory dump")
	failed := false
	reason := ""
	err = dom.CoreDumpWithFormat(dumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY)
	if err != nil {
		failed = true
		reason = fmt.Sprintf("%s: %s", failedDomainMemoryDump, err)
	} else {
		logger.Infof("Completed memory dump successfully")
	}

	l.setMemoryDumpResult(failed, reason)
	return err
}

func (l *LibvirtDomainManager) shouldSkipMemoryDump(dumpPath string) bool {
	memoryDumpMetadata, _ := l.metadataCache.MemoryDump.Load()
	if memoryDumpMetadata.FileName == filepath.Base(dumpPath) {
		// memory dump still in progress or have just completed
		// no need to trigger another one
		return true
	}
	return false
}

func (l *LibvirtDomainManager) initializeMemoryDumpMetadata(dumpPath string) {
	l.metadataCache.MemoryDump.WithSafeBlock(func(memoryDumpMetadata *api.MemoryDumpMetadata, initialized bool) {
		now := metav1.Now()
		*memoryDumpMetadata = api.MemoryDumpMetadata{
			FileName:       filepath.Base(dumpPath),
			StartTimestamp: &now,
		}
	})
	log.Log.V(4).Infof("initialize memory dump metadata: %s", l.metadataCache.MemoryDump.String())
}

func (l *LibvirtDomainManager) setMemoryDumpResult(failed bool, reason string) {
	l.metadataCache.MemoryDump.WithSafeBlock(func(memoryDumpMetadata *api.MemoryDumpMetadata, initialized bool) {
		if !initialized {
			// nothing to report if memory dump metadata is empty
			return
		}

		now := metav1.Now()
		memoryDumpMetadata.Completed = true
		memoryDumpMetadata.EndTimestamp = &now
		memoryDumpMetadata.Failed = failed
		memoryDumpMetadata.FailureReason = reason
	})
	log.Log.V(4).Infof("set memory dump results in metadata: %s", l.metadataCache.MemoryDump.String())
	return
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
		logger.Reason(err).Error(failedGetDomainState)
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
		logger.Reason(err).Error(failedGetDomainState)
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

func (l *LibvirtDomainManager) migrationInProgress() bool {
	migrationMetadata, exists := l.metadataCache.Migration.Load()
	return exists && migrationMetadata.StartTimestamp != nil && migrationMetadata.EndTimestamp == nil
}

func (l *LibvirtDomainManager) scheduleSafetyVMIUnfreeze(vmi *v1.VirtualMachineInstance, unfreezeTimeout time.Duration) {
	select {
	case <-time.After(unfreezeTimeout):
		log.Log.Warningf("Unfreeze was not called for vmi %s for more then %v, initiating unfreeze",
			vmi.Name, unfreezeTimeout)
		l.UnfreezeVMI(vmi)
	case <-l.cancelSafetyUnfreezeChan:
		log.Log.V(3).Infof("Canceling schedualed Unfreeze for vmi %s", vmi.Name)
		// aborted
	}
}

func (l *LibvirtDomainManager) cancelSafetyUnfreeze() {
	select {
	case l.cancelSafetyUnfreezeChan <- struct{}{}:
	default:
	}
}

func (l *LibvirtDomainManager) getParsedFSStatus(domainName string) (string, error) {
	cmdResult, err := l.virConn.QemuAgentCommand(`{"execute":"`+string(agentpoller.GET_FSFREEZE_STATUS)+`"}`, domainName)
	if err != nil {
		return "", err
	}
	fsfreezeStatus, err := agentpoller.ParseFSFreezeStatus(cmdResult)
	if err != nil {
		return "", err
	}

	return fsfreezeStatus.Status, nil
}

func (l *LibvirtDomainManager) FreezeVMI(vmi *v1.VirtualMachineInstance, unfreezeTimeoutSeconds int32) error {
	if l.migrationInProgress() {
		return fmt.Errorf("Failed to freeze VMI, VMI is currently during migration")
	}
	domainName := api.VMINamespaceKeyFunc(vmi)
	safetyUnfreezeTimeout := time.Duration(unfreezeTimeoutSeconds) * time.Second

	fsfreezeStatus, err := l.getParsedFSStatus(domainName)
	if err != nil {
		log.Log.Errorf("Failed to get fs status before freeze vmi %s, %s", vmi.Name, err.Error())
		return err
	}

	// idempotent - prevent failuer in case fs is already frozen
	if fsfreezeStatus == api.FSFrozen {
		return nil
	}
	_, err = l.virConn.QemuAgentCommand(`{"execute":"guest-fsfreeze-freeze"}`, domainName)
	if err != nil {
		log.Log.Errorf("Failed to freeze vmi, %s", err.Error())
		return err
	}

	l.cancelSafetyUnfreeze()
	if safetyUnfreezeTimeout != 0 {
		go l.scheduleSafetyVMIUnfreeze(vmi, safetyUnfreezeTimeout)
	}
	return nil
}

func (l *LibvirtDomainManager) UnfreezeVMI(vmi *v1.VirtualMachineInstance) error {
	l.cancelSafetyUnfreeze()
	domainName := api.VMINamespaceKeyFunc(vmi)
	fsfreezeStatus, err := l.getParsedFSStatus(domainName)
	if err == nil {
		// prevent initating fs thaw to prevent rerunning the thaw hook
		if fsfreezeStatus == api.FSThawed {
			return nil
		}
	}
	// even if failed we should still try to unfreeze the fs
	_, err = l.virConn.QemuAgentCommand(`{"execute":"guest-fsfreeze-thaw"}`, domainName)
	if err != nil {
		log.Log.Errorf("Failed to unfreeze vmi, %s", err.Error())
		return err
	}
	return nil
}

func (l *LibvirtDomainManager) SoftRebootVMI(vmi *v1.VirtualMachineInstance) error {
	domainRebootFlagValues := libvirt.DOMAIN_REBOOT_GUEST_AGENT
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	if !condManager.HasConditionWithStatus(vmi, v1.VirtualMachineInstanceAgentConnected, k8sv1.ConditionTrue) {
		if features := vmi.Spec.Domain.Features; features != nil && features.ACPI.Enabled != nil && !(*features.ACPI.Enabled) {
			err := fmt.Errorf("VMI neither have the agent connected nor the ACPI feature enabled")
			log.Log.Object(vmi).Reason(err).Error("Setting the domain reboot flag failed.")
			return err
		}
		domainRebootFlagValues = libvirt.DOMAIN_REBOOT_ACPI_POWER_BTN
	}

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Getting the domain for soft reboot failed.")
		return err
	}

	defer dom.Free()
	if err = dom.Reboot(domainRebootFlagValues); err != nil {
		libvirtError, ok := err.(libvirt.Error)
		if !ok || libvirtError.Code != libvirt.ERR_AGENT_UNRESPONSIVE {
			log.Log.Object(vmi).Reason(err).Error("Soft rebooting the domain failed.")
			return err
		}
	}

	return nil
}

func (l *LibvirtDomainManager) MarkGracefulShutdownVMI() {
	l.metadataCache.GracePeriod.WithSafeBlock(func(gracePeriodMetadata *api.GracePeriodMetadata, _ bool) {
		gracePeriodMetadata.MarkedForGracefulShutdown = pointer.Bool(true)
	})
	log.Log.V(4).Infof("Marked for graceful shutdown in metadata: %s", l.metadataCache.GracePeriod.String())
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
		log.Log.Object(vmi).Reason(err).Error(failedGetDomainState)
		return err
	}

	if domState == libvirt.DOMAIN_RUNNING || domState == libvirt.DOMAIN_PAUSED {
		err = dom.ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_ACPI_POWER_BTN)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Signalling graceful shutdown failed.")
			return err
		}
		log.Log.Object(vmi).Infof("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())

		l.metadataCache.GracePeriod.WithSafeBlock(func(gracePeriodMetadata *api.GracePeriodMetadata, _ bool) {
			if gracePeriodMetadata.DeletionTimestamp == nil {
				now := metav1.Now()
				gracePeriodMetadata.DeletionTimestamp = &now
			}
		})
		log.Log.V(4).Infof("Graceful period set in metadata: %s", l.metadataCache.GracePeriod.String())
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
			log.Log.Object(vmi).Reason(err).Error(failedGetDomain)
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
		log.Log.Object(vmi).Reason(err).Error(failedGetDomainState)
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

	log.Log.Object(vmi).Info("Domain not running, paused or shut down, nothing to do.")
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
			log.Log.Object(vmi).Reason(err).Error(failedGetDomain)
			return err
		}
	}
	defer dom.Free()

	err = dom.UndefineFlags(libvirt.DOMAIN_UNDEFINE_KEEP_NVRAM)
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
		spec.Metadata.KubeVirt = metadata.LoadKubevirtMetadata(l.metadataCache)
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

func (l *LibvirtDomainManager) GetQemuVersion() (string, error) {
	return l.virConn.GetQemuVersion()
}

func (l *LibvirtDomainManager) GetDomainStats() ([]*stats.DomainStats, error) {
	statsTypes := libvirt.DOMAIN_STATS_BALLOON | libvirt.DOMAIN_STATS_CPU_TOTAL | libvirt.DOMAIN_STATS_VCPU | libvirt.DOMAIN_STATS_INTERFACE | libvirt.DOMAIN_STATS_BLOCK | libvirt.DOMAIN_STATS_DIRTYRATE
	flags := libvirt.CONNECT_GET_ALL_DOMAINS_STATS_RUNNING | libvirt.CONNECT_GET_ALL_DOMAINS_STATS_PAUSED

	return l.virConn.GetDomainStats(statsTypes, l.migrateInfoStats, flags)
}

func formatPCIAddressStr(address *api.Address) string {
	return fmt.Sprintf("%s:%s:%s.%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
}

func addToDeviceMetadata(metadataType cloudinit.DeviceMetadataType, address *api.Address, mac string, tag string, devicesMetadata []cloudinit.DeviceData, numa *uint32, numaAlignedCPUs []uint32) []cloudinit.DeviceData {
	pciAddrStr := formatPCIAddressStr(address)
	deviceData := cloudinit.DeviceData{
		Type:    metadataType,
		Bus:     address.Type,
		Address: pciAddrStr,
		MAC:     mac,
		Tags:    []string{tag},
	}
	if numa != nil {
		deviceData.NumaNode = *numa
	}
	if len(numaAlignedCPUs) > 0 {
		deviceData.AlignedCPUs = numaAlignedCPUs
	}
	devicesMetadata = append(devicesMetadata, deviceData)
	return devicesMetadata
}

func getDeviceNUMACPUAffinity(dev api.HostDevice, vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) (numaNodePtr *uint32, cpuList []uint32) {
	if dev.Source.Address != nil {
		pciAddress := formatPCIAddressStr(dev.Source.Address)
		if vmi.Spec.Domain.CPU.NUMA != nil && vmi.Spec.Domain.CPU.NUMA.GuestMappingPassthrough != nil {
			if numa, err := hardware.GetDeviceNumaNode(pciAddress); err == nil {
				numaNodePtr = numa
				return
			}
		} else if vmi.IsCPUDedicated() {
			if vCPUList, err := hardware.LookupDeviceVCPUAffinity(pciAddress, domainSpec); err == nil {
				cpuList = vCPUList
				return
			}
		}
	}
	return
}

func (l *LibvirtDomainManager) buildDevicesMetadata(vmi *v1.VirtualMachineInstance, dom cli.VirDomain) ([]cloudinit.DeviceData, error) {
	taggedInterfaces := make(map[string]v1.Interface)
	taggedHostDevices := make(map[string]v1.HostDevice)
	taggedGPUs := make(map[string]v1.GPU)
	var devicesMetadata []cloudinit.DeviceData

	// Get all tagged interfaces for lookup
	for _, vif := range vmi.Spec.Domain.Devices.Interfaces {
		if vif.Tag != "" {
			taggedInterfaces[vif.Name] = vif
		}
	}

	// Get all tagged host devices for lookup
	for _, dev := range vmi.Spec.Domain.Devices.HostDevices {
		if dev.Tag != "" {
			taggedHostDevices[dev.Name] = dev
		}
	}

	// Get all tagged gpus for lookup
	for _, dev := range vmi.Spec.Domain.Devices.GPUs {
		if dev.Tag != "" {
			taggedGPUs[dev.Name] = dev
		}
	}

	domainSpec, err := getDomainSpec(dom)
	if err != nil {
		return nil, err
	}
	devices := domainSpec.Devices
	interfaces := devices.Interfaces
	for _, nic := range interfaces {
		if data, exist := taggedInterfaces[nic.Alias.GetName()]; exist {
			var mac string
			if nic.MAC != nil {
				mac = nic.MAC.MAC
			}
			// currently network Interfaces do not contain host devices thus have no NUMA alignment.
			deviceAlignedCPUs := []uint32{}
			devicesMetadata = addToDeviceMetadata(cloudinit.NICMetadataType,
				nic.Address,
				mac,
				data.Tag,
				devicesMetadata,
				nil,
				deviceAlignedCPUs,
			)
		}
	}

	hostDevices := devices.HostDevices
	for _, dev := range hostDevices {
		devAliasNoPrefix := strings.TrimPrefix(dev.Alias.GetName(), netsriov.AliasPrefix)
		hostDevAliasNoPrefix := strings.TrimPrefix(dev.Alias.GetName(), generic.AliasPrefix)
		gpuDevAliasNoPrefix := strings.TrimPrefix(dev.Alias.GetName(), gpu.AliasPrefix)
		if data, exist := taggedInterfaces[devAliasNoPrefix]; exist {
			deviceNumaNode, deviceAlignedCPUs := getDeviceNUMACPUAffinity(dev, vmi, domainSpec)
			devicesMetadata = addToDeviceMetadata(cloudinit.NICMetadataType,
				dev.Address,
				"",
				data.Tag,
				devicesMetadata,
				deviceNumaNode,
				deviceAlignedCPUs,
			)
		}
		if data, exist := taggedHostDevices[hostDevAliasNoPrefix]; exist {
			deviceNumaNode, deviceAlignedCPUs := getDeviceNUMACPUAffinity(dev, vmi, domainSpec)
			devicesMetadata = addToDeviceMetadata(cloudinit.HostDevMetadataType,
				dev.Address,
				"",
				data.Tag,
				devicesMetadata,
				deviceNumaNode,
				deviceAlignedCPUs,
			)
		}
		if data, exist := taggedHostDevices[gpuDevAliasNoPrefix]; exist {
			deviceNumaNode, deviceAlignedCPUs := getDeviceNUMACPUAffinity(dev, vmi, domainSpec)
			devicesMetadata = addToDeviceMetadata(cloudinit.HostDevMetadataType,
				dev.Address,
				"",
				data.Tag,
				devicesMetadata,
				deviceNumaNode,
				deviceAlignedCPUs,
			)
		}
	}
	return devicesMetadata, nil
}

// GetGuestInfo queries the agent store and return the aggregated data from Guest agent
func (l *LibvirtDomainManager) GetGuestInfo() v1.VirtualMachineInstanceGuestAgentInfo {
	sysInfo := l.agentData.GetSysInfo()
	fsInfo := l.agentData.GetFS(10)
	userInfo := l.agentData.GetUsers(10)
	var fsFreezestatus api.FSFreeze
	if status := l.agentData.GetFSFreezeStatus(); status != nil {
		fsFreezestatus = *status
	}

	gaInfo := l.agentData.GetGA()

	guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{
		GAVersion:         gaInfo.Version,
		SupportedCommands: gaInfo.SupportedCommands,
		Hostname:          sysInfo.Hostname,
		FSFreezeStatus:    fsFreezestatus.Status,
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

	return guestInfo
}

// InterfacesStatus returns the interfaces Guest Agent reported
func (l *LibvirtDomainManager) InterfacesStatus() []api.InterfaceStatus {
	return l.agentData.GetInterfaceStatus()
}

// GetGuestOSInfo returns the Guest OS version and architecture
func (l *LibvirtDomainManager) GetGuestOSInfo() *api.GuestOSInfo {
	return l.agentData.GetGuestOSInfo()
}

// GetUsers return the full list of users on the guest machine
func (l *LibvirtDomainManager) GetUsers() []v1.VirtualMachineInstanceGuestOSUser {
	userInfo := l.agentData.GetUsers(-1)
	userList := []v1.VirtualMachineInstanceGuestOSUser{}

	for _, user := range userInfo {
		userList = append(userList, v1.VirtualMachineInstanceGuestOSUser{
			UserName:  user.Name,
			Domain:    user.Domain,
			LoginTime: user.LoginTime,
		})
	}

	return userList
}

// GetFilesystems return the full list of filesystems on the guest machine
func (l *LibvirtDomainManager) GetFilesystems() []v1.VirtualMachineInstanceFileSystem {
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

	return fsList
}

func (l *LibvirtDomainManager) GetSEVInfo() (*v1.SEVPlatformInfo, error) {
	sevNodeParameters, err := l.virConn.GetSEVInfo()
	if err != nil {
		log.Log.Reason(err).Error("Getting SEV platform info failed")
		return nil, err
	}

	return &v1.SEVPlatformInfo{
		PDH:       sevNodeParameters.PDH,
		CertChain: sevNodeParameters.CertChain,
	}, nil
}

func (l *LibvirtDomainManager) GetLaunchMeasurement(vmi *v1.VirtualMachineInstance) (*v1.SEVMeasurementInfo, error) {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedGetDomain)
		return nil, err
	}
	defer dom.Free()

	const flags = uint32(0)
	domainLaunchSecurityParameters, err := dom.GetLaunchSecurityInfo(flags)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Getting launch security info failed")
		return nil, err
	}

	sevMeasurementInfo := v1.SEVMeasurementInfo{}
	if domainLaunchSecurityParameters.SEVMeasurementSet {
		sevMeasurementInfo.Measurement = domainLaunchSecurityParameters.SEVMeasurement
	}
	if domainLaunchSecurityParameters.SEVAPIMajorSet {
		sevMeasurementInfo.APIMajor = domainLaunchSecurityParameters.SEVAPIMajor
	}
	if domainLaunchSecurityParameters.SEVAPIMinorSet {
		sevMeasurementInfo.APIMinor = domainLaunchSecurityParameters.SEVAPIMinor
	}
	if domainLaunchSecurityParameters.SEVBuildIDSet {
		sevMeasurementInfo.BuildID = domainLaunchSecurityParameters.SEVBuildID
	}
	if domainLaunchSecurityParameters.SEVPolicySet {
		sevMeasurementInfo.Policy = domainLaunchSecurityParameters.SEVPolicy
	}

	loader := l.efiEnvironment.EFICode(false, true) // no secureBoot, with sev
	f, err := os.Open(loader)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Error opening loader binary %s", loader)
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Error reading loader binary %s", loader)
		return nil, err
	}

	sevMeasurementInfo.LoaderSHA = fmt.Sprintf("%x", h.Sum(nil))

	return &sevMeasurementInfo, nil
}

func (l *LibvirtDomainManager) InjectLaunchSecret(vmi *v1.VirtualMachineInstance, sevSecretOptions *v1.SEVSecretOptions) error {
	if sevSecretOptions.Header == "" {
		return fmt.Errorf("Header is required")
	} else if sevSecretOptions.Secret == "" {
		return fmt.Errorf("Secret is required")
	}

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := l.virConn.LookupDomainByName(domName)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedGetDomain)
		return err
	}
	defer dom.Free()

	domainLaunchSecurityStateParameters := &libvirt.DomainLaunchSecurityStateParameters{
		SEVSecret:          sevSecretOptions.Secret,
		SEVSecretSet:       true,
		SEVSecretHeader:    sevSecretOptions.Header,
		SEVSecretHeaderSet: true,
	}

	const flags = uint32(0)
	if err := dom.SetLaunchSecurityState(domainLaunchSecurityStateParameters, flags); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Setting launch security state failed")
		return err
	}

	return nil
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
	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		flags |= libvirt.DOMAIN_START_PAUSED
	}
	return flags
}
