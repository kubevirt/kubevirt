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
 * Copyright The KubeVirt Authors.
 *
*/

package converter

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/ignition"
	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
)

const (
	deviceTypeNotCompatibleFmt = "device %s is of type lun. Not compatible with a file based disk"
	defaultIOThread            = uint(1)
	bootMenuTimeoutMS          = uint(10000)
	QEMUSeaBiosDebugPipe       = "/var/run/kubevirt-private/QEMUSeaBiosDebugPipe"
)

type deviceNamer struct {
	existingNameMap map[string]string
	usedDeviceMap   map[string]string
}

type EFIConfiguration struct {
	EFICode      string
	EFIVars      string
	SecureLoader bool
}

type ConverterContext struct {
	Architecture                    arch.Converter
	AllowEmulation                  bool
	Secrets                         map[string]*k8sv1.Secret
	VirtualMachine                  *v1.VirtualMachineInstance
	CPUSet                          []int
	IsBlockPVC                      map[string]bool
	IsBlockDV                       map[string]bool
	ApplyCBT                        map[string]string
	HotplugVolumes                  map[string]v1.VolumeStatus
	PermanentVolumes                map[string]v1.VolumeStatus
	MigratedVolumes                 map[string]string
	DisksInfo                       map[string]*disk.DiskInfo
	SMBios                          *cmdv1.SMBios
	SRIOVDevices                    []api.HostDevice
	GenericHostDevices              []api.HostDevice
	GPUHostDevices                  []api.HostDevice
	EFIConfiguration                *EFIConfiguration
	MemBalloonStatsPeriod           uint
	UseVirtioTransitional           bool
	EphemeraldiskCreator            ephemeraldisk.EphemeralDiskCreatorInterface
	VolumesDiscardIgnore            []string
	Topology                        *cmdv1.Topology
	ExpandDisksEnabled              bool
	UseLaunchSecuritySEV            bool // For AMD SEV/ES/SNP
	UseLaunchSecurityTDX            bool // For Intel TDX
	UseLaunchSecurityPV             bool // For IBM SE(s390-pv)
	UseBlkMQ                        bool
	FreePageReporting               bool
	BochsForEFIGuests               bool
	SerialConsoleLog                bool
	DomainAttachmentByInterfaceName map[string]string
}

func assignDiskToSCSIController(disk *api.Disk, unit int) {
	// Ensure we assign this disk to the correct scsi controller
	if disk.Address == nil {
		disk.Address = &api.Address{}
	}
	disk.Address.Type = "drive"
	// This should be the index of the virtio-scsi controller, which is hard coded to 0
	disk.Address.Controller = "0"
	disk.Address.Bus = "0"
	disk.Address.Unit = strconv.Itoa(unit)
}

func setReservation(disk *api.Disk) {
	disk.Source.Reservations = &api.Reservations{
		Managed: "no",
		SourceReservations: &api.SourceReservations{
			Type: "unix",
			Path: reservation.GetPrHelperSocketPath(),
			Mode: "client",
		},
	}
}

func setErrorPolicy(diskDevice *v1.Disk, disk *api.Disk) error {
	if diskDevice.ErrorPolicy == nil {
		disk.Driver.ErrorPolicy = v1.DiskErrorPolicyStop
		return nil
	}
	switch *diskDevice.ErrorPolicy {
	case v1.DiskErrorPolicyStop, v1.DiskErrorPolicyIgnore, v1.DiskErrorPolicyReport, v1.DiskErrorPolicyEnospace:
		disk.Driver.ErrorPolicy = *diskDevice.ErrorPolicy
	default:
		return fmt.Errorf("error policy %s not recognized", *diskDevice.ErrorPolicy)
	}
	return nil
}

type DirectIOChecker interface {
	CheckBlockDevice(path string) (bool, error)
	CheckFile(path string) (bool, error)
}

type directIOChecker struct{}

func NewDirectIOChecker() DirectIOChecker {
	return &directIOChecker{}
}

func (c *directIOChecker) CheckBlockDevice(path string) (bool, error) {
	return c.check(path, syscall.O_RDONLY)
}

func (c *directIOChecker) CheckFile(path string) (bool, error) {
	flags := syscall.O_RDONLY
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// try to create the file and perform the check
		flags = flags | syscall.O_CREAT
		defer os.Remove(path)
	}
	return c.check(path, flags)
}

// based on https://gitlab.com/qemu-project/qemu/-/blob/master/util/osdep.c#L344
func (c *directIOChecker) check(path string, flags int) (bool, error) {
	// #nosec No risk for path injection as we only open the file, not read from it. The function leaks only whether the directory to `path` exists.
	f, err := os.OpenFile(path, flags|syscall.O_DIRECT, 0600)
	if err != nil {
		// EINVAL is returned if the filesystem does not support the O_DIRECT flag
		if err, ok := err.(*os.PathError); ok && err.Err == syscall.EINVAL {
			// #nosec No risk for path injection as we only open the file, not read from it. The function leaks only whether the directory to `path` exists.
			f, err := os.OpenFile(path, flags & ^syscall.O_DIRECT, 0600)
			if err == nil {
				defer util.CloseIOAndCheckErr(f, nil)
				return false, nil
			}
		}
		return false, err
	}
	defer util.CloseIOAndCheckErr(f, nil)
	return true, nil
}

func SetDriverCacheMode(disk *api.Disk, directIOChecker DirectIOChecker) error {
	var path string
	var err error
	supportDirectIO := true
	mode := v1.DriverCache(disk.Driver.Cache)
	isBlockDev := false

	switch {
	case disk.Source.File != "":
		path = disk.Source.File
	case disk.Source.Dev != "":
		path = disk.Source.Dev
	// handle empty cdrom
	case disk.Device == "cdrom":
		return nil
	default:
		return fmt.Errorf("unable to set a driver cache mode, disk is neither a block device nor a file")
	}

	if mode == "" || mode == v1.CacheNone {
		if isBlockDev {
			supportDirectIO, err = directIOChecker.CheckBlockDevice(path)
		} else {
			supportDirectIO, err = directIOChecker.CheckFile(path)
		}
		if err != nil {
			log.Log.Reason(err).Errorf("Direct IO check failed for %s", path)
		} else if !supportDirectIO {
			log.Log.Infof("%s file system does not support direct I/O", path)
		}
		// when the disk is backed-up by another file, we need to also check if that
		// file sits on a file system that supports direct I/O
		if backingFile := disk.BackingStore; backingFile != nil {
			backingFilePath := backingFile.Source.File
			backFileDirectIOSupport, err := directIOChecker.CheckFile(backingFilePath)
			if err != nil {
				log.Log.Reason(err).Errorf("Direct IO check failed for %s", backingFilePath)
			} else if !backFileDirectIOSupport {
				log.Log.Infof("%s backing file system does not support direct I/O", backingFilePath)
			}
			supportDirectIO = supportDirectIO && backFileDirectIOSupport
		}
	}

	// if user set a cache mode = 'none' and fs does not support direct I/O then return an error
	if mode == v1.CacheNone && !supportDirectIO {
		return fmt.Errorf("Unable to use '%s' cache mode, file system where %s is stored does not support direct I/O", mode, path)
	}

	// if user did not set a cache mode and fs supports direct I/O then set cache = 'none'
	// else set cache = 'writethrough
	if mode == "" && supportDirectIO {
		mode = v1.CacheNone
	} else if mode == "" && !supportDirectIO {
		mode = v1.CacheWriteThrough
	}

	disk.Driver.Cache = string(mode)
	log.Log.Infof("Driver cache mode for %s set to %s", path, mode)

	return nil
}

func IsPreAllocated(path string) bool {
	diskInf, err := disk.GetDiskInfo(path)
	if err != nil {
		return false
	}
	// ActualSize can be a little larger then VirtualSize for qcow2
	return diskInf.VirtualSize <= diskInf.ActualSize
}

// Set optimal io mode automatically
func SetOptimalIOMode(disk *api.Disk, isPreAllocated func(path string) bool) {
	var path string

	// If the user explicitly set the io mode do nothing
	if disk.Driver.IO != "" {
		return
	}

	if disk.Source.File != "" {
		path = disk.Source.File
	} else if disk.Source.Dev != "" {
		path = disk.Source.Dev
	} else {
		return
	}

	// O_DIRECT is needed for io="native"
	if v1.DriverCache(disk.Driver.Cache) == v1.CacheNone {
		// set native for block device or pre-allocateed image file
		if (disk.Source.Dev != "") || isPreAllocated(disk.Source.File) {
			disk.Driver.IO = v1.IONative
		}
	}
	// For now we don't explicitly set io=threads even for sparse files as it's
	// not clear it's better for all use-cases
	if disk.Driver.IO != "" {
		log.Log.Infof("Driver IO mode for %s set to %s", path, disk.Driver.IO)
	}
}

func Convert_v1_Usbredir_To_api_Usbredir(vmi *v1.VirtualMachineInstance, domainDevices *api.Devices, _ *ConverterContext) error {
	clientDevices := vmi.Spec.Domain.Devices.ClientPassthrough

	// Default is to have USB Redirection disabled
	if clientDevices == nil {
		return nil
	}

	// Note that at the moment, we don't require any specific input to configure the USB devices
	// so we simply create the maximum allowed dictated by v1.UsbClientPassthroughMaxNumberOf
	redirectDevices := make([]api.RedirectedDevice, v1.UsbClientPassthroughMaxNumberOf)

	for i := 0; i < v1.UsbClientPassthroughMaxNumberOf; i++ {
		path := fmt.Sprintf("/var/run/kubevirt-private/%s/virt-usbredir-%d", vmi.ObjectMeta.UID, i)
		redirectDevices[i] = api.RedirectedDevice{
			Type: "unix",
			Bus:  "usb",
			Source: api.RedirectedDeviceSource{
				Mode: "bind",
				Path: path,
			},
		}
	}
	domainDevices.Redirs = redirectDevices
	return nil
}

func convertFeatureState(source *v1.FeatureState) *api.FeatureState {
	if source != nil {
		return &api.FeatureState{
			State: boolToOnOff(source.Enabled, true),
		}
	}
	return nil
}

func Convert_v1_Features_To_api_Features(source *v1.Features, features *api.Features, c *ConverterContext) error {
	if source.ACPI.Enabled == nil || *source.ACPI.Enabled {
		features.ACPI = &api.FeatureEnabled{}
	}
	if source.SMM != nil {
		if source.SMM.Enabled == nil || *source.SMM.Enabled {
			features.SMM = &api.FeatureEnabled{}
		}
	}
	if source.APIC != nil {
		if source.APIC.Enabled == nil || *source.APIC.Enabled {
			features.APIC = &api.FeatureEnabled{}
		}
	}
	if source.Hyperv != nil {
		features.Hyperv = &api.FeatureHyperv{}
		err := Convert_v1_FeatureHyperv_To_api_FeatureHyperv(source.Hyperv, features.Hyperv)
		if err != nil {
			return nil
		}
	} else if source.HypervPassthrough != nil && *source.HypervPassthrough.Enabled {
		features.Hyperv = &api.FeatureHyperv{
			Mode: api.HypervModePassthrough,
		}
	}
	if source.KVM != nil {
		features.KVM = &api.FeatureKVM{
			Hidden: &api.FeatureState{
				State: boolToOnOff(&source.KVM.Hidden, false),
			},
		}
	}
	if source.Pvspinlock != nil {
		features.PVSpinlock = &api.FeaturePVSpinlock{
			State: boolToOnOff(source.Pvspinlock.Enabled, true),
		}
	}

	if c.UseLaunchSecurityTDX {
		features.PMU = &api.FeatureState{
			State: "off",
		}
	}

	return nil
}

func Convert_v1_FeatureHyperv_To_api_FeatureHyperv(source *v1.FeatureHyperv, hyperv *api.FeatureHyperv) error {
	if source.Spinlocks != nil {
		hyperv.Spinlocks = &api.FeatureSpinlocks{
			State:   boolToOnOff(source.Spinlocks.Enabled, true),
			Retries: source.Spinlocks.Retries,
		}
	}
	if source.VendorID != nil {
		hyperv.VendorID = &api.FeatureVendorID{
			State: boolToOnOff(source.VendorID.Enabled, true),
			Value: source.VendorID.VendorID,
		}
	}

	hyperv.Relaxed = convertFeatureState(source.Relaxed)
	hyperv.Reset = convertFeatureState(source.Reset)
	hyperv.Runtime = convertFeatureState(source.Runtime)
	hyperv.SyNIC = convertFeatureState(source.SyNIC)
	hyperv.SyNICTimer = convertV1ToAPISyNICTimer(source.SyNICTimer)
	hyperv.VAPIC = convertFeatureState(source.VAPIC)
	hyperv.VPIndex = convertFeatureState(source.VPIndex)
	hyperv.Frequencies = convertFeatureState(source.Frequencies)
	hyperv.Reenlightenment = convertFeatureState(source.Reenlightenment)
	hyperv.TLBFlush = convertFeatureState(source.TLBFlush)
	hyperv.IPI = convertFeatureState(source.IPI)
	hyperv.EVMCS = convertFeatureState(source.EVMCS)
	return nil
}

func convertV1ToAPISyNICTimer(syNICTimer *v1.SyNICTimer) *api.SyNICTimer {
	if syNICTimer == nil {
		return nil
	}

	result := &api.SyNICTimer{
		State: boolToOnOff(syNICTimer.Enabled, true),
	}

	if syNICTimer.Direct != nil {
		result.Direct = &api.FeatureState{
			State: boolToOnOff(syNICTimer.Direct.Enabled, true),
		}
	}
	return result
}

func initializeQEMUCmdAndQEMUArg(domain *api.Domain) {
	if domain.Spec.QEMUCmd == nil {
		domain.Spec.QEMUCmd = &api.Commandline{}
	}

	if domain.Spec.QEMUCmd.QEMUArg == nil {
		domain.Spec.QEMUCmd.QEMUArg = make([]api.Arg, 0)
	}
}

func setupDomainMemory(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Memory == nil ||
		vmi.Spec.Domain.Memory.MaxGuest == nil ||
		vmi.Spec.Domain.Memory.Guest.Equal(*vmi.Spec.Domain.Memory.MaxGuest) {
		var err error

		domain.Spec.Memory, err = vcpu.QuantityToByte(*vcpu.GetVirtualMemory(vmi))
		if err != nil {
			return err
		}
		return nil
	}

	maxMemory, err := vcpu.QuantityToByte(*vmi.Spec.Domain.Memory.MaxGuest)
	if err != nil {
		return err
	}

	domain.Spec.MaxMemory = &api.MaxMemory{
		Unit:  maxMemory.Unit,
		Value: maxMemory.Value,
	}

	currentMemory, err := vcpu.QuantityToByte(*vmi.Spec.Domain.Memory.Guest)
	if err != nil {
		return err
	}

	domain.Spec.Memory = currentMemory

	return nil
}

func Convert_v1_Firmware_To_related_apis(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *ConverterContext) error {
	firmware := vmi.Spec.Domain.Firmware
	if firmware == nil {
		return nil
	}

	domain.Spec.SysInfo.System = []api.Entry{
		{
			Name:  "uuid",
			Value: string(firmware.UUID),
		},
	}

	if vmi.IsBootloaderEFI() {
		domain.Spec.OS.BootLoader = &api.Loader{
			Path:     c.EFIConfiguration.EFICode,
			ReadOnly: "yes",
			Secure:   boolToYesNo(&c.EFIConfiguration.SecureLoader, false),
		}

		if util.IsSEVSNPVMI(vmi) || util.IsTDXVMI(vmi) {
			// Use stateless firmware for the TDX/SNP VMs
			domain.Spec.OS.BootLoader.Type = "rom"
			domain.Spec.OS.NVRam = nil
		} else {
			domain.Spec.OS.BootLoader.Type = "pflash"
			domain.Spec.OS.NVRam = &api.NVRam{
				Template: c.EFIConfiguration.EFIVars,
				NVRam:    filepath.Join(services.PathForNVram(vmi), vmi.Name+"_VARS.fd"),
			}
		}
	}

	if firmware.Bootloader != nil && firmware.Bootloader.BIOS != nil {
		if firmware.Bootloader.BIOS.UseSerial != nil && *firmware.Bootloader.BIOS.UseSerial {
			domain.Spec.OS.BIOS = &api.BIOS{
				UseSerial: "yes",
			}
		}
	}

	if len(firmware.Serial) > 0 {
		domain.Spec.SysInfo.System = append(domain.Spec.SysInfo.System, api.Entry{
			Name:  "serial",
			Value: firmware.Serial,
		})
	}

	if util.HasKernelBootContainerImage(vmi) {
		kb := firmware.KernelBoot

		log.Log.Object(vmi).Infof("kernel boot defined for VMI. Converting to domain XML")
		if kb.Container.KernelPath != "" {
			kernelPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(kb.Container.KernelPath)
			log.Log.Object(vmi).Infof("setting kernel path for kernel boot: %s", kernelPath)
			domain.Spec.OS.Kernel = kernelPath
		}

		if kb.Container.InitrdPath != "" {
			initrdPath := containerdisk.GetKernelBootArtifactPathFromLauncherView(kb.Container.InitrdPath)
			log.Log.Object(vmi).Infof("setting initrd path for kernel boot: %s", initrdPath)
			domain.Spec.OS.Initrd = initrdPath
		}

	}

	// Define custom command-line arguments even if kernel-boot container is not defined
	if firmware.KernelBoot != nil {
		log.Log.Object(vmi).Infof("setting custom kernel arguments: %s", firmware.KernelBoot.KernelArgs)
		domain.Spec.OS.KernelArgs = firmware.KernelBoot.KernelArgs
	}

	if err := Convert_v1_Firmware_ACPI_To_related_apis(firmware, domain, vmi.Spec.Volumes); err != nil {
		return err
	}

	return nil
}

func Convert_v1_Firmware_ACPI_To_related_apis(firmware *v1.Firmware, domain *api.Domain, volumes []v1.Volume) error {
	if firmware.ACPI == nil {
		return nil
	}

	if firmware.ACPI.SlicNameRef == "" && firmware.ACPI.MsdmNameRef == "" {
		return fmt.Errorf("No ACPI tables were set. Expecting at least one.")
	}

	if domain.Spec.OS.ACPI == nil {
		domain.Spec.OS.ACPI = &api.OSACPI{}
	}

	if val, err := createACPITable("slic", firmware.ACPI.SlicNameRef, volumes); err != nil {
		return err
	} else if val != nil {
		domain.Spec.OS.ACPI.Table = append(domain.Spec.OS.ACPI.Table, *val)
	}

	if val, err := createACPITable("msdm", firmware.ACPI.MsdmNameRef, volumes); err != nil {
		return err
	} else if val != nil {
		domain.Spec.OS.ACPI.Table = append(domain.Spec.OS.ACPI.Table, *val)
	}

	// if field was set but volume was not found, helper function will return error
	return nil
}

func createACPITable(source, volumeName string, volumes []v1.Volume) (*api.ACPITable, error) {
	if volumeName == "" {
		return nil, nil
	}

	for _, volume := range volumes {
		if volume.Name != volumeName {
			continue
		}

		if volume.Secret == nil {
			// Unsupported. This should have been blocked by webhook, so warn user.
			return nil, fmt.Errorf("Firmware's volume type is unsupported for %s", source)
		}

		// Return path to table's binary data
		sourcePath := config.GetSecretSourcePath(volumeName)
		sourcePath = filepath.Join(sourcePath, fmt.Sprintf("%s.bin", source))
		return &api.ACPITable{
			Type: source,
			Path: sourcePath,
		}, nil
	}

	return nil, fmt.Errorf("Firmware's volume for %s was not found", source)
}

func hasIOThreads(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		return true
	}
	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			return true
		}
	}
	return false
}

func getIOThreadsCountType(vmi *v1.VirtualMachineInstance) (ioThreadCount, autoThreads int) {
	dedicatedThreads := 0

	var threadPoolLimit int
	policy := vmi.Spec.Domain.IOThreadsPolicy
	switch {
	case policy == nil:
		threadPoolLimit = 1
	case *policy == v1.IOThreadsPolicyShared:
		threadPoolLimit = 1
	case *policy == v1.IOThreadsPolicyAuto:
		// When IOThreads policy is set to auto and we've allocated a dedicated
		// pCPU for the emulator thread, we can place IOThread and Emulator thread in the same pCPU
		if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
			threadPoolLimit = 1
		} else {
			numCPUs := 1
			// Requested CPU's is guaranteed to be no greater than the limit
			if cpuRequests, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU]; ok {
				numCPUs = int(cpuRequests.Value())
			} else if cpuLimit, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]; ok {
				numCPUs = int(cpuLimit.Value())
			}

			threadPoolLimit = numCPUs * 2
		}
	case *policy == v1.IOThreadsPolicySupplementalPool:
		if vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
			ioThreadCount = int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount)
		}
		return
	}

	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		if diskDevice.DedicatedIOThread != nil && *diskDevice.DedicatedIOThread {
			dedicatedThreads += 1
		} else {
			autoThreads += 1
		}
	}

	if (autoThreads + dedicatedThreads) > threadPoolLimit {
		autoThreads = threadPoolLimit - dedicatedThreads
		// We need at least one shared thread
		if autoThreads < 1 {
			autoThreads = 1
		}
	}

	ioThreadCount = autoThreads + dedicatedThreads
	return
}

func setIOThreads(vmi *v1.VirtualMachineInstance, domain *api.Domain, vcpus uint) {
	if !hasIOThreads(vmi) {
		return
	}
	currentAutoThread := defaultIOThread
	ioThreadCount, autoThreads := getIOThreadsCountType(vmi)
	if ioThreadCount != 0 {
		if domain.Spec.IOThreads == nil {
			domain.Spec.IOThreads = &api.IOThreads{}
		}
		domain.Spec.IOThreads.IOThreads = uint(ioThreadCount)
	}
	if vmi.Spec.Domain.IOThreadsPolicy != nil &&
		*vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool {
		iothreads := &api.DiskIOThreads{}
		for id := 1; id <= int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount); id++ {
			iothreads.IOThread = append(iothreads.IOThread, api.DiskIOThread{Id: uint32(id)})
		}
		for i, disk := range domain.Spec.Devices.Disks {
			// Only disks with virtio bus support IOThreads
			if disk.Target.Bus == v1.DiskBusVirtio {
				domain.Spec.Devices.Disks[i].Driver.IOThreads = iothreads
			}
		}
	} else {
		currentDedicatedThread := uint(autoThreads + 1)
		for i, disk := range domain.Spec.Devices.Disks {
			// Only disks with virtio bus support IOThreads
			if disk.Target.Bus == v1.DiskBusVirtio {
				if vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread != nil && *vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread {
					domain.Spec.Devices.Disks[i].Driver.IOThread = pointer.P(currentDedicatedThread)
					currentDedicatedThread += 1
				} else {
					domain.Spec.Devices.Disks[i].Driver.IOThread = pointer.P(currentAutoThread)
					// increment the threadId to be used next but wrap around at the thread limit
					// the odd math here is because thread ID's start at 1, not 0
					currentAutoThread = (currentAutoThread % uint(autoThreads)) + 1
				}
			}
		}
	}

	// Virtio-scsi doesn't support IO threads yet, only the SCSI controller supports.
	setIOThreadSCSIController := false
	for i, disk := range domain.Spec.Devices.Disks {
		// Only disks with virtio bus support IOThreads
		if disk.Target.Bus == v1.DiskBusSCSI {
			if vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread != nil && *vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread {
				setIOThreadSCSIController = true
				break
			}
		}
	}
	if !setIOThreadSCSIController {
		return
	}
	for i, controller := range domain.Spec.Devices.Controllers {
		if controller.Type == "scsi" {
			if controller.Driver == nil {
				domain.Spec.Devices.Controllers[i].Driver = &api.ControllerDriver{}
			}
			domain.Spec.Devices.Controllers[i].Driver.IOThread = pointer.P(currentAutoThread)
			domain.Spec.Devices.Controllers[i].Driver.Queues = pointer.P(vcpus)
		}
	}
}

func Convert_v1_VirtualMachineInstance_To_api_Domain(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *ConverterContext) (err error) {
	var controllerDriver *api.ControllerDriver

	precond.MustNotBeNil(vmi)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	architecture := c.Architecture.GetArchitecture()
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)

	builder := NewDomainBuilder(
		metadata.DomainConfigurator{},
		network.NewDomainConfigurator(
			network.WithDomainAttachmentByInterfaceName(c.DomainAttachmentByInterfaceName),
			network.WithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			network.WithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
		),
		compute.TPMDomainConfigurator{},
		compute.VSOCKDomainConfigurator{},
		compute.NewLaunchSecurityDomainConfigurator(architecture),
		compute.ChannelsDomainConfigurator{},
		compute.ClockDomainConfigurator{},
		compute.NewRNGDomainConfigurator(
			compute.RNGWithArchitecture(architecture),
			compute.RNGWithUseVirtioTransitional(c.UseVirtioTransitional),
			compute.RNGWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			compute.RNGWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
		),
		compute.NewInputDeviceDomainConfigurator(architecture),
		compute.NewBalloonDomainConfigurator(
			compute.BalloonWithArchitecture(architecture),
			compute.BalloonWithUseVirtioTransitional(c.UseVirtioTransitional),
			compute.BalloonWithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			compute.BalloonWithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			compute.BalloonWithFreePageReporting(c.FreePageReporting),
			compute.BalloonWithMemBalloonStatsPeriod(c.MemBalloonStatsPeriod),
		),
		compute.NewGraphicsDomainConfigurator(architecture, c.BochsForEFIGuests),
		compute.SoundDomainConfigurator{},
		compute.NewHostDeviceDomainConfigurator(
			c.GenericHostDevices,
			c.GPUHostDevices,
			c.SRIOVDevices,
		),
		compute.NewWatchdogDomainConfigurator(architecture),
		compute.NewConsoleDomainConfigurator(c.SerialConsoleLog),
		compute.PanicDevicesDomainConfigurator{},
		storage.NewDiskConfigurator(
			storage.WithArchitecture(architecture),
			storage.WithHotplugVolumes(c.HotplugVolumes),
			storage.WithPermanentVolumes(c.PermanentVolumes),
			storage.WithDisksInfo(c.DisksInfo),
			storage.WithIsBlockPVC(c.IsBlockPVC),
			storage.WithIsBlockDV(c.IsBlockDV),
			storage.WithApplyCBT(c.ApplyCBT),
			storage.WithUseVirtioTransitional(c.UseVirtioTransitional),
			storage.WithUseLaunchSecuritySEV(c.UseLaunchSecuritySEV),
			storage.WithUseLaunchSecurityPV(c.UseLaunchSecurityPV),
			storage.WithExpandDisksEnabled(c.ExpandDisksEnabled),
			storage.WithUseBlkMQ(c.UseBlkMQ),
			storage.WithVcpus(uint(cpuCount)),
			storage.WithVolumesDiscardIgnore(c.VolumesDiscardIgnore),
			storage.WithEphemeralDiskCreator(c.EphemeraldiskCreator),
		),
	)
	if err := builder.Build(vmi, domain); err != nil {
		return err
	}

	// Set VM CPU cores
	// CPU topology will be created everytime, because user can specify
	// number of cores in vmi.Spec.Domain.Resources.Requests/Limits, not only
	// in vmi.Spec.Domain.CPU

	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}
	// set the maximum number of sockets here to allow hot-plug CPUs
	if vmiCPU := vmi.Spec.Domain.CPU; vmiCPU != nil && vmiCPU.MaxSockets != 0 && c.Architecture.SupportCPUHotplug() {
		domainVCPUTopologyForHotplug(vmi, domain)
	}

	kvmPath := "/dev/kvm"
	if _, err := os.Stat(kvmPath); errors.Is(err, os.ErrNotExist) {
		if c.AllowEmulation {
			logger := log.DefaultLogger()
			logger.Infof("Hardware emulation device '%s' not present. Using software emulation.", kvmPath)
			domain.Spec.Type = "qemu"
		} else {
			return fmt.Errorf("hardware emulation device '%s' not present", kvmPath)
		}
	} else if err != nil {
		return err
	}

	domain.Spec.SysInfo = &api.SysInfo{}

	err = Convert_v1_Firmware_To_related_apis(vmi, domain, c)
	if err != nil {
		return err
	}

	if c.UseLaunchSecuritySEV || c.UseLaunchSecurityPV {
		controllerDriver = &api.ControllerDriver{
			IOMMU: "on",
		}
	}

	if c.SMBios != nil {
		domain.Spec.SysInfo.System = append(domain.Spec.SysInfo.System,
			api.Entry{
				Name:  "manufacturer",
				Value: c.SMBios.Manufacturer,
			},
			api.Entry{
				Name:  "family",
				Value: c.SMBios.Family,
			},
			api.Entry{
				Name:  "product",
				Value: c.SMBios.Product,
			},
			api.Entry{
				Name:  "sku",
				Value: c.SMBios.Sku,
			},
			api.Entry{
				Name:  "version",
				Value: c.SMBios.Version,
			},
		)
	}

	// Take SMBios values from the VirtualMachineOptions
	if c.Architecture.IsSMBiosNeeded() {
		domain.Spec.OS.SMBios = &api.SMBios{
			Mode: "sysinfo",
		}
	}

	if vmi.Spec.Domain.Chassis != nil {
		domain.Spec.SysInfo.Chassis = []api.Entry{
			{
				Name:  "manufacturer",
				Value: vmi.Spec.Domain.Chassis.Manufacturer,
			},
			{
				Name:  "version",
				Value: vmi.Spec.Domain.Chassis.Version,
			},
			{
				Name:  "serial",
				Value: vmi.Spec.Domain.Chassis.Serial,
			},
			{
				Name:  "asset",
				Value: vmi.Spec.Domain.Chassis.Asset,
			},
			{
				Name:  "sku",
				Value: vmi.Spec.Domain.Chassis.Sku,
			},
		}
	}

	if err = setupDomainMemory(vmi, domain); err != nil {
		return err
	}

	var isMemfdRequired = false
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		domain.Spec.MemoryBacking = &api.MemoryBacking{
			HugePages: &api.HugePages{},
		}
		if val := vmi.Annotations[v1.MemfdMemoryBackend]; val != "false" {
			isMemfdRequired = true
		}
	}
	// virtiofs require shared access
	if util.IsVMIVirtiofsEnabled(vmi) {
		if domain.Spec.MemoryBacking == nil {
			domain.Spec.MemoryBacking = &api.MemoryBacking{}
		}
		domain.Spec.MemoryBacking.Access = &api.MemoryBackingAccess{
			Mode: "shared",
		}
		isMemfdRequired = true
	}

	if isMemfdRequired {
		// Set memfd as memory backend to solve SELinux restrictions
		// See the issue: https://github.com/kubevirt/kubevirt/issues/3781
		domain.Spec.MemoryBacking.Source = &api.MemoryBackingSource{Type: "memfd"}

		// NUMA is required in order to use memfd
		if domain.Spec.CPU.NUMA == nil {
			domain.Spec.CPU.NUMA = &api.NUMA{
				Cells: []api.NUMACell{
					{
						ID:     "0",
						CPUs:   fmt.Sprintf("0-%d", domain.Spec.VCPU.CPUs-1),
						Memory: uint64(vcpu.GetVirtualMemory(vmi).Value() / int64(1024)),
						Unit:   "KiB",
					},
				},
			}
		}
	}

	volumeIndices := map[string]int{}
	volumes := map[string]*v1.Volume{}
	for i, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
		volumeIndices[volume.Name] = i
	}

	// Handle virtioFS
	domain.Spec.Devices.Filesystems = append(domain.Spec.Devices.Filesystems, convertFileSystems(vmi.Spec.Domain.Devices.Filesystems)...)

	err = Convert_v1_Usbredir_To_api_Usbredir(vmi, &domain.Spec.Devices, c)
	if err != nil {
		return err
	}

	// Creating USB controller, disabled by default
	usbController := api.Controller{
		Type:  "usb",
		Index: "0",
		Model: "none",
	}
	if c.Architecture.IsUSBNeeded(vmi) {
		usbController.Model = "qemu-xhci"
	}
	domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, usbController)

	if needsSCSIController(vmi) {
		scsiController := c.Architecture.ScsiController(virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture()), controllerDriver)
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, scsiController)
	}

	if c.Architecture.SupportPCIHole64Disabling() && shouldDisablePCIHole64(vmi) {
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers,
			api.Controller{
				Type:  "pci",
				Index: "0",
				Model: "pcie-root",
				PCIHole64: &api.PCIHole64{
					Value: 0,
					Unit:  "KiB",
				},
			},
		)
	}

	if vmi.Spec.Domain.Features != nil {
		domain.Spec.Features = &api.Features{}
		err := Convert_v1_Features_To_api_Features(vmi.Spec.Domain.Features, domain.Spec.Features, c)

		if c.Architecture.HasVMPort() {
			domain.Spec.Features.VMPort = &api.FeatureState{State: "off"}
		}

		if err != nil {
			return err
		}
	}

	if machine := vmi.Spec.Domain.Machine; machine != nil {
		domain.Spec.OS.Type.Machine = machine.Type
	}

	if vmi.Spec.Domain.CPU != nil {
		// Set VM CPU model and vendor
		if vmi.Spec.Domain.CPU.Model != "" {
			if vmi.Spec.Domain.CPU.Model == v1.CPUModeHostModel || vmi.Spec.Domain.CPU.Model == v1.CPUModeHostPassthrough {
				domain.Spec.CPU.Mode = vmi.Spec.Domain.CPU.Model
			} else {
				domain.Spec.CPU.Mode = "custom"
				domain.Spec.CPU.Model = vmi.Spec.Domain.CPU.Model
			}
		}

		// Set VM CPU features
		existingFeatures := make(map[string]struct{})
		if vmi.Spec.Domain.CPU.Features != nil {
			for _, feature := range vmi.Spec.Domain.CPU.Features {
				existingFeatures[feature.Name] = struct{}{}
				domain.Spec.CPU.Features = append(domain.Spec.CPU.Features, api.CPUFeature{
					Name:   feature.Name,
					Policy: feature.Policy,
				})
			}
		}

		/*
						Libvirt validation fails when a CPU model is usable
						by QEMU but lacks features listed in
						`/usr/share/libvirt/cpu_map/[CPU Model].xml` on a node
						To avoid the validation error mentioned above we can disable
						deprecated features in the `/usr/share/libvirt/cpu_map/[CPU Model].xml` files.
						Examples of validation error:
			    		https://bugzilla.redhat.com/show_bug.cgi?id=2122283 - resolve by obsolete Opteron_G2
						https://gitlab.com/libvirt/libvirt/-/issues/304 - resolve by disabling mpx which is deprecated
						Issue in Libvirt: https://gitlab.com/libvirt/libvirt/-/issues/608
						once the issue is resolved we can remove mpx disablement
		*/

		_, exists := existingFeatures["mpx"]
		if c.Architecture.RequiresMPXCPUValidation() && !exists && vmi.Spec.Domain.CPU.Model != v1.CPUModeHostModel && vmi.Spec.Domain.CPU.Model != v1.CPUModeHostPassthrough {
			domain.Spec.CPU.Features = append(domain.Spec.CPU.Features, api.CPUFeature{
				Name:   "mpx",
				Policy: "disable",
			})
		}

		// Adjust guest vcpu config. Currently will handle vCPUs to pCPUs pinning
		if vmi.IsCPUDedicated() {
			err = vcpu.AdjustDomainForTopologyAndCPUSet(domain, vmi, c.Topology, c.CPUSet, hasIOThreads(vmi))
			if err != nil {
				return err
			}
		}
	}

	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		domain.Spec.CPU.Mode = v1.CPUModeHostModel
	}

	if vmi.Spec.Domain.Devices.AutoattachSerialConsole == nil || *vmi.Spec.Domain.Devices.AutoattachSerialConsole {
		// Add mandatory console device
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, api.Controller{
			Type:   "virtio-serial",
			Index:  "0",
			Model:  virtio.InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture.GetArchitecture()),
			Driver: controllerDriver,
		})
	}

	// Add Ignition Command Line if present
	ignitiondata := vmi.Annotations[v1.IgnitionAnnotation]
	if ignitiondata != "" && strings.Contains(ignitiondata, "ignition") {
		initializeQEMUCmdAndQEMUArg(domain)
		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-fw_cfg"})
		ignitionpath := fmt.Sprintf("%s/%s", ignition.GetDomainBasePath(c.VirtualMachine.Name, c.VirtualMachine.Namespace), ignition.IgnitionFile)
		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: fmt.Sprintf("name=opt/com.coreos/config,file=%s", ignitionpath)})
	}

	if val := vmi.Annotations[v1.PlacePCIDevicesOnRootComplex]; val == "true" {
		if err := PlacePCIDevicesOnRootComplex(&domain.Spec); err != nil {
			return err
		}
	}

	if c.Architecture.ShouldVerboseLogsBeEnabled() {
		virtLauncherLogVerbosity, err := strconv.Atoi(os.Getenv(services.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY))
		if err == nil && virtLauncherLogVerbosity > services.EXT_LOG_VERBOSITY_THRESHOLD {
			// isa-debugcon device is only for x86_64
			initializeQEMUCmdAndQEMUArg(domain)

			domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg,
				api.Arg{Value: "-chardev"},
				api.Arg{Value: fmt.Sprintf("file,id=firmwarelog,path=%s", QEMUSeaBiosDebugPipe)},
				api.Arg{Value: "-device"},
				api.Arg{Value: "isa-debugcon,iobase=0x402,chardev=firmwarelog"})
		}
	}

	// set bootmenu to give time to access bios
	if vmi.ShouldStartPaused() {
		domain.Spec.OS.BootMenu = &api.BootMenu{
			Enable:  "yes",
			Timeout: pointer.P(bootMenuTimeoutMS),
		}
	}

	vcpus := uint(cpuCount)
	if vcpus == 0 {
		vcpus = uint(1)
	}

	setIOThreads(vmi, domain, vcpus)

	return nil
}

func boolToOnOff(value *bool, defaultOn bool) string {
	return boolToString(value, defaultOn, "on", "off")
}

func boolToYesNo(value *bool, defaultYes bool) string {
	return boolToString(value, defaultYes, "yes", "no")
}

func boolToString(value *bool, defaultPositive bool, positive string, negative string) string {
	toString := func(value bool) string {
		if value {
			return positive
		}
		return negative
	}

	if value == nil {
		return toString(defaultPositive)
	}
	return toString(*value)
}

func needsSCSIController(vmi *v1.VirtualMachineInstance) bool {
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if getBusFromDisk(disk) == v1.DiskBusSCSI {
			return true
		}
	}
	return !vmi.Spec.Domain.Devices.DisableHotplug
}

func shouldDisablePCIHole64(vmi *v1.VirtualMachineInstance) bool {
	if val, ok := vmi.Annotations[v1.DisablePCIHole64]; ok {
		return strings.EqualFold(val, "true")
	}
	return false
}

func getBusFromDisk(disk v1.Disk) v1.DiskBus {
	if disk.LUN != nil {
		return disk.LUN.Bus
	}
	if disk.Disk != nil {
		return disk.Disk.Bus
	}
	if disk.CDRom != nil {
		return disk.CDRom.Bus
	}
	return ""
}

func getPrefixFromBus(bus v1.DiskBus) string {
	switch bus {
	case v1.DiskBusVirtio:
		return "vd"
	case v1.DiskBusSATA, v1.DiskBusSCSI, v1.DiskBusUSB:
		return "sd"
	default:
		log.Log.Errorf("Unrecognized bus '%s'", bus)
		return ""
	}
}

func newDeviceNamer(volumeStatuses []v1.VolumeStatus, disks []v1.Disk) map[string]deviceNamer {
	prefixMap := make(map[string]deviceNamer)
	volumeTargetMap := make(map[string]string)
	for _, volumeStatus := range volumeStatuses {
		if volumeStatus.Target != "" {
			volumeTargetMap[volumeStatus.Name] = volumeStatus.Target
		}
	}

	for _, disk := range disks {
		if disk.Disk == nil {
			continue
		}
		prefix := getPrefixFromBus(disk.Disk.Bus)
		if _, ok := prefixMap[prefix]; !ok {
			prefixMap[prefix] = deviceNamer{
				existingNameMap: make(map[string]string),
				usedDeviceMap:   make(map[string]string),
			}
		}
		namer := prefixMap[prefix]
		if _, ok := volumeTargetMap[disk.Name]; ok {
			namer.existingNameMap[disk.Name] = volumeTargetMap[disk.Name]
			namer.usedDeviceMap[volumeTargetMap[disk.Name]] = disk.Name
		}
	}
	return prefixMap
}

func GetVolumeNameByDisk(disk api.Disk) string {
	return disk.Alias.GetName()
}

// GetVolumeNameByTarget returns the volume name associated to the device target in the domain (e.g vda)
func GetVolumeNameByTarget(domain *api.Domain, target string) string {
	for _, d := range domain.Spec.Devices.Disks {
		if d.Target.Device == target {
			return GetVolumeNameByDisk(d)
		}
	}
	return ""
}

func GracePeriodSeconds(vmi *v1.VirtualMachineInstance) int64 {
	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}
	return gracePeriodSeconds
}

func domainVCPUTopologyForHotplug(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	// Always allow to hotplug to minimum of 1 socket
	minEnabledCpuCount := cpuTopology.Cores * cpuTopology.Threads
	// Total vCPU count
	enabledCpuCount := cpuCount
	cpuTopology.Sockets = vmi.Spec.Domain.CPU.MaxSockets
	cpuCount = vcpu.CalculateRequestedVCPUs(cpuTopology)
	VCPUs := &api.VCPUs{}
	for id := uint32(0); id < cpuCount; id++ {
		// Enable all requestd vCPUs
		isEnabled := id < enabledCpuCount
		// There should not be fewer vCPU than cores and threads within a single socket
		isHotpluggable := id >= minEnabledCpuCount
		vcpu := api.VCPUsVCPU{
			ID:           id,
			Enabled:      boolToYesNo(&isEnabled, true),
			Hotpluggable: boolToYesNo(&isHotpluggable, false),
		}
		VCPUs.VCPU = append(VCPUs.VCPU, vcpu)
	}

	domain.Spec.VCPUs = VCPUs
	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}
}
