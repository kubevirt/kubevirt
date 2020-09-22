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

package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/ignition"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
)

const (
	CPUModeHostPassthrough = "host-passthrough"
	CPUModeHostModel       = "host-model"
	defaultIOThread        = uint(1)
	EFICode                = "OVMF_CODE.fd"
	EFIVars                = "OVMF_VARS.fd"
	EFICodeSecureBoot      = "OVMF_CODE.secboot.fd"
	EFIVarsSecureBoot      = "OVMF_VARS.secboot.fd"
)

// +k8s:deepcopy-gen=false
type ConverterContext struct {
	Architecture          string
	UseEmulation          bool
	Secrets               map[string]*k8sv1.Secret
	VirtualMachine        *v1.VirtualMachineInstance
	CPUSet                []int
	IsBlockPVC            map[string]bool
	IsBlockDV             map[string]bool
	DiskType              map[string]*containerdisk.DiskInfo
	SRIOVDevices          map[string][]string
	SMBios                *cmdv1.SMBios
	GpuDevices            []string
	VgpuDevices           []string
	EmulatorThreadCpu     *int
	OVMFPath              string
	MemBalloonStatsPeriod uint
}

func Convert_v1_Disk_To_api_Disk(diskDevice *v1.Disk, disk *Disk, devicePerBus map[string]int, numQueues *uint) error {

	if diskDevice.Disk != nil {
		disk.Device = "disk"
		disk.Target.Bus = diskDevice.Disk.Bus
		disk.Target.Device = makeDeviceName(diskDevice.Disk.Bus, devicePerBus)
		if diskDevice.Disk.PciAddress != "" {
			if diskDevice.Disk.Bus != "virtio" {
				return fmt.Errorf("setting a pci address is not allowed for non-virtio bus types, for disk %s", diskDevice.Name)
			}
			addr, err := decoratePciAddressField(diskDevice.Disk.PciAddress)
			if err != nil {
				return fmt.Errorf("failed to configure disk %s: %v", diskDevice.Name, err)
			}
			disk.Address = addr
		}
		disk.ReadOnly = toApiReadOnly(diskDevice.Disk.ReadOnly)
		disk.Serial = diskDevice.Serial
	} else if diskDevice.LUN != nil {
		disk.Device = "lun"
		disk.Target.Bus = diskDevice.LUN.Bus
		disk.Target.Device = makeDeviceName(diskDevice.LUN.Bus, devicePerBus)
		disk.ReadOnly = toApiReadOnly(diskDevice.LUN.ReadOnly)
	} else if diskDevice.Floppy != nil {
		disk.Device = "floppy"
		disk.Target.Bus = "fdc"
		disk.Target.Tray = string(diskDevice.Floppy.Tray)
		disk.Target.Device = makeDeviceName(disk.Target.Bus, devicePerBus)
		disk.ReadOnly = toApiReadOnly(diskDevice.Floppy.ReadOnly)
	} else if diskDevice.CDRom != nil {
		disk.Device = "cdrom"
		disk.Target.Tray = string(diskDevice.CDRom.Tray)
		disk.Target.Bus = diskDevice.CDRom.Bus
		disk.Target.Device = makeDeviceName(diskDevice.CDRom.Bus, devicePerBus)
		if diskDevice.CDRom.ReadOnly != nil {
			disk.ReadOnly = toApiReadOnly(*diskDevice.CDRom.ReadOnly)
		} else {
			disk.ReadOnly = toApiReadOnly(true)
		}
	}
	disk.Driver = &DiskDriver{
		Name:  "qemu",
		Cache: string(diskDevice.Cache),
		IO:    string(diskDevice.IO),
	}
	if numQueues != nil && disk.Target.Bus == "virtio" {
		disk.Driver.Queues = numQueues
	}
	disk.Alias = &Alias{Name: diskDevice.Name}
	if diskDevice.BootOrder != nil {
		disk.BootOrder = &BootOrder{Order: *diskDevice.BootOrder}
	}

	return nil
}

func checkDirectIOFlag(path string) bool {
	// check if fs where disk.img file is located or block device
	// support direct i/o
	f, err := os.OpenFile(path, syscall.O_RDONLY|syscall.O_DIRECT, 0)
	if err != nil && !os.IsNotExist(err) {
		return false
	}
	defer f.Close()
	return true
}

func SetDriverCacheMode(disk *Disk) error {
	var path string
	supportDirectIO := true
	mode := v1.DriverCache(disk.Driver.Cache)

	if disk.Source.File != "" {
		path = disk.Source.File
	} else if disk.Source.Dev != "" {
		path = disk.Source.Dev
	} else {
		return fmt.Errorf("Unable to set a driver cache mode, disk is neither a block device nor a file")
	}

	if mode == "" || mode == v1.CacheNone {
		supportDirectIO = checkDirectIOFlag(path)
		if !supportDirectIO {
			log.Log.Infof("%s file system does not support direct I/O", path)
		}
		// when the disk is backed-up by another file, we need to also check if that
		// file sits on a file system that supports direct I/O
		if backingFile := disk.BackingStore; backingFile != nil {
			backingFilePath := backingFile.Source.File
			backFileDirectIOSupport := checkDirectIOFlag(backingFilePath)
			log.Log.Infof("%s backing file system does not support direct I/O", backingFilePath)
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

func makeDeviceName(bus string, devicePerBus map[string]int) string {
	index := devicePerBus[bus]
	devicePerBus[bus] += 1

	prefix := ""
	switch bus {
	case "virtio":
		prefix = "vd"
	case "sata", "scsi":
		prefix = "sd"
	case "fdc":
		prefix = "fd"
	default:
		log.Log.Errorf("Unrecognized bus '%s'", bus)
		return ""
	}
	return formatDeviceName(prefix, index)
}

// port of http://elixir.free-electrons.com/linux/v4.15/source/drivers/scsi/sd.c#L3211
func formatDeviceName(prefix string, index int) string {
	base := int('z' - 'a' + 1)
	name := ""

	for index >= 0 {
		name = string('a'+(index%base)) + name
		index = (index / base) - 1
	}
	return prefix + name
}

func toApiReadOnly(src bool) *ReadOnly {
	if src {
		return &ReadOnly{}
	}
	return nil
}

// Add_Agent_To_api_Channel creates the channel for guest agent communication
func Add_Agent_To_api_Channel() (channel Channel) {
	channel.Type = "unix"
	// let libvirt decide which path to use
	channel.Source = nil
	channel.Target = &ChannelTarget{
		Name: "org.qemu.guest_agent.0",
		Type: "virtio",
	}

	return
}

func Convert_v1_Volume_To_api_Disk(source *v1.Volume, disk *Disk, c *ConverterContext, diskIndex int) error {

	if source.ContainerDisk != nil {
		return Convert_v1_ContainerDiskSource_To_api_Disk(source.Name, source.ContainerDisk, disk, c, diskIndex)
	}

	if source.CloudInitNoCloud != nil || source.CloudInitConfigDrive != nil {
		return Convert_v1_CloudInitSource_To_api_Disk(source.VolumeSource, disk, c)
	}

	if source.HostDisk != nil {
		return Convert_v1_HostDisk_To_api_Disk(source.Name, source.HostDisk.Path, disk, c)
	}

	if source.PersistentVolumeClaim != nil {
		return Convert_v1_PersistentVolumeClaim_To_api_Disk(source.Name, disk, c)
	}

	if source.DataVolume != nil {
		return Convert_v1_DataVolume_To_api_Disk(source.Name, disk, c)
	}

	if source.Ephemeral != nil {
		return Convert_v1_EphemeralVolumeSource_To_api_Disk(source.Name, source.Ephemeral, disk, c)
	}
	if source.EmptyDisk != nil {
		return Convert_v1_EmptyDiskSource_To_api_Disk(source.Name, source.EmptyDisk, disk, c)
	}
	if source.ConfigMap != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.ConfigMap)
	}
	if source.Secret != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.Secret)
	}
	if source.ServiceAccount != nil {
		return Convert_v1_Config_To_api_Disk(source.Name, disk, config.ServiceAccount)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.Name)
}

func Convert_v1_Config_To_api_Disk(volumeName string, disk *Disk, configType config.Type) error {
	disk.Type = "file"
	disk.Driver.Type = "raw"
	switch configType {
	case config.ConfigMap:
		disk.Source.File = config.GetConfigMapDiskPath(volumeName)
		break
	case config.Secret:
		disk.Source.File = config.GetSecretDiskPath(volumeName)
		break
	case config.ServiceAccount:
		disk.Source.File = config.GetServiceAccountDiskPath()
		break
	default:
		return fmt.Errorf("Cannot convert config '%s' to disk, unrecognized type", configType)
	}

	return nil
}

func GetFilesystemVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "var", "run", "kubevirt-private", "vmi-disks", volumeName, "disk.img")
}

func GetBlockDeviceVolumePath(volumeName string) string {
	return filepath.Join(string(filepath.Separator), "dev", volumeName)
}

func Convert_v1_PersistentVolumeClaim_To_api_Disk(name string, disk *Disk, c *ConverterContext) error {
	if c.IsBlockPVC[name] {
		return Convert_v1_BlockVolumeSource_To_api_Disk(name, disk, c)
	}
	return Convert_v1_FilesystemVolumeSource_To_api_Disk(name, disk, c)
}

func Convert_v1_DataVolume_To_api_Disk(name string, disk *Disk, c *ConverterContext) error {
	if c.IsBlockDV[name] {
		return Convert_v1_BlockVolumeSource_To_api_Disk(name, disk, c)
	}
	return Convert_v1_FilesystemVolumeSource_To_api_Disk(name, disk, c)
}

// Convert_v1_FilesystemVolumeSource_To_api_Disk takes a FS source and builds the KVM Disk representation
func Convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *Disk, c *ConverterContext) error {
	disk.Type = "file"
	disk.Driver.Type = "raw"
	disk.Source.File = GetFilesystemVolumePath(volumeName)
	return nil
}

func Convert_v1_BlockVolumeSource_To_api_Disk(volumeName string, disk *Disk, c *ConverterContext) error {
	disk.Type = "block"
	disk.Driver.Type = "raw"
	disk.Source.Dev = GetBlockDeviceVolumePath(volumeName)
	return nil
}

func Convert_v1_HostDisk_To_api_Disk(volumeName string, path string, disk *Disk, c *ConverterContext) error {
	disk.Type = "file"
	disk.Driver.Type = "raw"
	disk.Source.File = hostdisk.GetMountedHostDiskPath(volumeName, path)
	return nil
}

func Convert_v1_CloudInitSource_To_api_Disk(source v1.VolumeSource, disk *Disk, c *ConverterContext) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.Name)
	}

	var dataSource cloudinit.DataSourceType
	if source.CloudInitNoCloud != nil {
		dataSource = cloudinit.DataSourceNoCloud
	} else if source.CloudInitConfigDrive != nil {
		dataSource = cloudinit.DataSourceConfigDrive
	} else {
		return fmt.Errorf("Only nocloud and configdrive are valid cloud-init volumes")
	}

	disk.Source.File = cloudinit.GetIsoFilePath(dataSource, c.VirtualMachine.Name, c.VirtualMachine.Namespace)
	disk.Type = "file"
	disk.Driver.Type = "raw"
	return nil
}

func Convert_v1_IgnitionData_To_api_Disk(disk *Disk, c *ConverterContext) error {
	disk.Source.File = fmt.Sprintf("%s/%s", ignition.GetDomainBasePath(c.VirtualMachine.Name, c.VirtualMachine.Namespace), c.VirtualMachine.Annotations[v1.IgnitionAnnotation])
	disk.Type = "file"
	disk.Driver.Type = "raw"
	return nil
}

func Convert_v1_EmptyDiskSource_To_api_Disk(volumeName string, _ *v1.EmptyDiskSource, disk *Disk, c *ConverterContext) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.Name)
	}

	disk.Type = "file"
	disk.Driver.Type = "qcow2"
	disk.Source.File = emptydisk.FilePathForVolumeName(volumeName)

	return nil
}

func Convert_v1_ContainerDiskSource_To_api_Disk(volumeName string, _ *v1.ContainerDiskSource, disk *Disk, c *ConverterContext, diskIndex int) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.Name)
	}
	disk.Type = "file"
	disk.Driver.Type = "qcow2"
	disk.Source.File = ephemeraldisk.GetFilePath(volumeName)
	disk.BackingStore = &BackingStore{
		Format: &BackingStoreFormat{},
		Source: &DiskSource{},
	}

	source := containerdisk.GetDiskTargetPathFromLauncherView(diskIndex)

	disk.BackingStore.Format.Type = c.DiskType[volumeName].Format
	disk.BackingStore.Source.File = source
	disk.BackingStore.Type = "file"

	return nil
}

func Convert_v1_EphemeralVolumeSource_To_api_Disk(volumeName string, source *v1.EphemeralVolumeSource, disk *Disk, c *ConverterContext) error {
	disk.Type = "file"
	disk.Driver.Type = "qcow2"
	disk.Source.File = ephemeraldisk.GetFilePath(volumeName)
	disk.BackingStore = &BackingStore{
		Format: &BackingStoreFormat{},
		Source: &DiskSource{},
	}

	backingDisk := &Disk{Driver: &DiskDriver{}}
	err := Convert_v1_FilesystemVolumeSource_To_api_Disk(volumeName, backingDisk, c)
	if err != nil {
		return err
	}

	disk.BackingStore.Format.Type = backingDisk.Driver.Type
	disk.BackingStore.Source = &backingDisk.Source
	disk.BackingStore.Type = backingDisk.Type

	return nil
}

func Convert_v1_Watchdog_To_api_Watchdog(source *v1.Watchdog, watchdog *Watchdog, _ *ConverterContext) error {
	watchdog.Alias = &Alias{
		Name: source.Name,
	}
	if source.I6300ESB != nil {
		watchdog.Model = "i6300esb"
		watchdog.Action = string(source.I6300ESB.Action)
		return nil
	}
	return fmt.Errorf("watchdog %s can't be mapped, no watchdog type specified", source.Name)
}

func Convert_v1_Rng_To_api_Rng(source *v1.Rng, rng *Rng, _ *ConverterContext) error {

	// default rng model for KVM/QEMU virtualization
	rng.Model = "virtio"

	// default backend model, random
	rng.Backend = &RngBackend{
		Model: "random",
	}

	// the default source for rng is dev urandom
	rng.Backend.Source = "/dev/urandom"

	return nil
}

func Convert_v1_Input_To_api_InputDevice(input *v1.Input, inputDevice *Input, _ *ConverterContext) error {
	if input.Bus != "virtio" && input.Bus != "usb" && input.Bus != "" {
		return fmt.Errorf("input contains unsupported bus %s", input.Bus)
	}

	if input.Bus != "virtio" && input.Bus != "usb" {
		input.Bus = "usb"
	}

	if input.Type != "tablet" {
		return fmt.Errorf("input contains unsupported type %s", input.Type)
	}

	inputDevice.Bus = input.Bus
	inputDevice.Type = input.Type
	inputDevice.Alias = &Alias{Name: input.Name}
	return nil
}

func Convert_v1_Clock_To_api_Clock(source *v1.Clock, clock *Clock, c *ConverterContext) error {
	if source.UTC != nil {
		clock.Offset = "utc"
		if source.UTC.OffsetSeconds != nil {
			clock.Adjustment = strconv.Itoa(*source.UTC.OffsetSeconds)
		} else {
			clock.Adjustment = "reset"
		}
	} else if source.Timezone != nil {
		clock.Offset = "timezone"
		clock.Timezone = string(*source.Timezone)
	}

	if source.Timer != nil {
		if source.Timer.RTC != nil {
			newTimer := Timer{Name: "rtc"}
			newTimer.Track = string(source.Timer.RTC.Track)
			newTimer.TickPolicy = string(source.Timer.RTC.TickPolicy)
			newTimer.Present = boolToYesNo(source.Timer.RTC.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.PIT != nil {
			newTimer := Timer{Name: "pit"}
			newTimer.Present = boolToYesNo(source.Timer.PIT.Enabled, true)
			newTimer.TickPolicy = string(source.Timer.PIT.TickPolicy)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.KVM != nil {
			newTimer := Timer{Name: "kvmclock"}
			newTimer.Present = boolToYesNo(source.Timer.KVM.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.HPET != nil {
			newTimer := Timer{Name: "hpet"}
			newTimer.Present = boolToYesNo(source.Timer.HPET.Enabled, true)
			newTimer.TickPolicy = string(source.Timer.HPET.TickPolicy)
			clock.Timer = append(clock.Timer, newTimer)
		}
		if source.Timer.Hyperv != nil {
			newTimer := Timer{Name: "hypervclock"}
			newTimer.Present = boolToYesNo(source.Timer.Hyperv.Enabled, true)
			clock.Timer = append(clock.Timer, newTimer)
		}
	}

	return nil
}

func convertFeatureState(source *v1.FeatureState) *FeatureState {
	if source != nil {
		return &FeatureState{
			State: boolToOnOff(source.Enabled, true),
		}
	}
	return nil
}

func Convert_v1_Features_To_api_Features(source *v1.Features, features *Features, c *ConverterContext) error {
	if source.ACPI.Enabled == nil || *source.ACPI.Enabled {
		features.ACPI = &FeatureEnabled{}
	}
	if source.SMM != nil {
		if source.SMM.Enabled == nil || *source.SMM.Enabled {
			features.SMM = &FeatureEnabled{}
		}
	}
	if source.APIC != nil {
		if source.APIC.Enabled == nil || *source.APIC.Enabled {
			features.APIC = &FeatureEnabled{}
		}
	}
	if source.Hyperv != nil {
		features.Hyperv = &FeatureHyperv{}
		err := Convert_v1_FeatureHyperv_To_api_FeatureHyperv(source.Hyperv, features.Hyperv, c)
		if err != nil {
			return nil
		}
	}
	if source.KVM != nil {
		features.KVM = &FeatureKVM{
			Hidden: &FeatureState{
				State: boolToOnOff(&source.KVM.Hidden, false),
			},
		}
	}
	return nil
}

func Convert_v1_Machine_To_api_OSType(source *v1.Machine, ost *OSType, c *ConverterContext) error {
	ost.Machine = source.Type

	return nil
}

func Convert_v1_FeatureHyperv_To_api_FeatureHyperv(source *v1.FeatureHyperv, hyperv *FeatureHyperv, c *ConverterContext) error {
	if source.Spinlocks != nil {
		hyperv.Spinlocks = &FeatureSpinlocks{
			State:   boolToOnOff(source.Spinlocks.Enabled, true),
			Retries: source.Spinlocks.Retries,
		}
	}
	if source.VendorID != nil {
		hyperv.VendorID = &FeatureVendorID{
			State: boolToOnOff(source.VendorID.Enabled, true),
			Value: source.VendorID.VendorID,
		}
	}
	hyperv.Relaxed = convertFeatureState(source.Relaxed)
	hyperv.Reset = convertFeatureState(source.Reset)
	hyperv.Runtime = convertFeatureState(source.Runtime)
	hyperv.SyNIC = convertFeatureState(source.SyNIC)
	hyperv.SyNICTimer = convertFeatureState(source.SyNICTimer)
	hyperv.VAPIC = convertFeatureState(source.VAPIC)
	hyperv.VPIndex = convertFeatureState(source.VPIndex)
	hyperv.Frequencies = convertFeatureState(source.Frequencies)
	hyperv.Reenlightenment = convertFeatureState(source.Reenlightenment)
	hyperv.TLBFlush = convertFeatureState(source.TLBFlush)
	hyperv.IPI = convertFeatureState(source.IPI)
	hyperv.EVMCS = convertFeatureState(source.EVMCS)
	return nil
}

func ConvertV1ToAPIBalloning(source *v1.Devices, ballooning *MemBalloon, c *ConverterContext) {
	if source != nil && source.AutoattachMemBalloon != nil && *source.AutoattachMemBalloon == false {
		ballooning.Model = "none"
		ballooning.Stats = nil
	} else {
		ballooning.Model = "virtio"
		if c.MemBalloonStatsPeriod != 0 {
			ballooning.Stats = &Stats{Period: c.MemBalloonStatsPeriod}
		}

	}
}

func filterAddress(addrs []string, addr string) []string {
	var res []string
	for _, a := range addrs {
		if a != addr {
			res = append(res, a)
		}
	}
	return res
}

func reserveAddress(addrsMap map[string][]string, addr string) {
	// Sometimes the same address is available to multiple networks,
	// specifically when two networks refer to the same resourceName. In this
	// case, we should make sure that a reserved address is removed from *all*
	// per-network lists of available devices, to avoid configuring the same
	// device ID for multiple interfaces.
	for networkName, addrs := range addrsMap {
		addrsMap[networkName] = filterAddress(addrs, addr)
	}
	return
}

// Get the next PCI address available to a particular SR-IOV network. The
// function makes sure that the allocated address is not allocated to next
// callers, whether they request an address for the same network or another
// network that is backed by the same resourceName.
func popSRIOVPCIAddress(networkName string, addrsMap map[string][]string) (string, map[string][]string, error) {
	if len(addrsMap[networkName]) > 0 {
		addr := addrsMap[networkName][0]
		reserveAddress(addrsMap, addr)
		return addr, addrsMap, nil
	}
	return "", addrsMap, fmt.Errorf("no more SR-IOV PCI addresses to allocate")
}

func getInterfaceType(iface *v1.Interface) string {
	if iface.Slirp != nil {
		// Slirp configuration works only with e1000 or rtl8139
		if iface.Model != "e1000" && iface.Model != "rtl8139" {
			log.Log.Infof("The network interface type of %s was changed to e1000 due to unsupported interface type by qemu slirp network", iface.Name)
			return "e1000"
		}
		return iface.Model
	}
	if iface.Model != "" {
		return iface.Model
	}
	return "virtio"
}

func Convert_v1_VirtualMachine_To_api_Domain(vmi *v1.VirtualMachineInstance, domain *Domain, c *ConverterContext) (err error) {
	precond.MustNotBeNil(vmi)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	domain.Spec.Name = VMINamespaceKeyFunc(vmi)
	domain.ObjectMeta.Name = vmi.ObjectMeta.Name
	domain.ObjectMeta.Namespace = vmi.ObjectMeta.Namespace

	// Set VM CPU cores
	// CPU topology will be created everytime, because user can specify
	// number of cores in vmi.Spec.Domain.Resources.Requests/Limits, not only
	// in vmi.Spec.Domain.CPU
	domain.Spec.CPU.Topology = getCPUTopology(vmi)
	domain.Spec.VCPU = &VCPU{
		Placement: "static",
		CPUs:      calculateRequestedVCPUs(domain.Spec.CPU.Topology),
	}

	if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
		if c.UseEmulation {
			logger := log.DefaultLogger()
			logger.Infof("Hardware emulation device '/dev/kvm' not present. Using software emulation.")
			domain.Spec.Type = "qemu"
		} else {
			return fmt.Errorf("hardware emulation device '/dev/kvm' not present")
		}
	} else if err != nil {
		return err
	}

	virtioNetProhibited := false
	if _, err := os.Stat("/dev/vhost-net"); os.IsNotExist(err) {
		if c.UseEmulation {
			logger := log.DefaultLogger()
			logger.Infof("In-kernel virtio-net device emulation '/dev/vhost-net' not present. Falling back to QEMU userland emulation.")
		} else {
			virtioNetProhibited = true
		}
	} else if err != nil {
		return err
	}

	// Spec metadata

	newChannel := Add_Agent_To_api_Channel()
	domain.Spec.Devices.Channels = append(domain.Spec.Devices.Channels, newChannel)

	domain.Spec.Metadata.KubeVirt.UID = vmi.UID
	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}
	domain.Spec.Metadata.KubeVirt.GracePeriod = &GracePeriodMetadata{
		DeletionGracePeriodSeconds: gracePeriodSeconds,
	}

	domain.Spec.SysInfo = &SysInfo{}
	if vmi.Spec.Domain.Firmware != nil {
		domain.Spec.SysInfo.System = []Entry{
			{
				Name:  "uuid",
				Value: string(vmi.Spec.Domain.Firmware.UUID),
			},
		}

		if vmi.Spec.Domain.Firmware.Bootloader != nil && vmi.Spec.Domain.Firmware.Bootloader.EFI != nil {
			if vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot == nil || *vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot {
				domain.Spec.OS.BootLoader = &Loader{
					Path:     filepath.Join(c.OVMFPath, EFICodeSecureBoot),
					ReadOnly: "yes",
					Secure:   "yes",
					Type:     "pflash",
				}

				domain.Spec.OS.NVRam = &NVRam{
					NVRam:    filepath.Join("/tmp", domain.Spec.Name),
					Template: filepath.Join(c.OVMFPath, EFIVarsSecureBoot),
				}
			} else {
				domain.Spec.OS.BootLoader = &Loader{
					Path:     filepath.Join(c.OVMFPath, EFICode),
					ReadOnly: "yes",
					Secure:   "no",
					Type:     "pflash",
				}

				domain.Spec.OS.NVRam = &NVRam{
					NVRam:    filepath.Join("/tmp", domain.Spec.Name),
					Template: filepath.Join(c.OVMFPath, EFIVars),
				}
			}
		}

		if len(vmi.Spec.Domain.Firmware.Serial) > 0 {
			domain.Spec.SysInfo.System = append(domain.Spec.SysInfo.System, Entry{Name: "serial", Value: string(vmi.Spec.Domain.Firmware.Serial)})
		}
	}
	if c.SMBios != nil {
		domain.Spec.SysInfo.System = append(domain.Spec.SysInfo.System,
			Entry{
				Name:  "manufacturer",
				Value: c.SMBios.Manufacturer,
			},
			Entry{
				Name:  "family",
				Value: c.SMBios.Family,
			},
			Entry{
				Name:  "product",
				Value: c.SMBios.Product,
			},
			Entry{
				Name:  "sku",
				Value: c.SMBios.Sku,
			},
			Entry{
				Name:  "version",
				Value: c.SMBios.Version,
			},
		)
	}

	// Take SMBios values from the VirtualMachineOptions
	// SMBios option does not work in Power, attempting to set it will result in the following error message:
	// "Option not supported for this target" issued by qemu-system-ppc64, so don't set it in case GOARCH is ppc64le
	if c.Architecture != "ppc64le" {
		domain.Spec.OS.SMBios = &SMBios{
			Mode: "sysinfo",
		}
	}

	if vmi.Spec.Domain.Chassis != nil {
		domain.Spec.SysInfo.Chassis = []Entry{
			{
				Name:  "manufacturer",
				Value: string(vmi.Spec.Domain.Chassis.Manufacturer),
			},
			{
				Name:  "version",
				Value: string(vmi.Spec.Domain.Chassis.Version),
			},
			{
				Name:  "serial",
				Value: string(vmi.Spec.Domain.Chassis.Serial),
			},
			{
				Name:  "asset",
				Value: string(vmi.Spec.Domain.Chassis.Asset),
			},
			{
				Name:  "sku",
				Value: string(vmi.Spec.Domain.Chassis.Sku),
			},
		}
	}

	if domain.Spec.Memory, err = QuantityToByte(*getVirtualMemory(vmi)); err != nil {
		return err
	}

	var isMemfdRequired = false
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		domain.Spec.MemoryBacking = &MemoryBacking{
			HugePages: &HugePages{},
		}
		if val := vmi.Annotations[v1.MemfdMemoryBackend]; val != "false" {
			isMemfdRequired = true
		}
	}
	// virtiofs require shared access
	if util.IsVMIVirtiofsEnabled(vmi) {
		if domain.Spec.MemoryBacking == nil {
			domain.Spec.MemoryBacking = &MemoryBacking{}
		}
		domain.Spec.MemoryBacking.Access = &MemoryBackingAccess{
			Mode: "shared",
		}
		isMemfdRequired = true
	}

	if isMemfdRequired {
		// Set memfd as memory backend to solve SELinux restrictions
		// See the issue: https://github.com/kubevirt/kubevirt/issues/3781
		domain.Spec.MemoryBacking.Source = &MemoryBackingSource{Type: "memfd"}
		// NUMA is required in order to use memfd
		domain.Spec.CPU.NUMA = &NUMA{
			Cells: []NUMACell{
				{
					ID:     "0",
					CPUs:   fmt.Sprintf("0-%d", domain.Spec.VCPU.CPUs-1),
					Memory: fmt.Sprintf("%d", getVirtualMemory(vmi).Value()/int64(1024)),
					Unit:   "KiB",
				},
			},
		}
	}

	volumeIndices := map[string]int{}
	volumes := map[string]*v1.Volume{}
	for i, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
		volumeIndices[volume.Name] = i
	}

	dedicatedThreads := 0
	autoThreads := 0
	useIOThreads := false
	threadPoolLimit := 1

	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		useIOThreads = true

		if (*vmi.Spec.Domain.IOThreadsPolicy) == v1.IOThreadsPolicyAuto {
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
		}
	}
	for _, diskDevice := range vmi.Spec.Domain.Devices.Disks {
		dedicatedThread := false
		if diskDevice.DedicatedIOThread != nil {
			dedicatedThread = *diskDevice.DedicatedIOThread
		}
		if dedicatedThread {
			useIOThreads = true
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

	ioThreadCount := (autoThreads + dedicatedThreads)
	if ioThreadCount != 0 {
		if domain.Spec.IOThreads == nil {
			domain.Spec.IOThreads = &IOThreads{}
		}
		domain.Spec.IOThreads.IOThreads = uint(ioThreadCount)
	}

	currentAutoThread := defaultIOThread
	currentDedicatedThread := uint(autoThreads + 1)

	var numQueues *uint
	var numBlkQueues *uint
	virtioBlkMQRequested := (vmi.Spec.Domain.Devices.BlockMultiQueue != nil) && (*vmi.Spec.Domain.Devices.BlockMultiQueue)
	virtioNetMQRequested := (vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue != nil) && (*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue)
	vcpus := uint(calculateRequestedVCPUs(domain.Spec.CPU.Topology))
	if vcpus == 0 {
		vcpus = uint(1)
	}
	if virtioNetMQRequested {
		numQueues = &vcpus
	}
	if virtioBlkMQRequested {
		numBlkQueues = &vcpus
	}

	devicePerBus := make(map[string]int)
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := Disk{}

		err := Convert_v1_Disk_To_api_Disk(&disk, &newDisk, devicePerBus, numBlkQueues)
		if err != nil {
			return err
		}
		volume := volumes[disk.Name]
		if volume == nil {
			return fmt.Errorf("No matching volume with name %s found", disk.Name)
		}
		err = Convert_v1_Volume_To_api_Disk(volume, &newDisk, c, volumeIndices[disk.Name])
		if err != nil {
			return err
		}

		if useIOThreads {
			ioThreadId := defaultIOThread
			dedicatedThread := false
			if disk.DedicatedIOThread != nil {
				dedicatedThread = *disk.DedicatedIOThread
			}

			if dedicatedThread {
				ioThreadId = currentDedicatedThread
				currentDedicatedThread += 1
			} else {
				ioThreadId = currentAutoThread
				// increment the threadId to be used next but wrap around at the thread limit
				// the odd math here is because thread ID's start at 1, not 0
				currentAutoThread = (currentAutoThread % uint(autoThreads)) + 1
			}
			newDisk.Driver.IOThread = &ioThreadId
		}

		domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
	}
	// Handle virtioFS
	for _, fs := range vmi.Spec.Domain.Devices.Filesystems {
		if fs.Virtiofs != nil {
			newFS := FilesystemDevice{}

			newFS.Type = "mount"
			newFS.AccessMode = "passthrough"
			newFS.Driver = &FilesystemDriver{
				Type:  "virtiofs",
				Queue: "1024",
			}
			newFS.Binary = &FilesystemBinary{
				Path:  "/usr/libexec/virtiofsd",
				Xattr: "on",
				Cache: &FilesystemBinaryCache{
					Mode: "always",
				},
				Lock: &FilesystemBinaryLock{
					Posix: "on",
					Flock: "on",
				},
			}
			newFS.Target = &FilesystemTarget{
				Dir: fs.Name,
			}

			volume := volumes[fs.Name]
			if volume == nil {
				return fmt.Errorf("No matching volume with name %s found", fs.Name)
			}
			volDir, _ := filepath.Split(GetFilesystemVolumePath(volume.Name))
			newFS.Source = &FilesystemSource{}
			newFS.Source.Dir = volDir
			domain.Spec.Devices.Filesystems = append(domain.Spec.Devices.Filesystems, newFS)
		}
	}

	if vmi.Spec.Domain.Devices.Watchdog != nil {
		newWatchdog := &Watchdog{}
		err := Convert_v1_Watchdog_To_api_Watchdog(vmi.Spec.Domain.Devices.Watchdog, newWatchdog, c)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Watchdog = newWatchdog
	}

	if vmi.Spec.Domain.Devices.Rng != nil {
		newRng := &Rng{}
		err := Convert_v1_Rng_To_api_Rng(vmi.Spec.Domain.Devices.Rng, newRng, c)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Rng = newRng
	}

	isUSBDevicePresent := false
	if vmi.Spec.Domain.Devices.Inputs != nil {
		inputDevices := make([]Input, 0)
		for _, input := range vmi.Spec.Domain.Devices.Inputs {
			inputDevice := Input{}
			err := Convert_v1_Input_To_api_InputDevice(&input, &inputDevice, c)
			if err != nil {
				return err
			}
			inputDevices = append(inputDevices, inputDevice)
			if inputDevice.Bus == "usb" {
				isUSBDevicePresent = true
			}
		}
		domain.Spec.Devices.Inputs = inputDevices
	}

	domain.Spec.Devices.Ballooning = &MemBalloon{}
	ConvertV1ToAPIBalloning(&vmi.Spec.Domain.Devices, domain.Spec.Devices.Ballooning, c)

	//usb controller is turned on, only when user specify input device with usb bus,
	//otherwise it is turned off
	//In ppc64le usb devices like mouse / keyboard are set by default,
	//so we can't disable the controller otherwise we run into the following error:
	//"unsupported configuration: USB is disabled for this domain, but USB devices are present in the domain XML"
	if !isUSBDevicePresent && c.Architecture != "ppc64le" {
		// disable usb controller
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, Controller{
			Type:  "usb",
			Index: "0",
			Model: "none",
		})
	} else {
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, Controller{
			Type:  "usb",
			Index: "0",
			Model: "qemu-xhci",
		})
	}

	if vmi.Spec.Domain.Clock != nil {
		clock := vmi.Spec.Domain.Clock
		newClock := &Clock{}
		err := Convert_v1_Clock_To_api_Clock(clock, newClock, c)
		if err != nil {
			return err
		}
		domain.Spec.Clock = newClock
	}

	if vmi.Spec.Domain.Features != nil {
		domain.Spec.Features = &Features{}
		err := Convert_v1_Features_To_api_Features(vmi.Spec.Domain.Features, domain.Spec.Features, c)
		if err != nil {
			return err
		}
	}
	apiOst := &vmi.Spec.Domain.Machine
	err = Convert_v1_Machine_To_api_OSType(apiOst, &domain.Spec.OS.Type, c)
	if err != nil {
		return err
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
		if vmi.Spec.Domain.CPU.Features != nil {
			for _, feature := range vmi.Spec.Domain.CPU.Features {
				domain.Spec.CPU.Features = append(domain.Spec.CPU.Features, CPUFeature{
					Name:   feature.Name,
					Policy: feature.Policy,
				})
			}
		}

		// Adjust guest vcpu config. Currenty will handle vCPUs to pCPUs pinning
		if vmi.IsCPUDedicated() {
			if err := formatDomainCPUTune(vmi, domain, c); err != nil {
				log.Log.Reason(err).Error("failed to format domain cputune.")
				return err
			}
			if vmi.Spec.Domain.CPU.IsolateEmulatorThread {
				if c.EmulatorThreadCpu == nil {
					err := fmt.Errorf("no CPUs allocated for the emulation thread")
					log.Log.Reason(err).Error("failed to format emulation thread pin")
					return err

				}
				appendDomainEmulatorThreadPin(domain, *c.EmulatorThreadCpu)
			}
			if useIOThreads {
				if err := formatDomainIOThreadPin(vmi, domain, c); err != nil {
					log.Log.Reason(err).Error("failed to format domain iothread pinning.")
					return err
				}

			}
		}
	}

	// Append HostDevices to DomXML if GPU is requested
	if util.IsGPUVMI(vmi) {
		vgpuMdevUUID := append([]string{}, c.VgpuDevices...)
		hostDevices, err := createHostDevicesFromMdevUUIDList(vgpuMdevUUID)
		if err != nil {
			log.Log.Reason(err).Error("Unable to parse Mdev UUID addresses")
		} else {
			domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, hostDevices...)
		}
		gpuPCIAddresses := append([]string{}, c.GpuDevices...)
		hostDevices, err = createHostDevicesFromPCIAddresses(gpuPCIAddresses)
		if err != nil {
			log.Log.Reason(err).Error("Unable to parse PCI addresses")
		} else {
			domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, hostDevices...)
		}
	}

	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		domain.Spec.CPU.Mode = v1.CPUModeHostModel
	}

	if vmi.Spec.Domain.Devices.AutoattachSerialConsole == nil || *vmi.Spec.Domain.Devices.AutoattachSerialConsole == true {
		// Add mandatory console device
		domain.Spec.Devices.Controllers = append(domain.Spec.Devices.Controllers, Controller{
			Type:  "virtio-serial",
			Index: "0",
		})

		var serialPort uint = 0
		var serialType string = "serial"
		domain.Spec.Devices.Consoles = []Console{
			{
				Type: "pty",
				Target: &ConsoleTarget{
					Type: &serialType,
					Port: &serialPort,
				},
			},
		}

		domain.Spec.Devices.Serials = []Serial{
			{
				Type: "unix",
				Target: &SerialTarget{
					Port: &serialPort,
				},
				Source: &SerialSource{
					Mode: "bind",
					Path: fmt.Sprintf("/var/run/kubevirt-private/%s/virt-serial%d", vmi.ObjectMeta.UID, serialPort),
				},
			},
		}
	}

	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == nil || *vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == true {
		var heads uint = 1
		var vram uint = 16384
		domain.Spec.Devices.Video = []Video{
			{
				Model: VideoModel{
					Type:  "vga",
					Heads: &heads,
					VRam:  &vram,
				},
			},
		}
		domain.Spec.Devices.Graphics = []Graphics{
			{
				Listen: &GraphicsListen{
					Type:   "socket",
					Socket: fmt.Sprintf("/var/run/kubevirt-private/%s/virt-vnc", vmi.ObjectMeta.UID),
				},
				Type: "vnc",
			},
		}
	}

	networks := map[string]*v1.Network{}
	cniNetworks := map[string]int{}
	multusNetworkIndex := 1
	for _, network := range vmi.Spec.Networks {
		numberOfSources := 0
		if network.Pod != nil {
			numberOfSources++
		}
		if network.Multus != nil {
			if network.Multus.Default {
				// default network is eth0
				cniNetworks[network.Name] = 0
			} else {
				cniNetworks[network.Name] = multusNetworkIndex
				multusNetworkIndex++
			}
			numberOfSources++
		}
		if numberOfSources == 0 {
			return fmt.Errorf("fail network %s must have a network type", network.Name)
		} else if numberOfSources > 1 {
			return fmt.Errorf("fail network %s must have only one network type", network.Name)
		}
		networks[network.Name] = network.DeepCopy()
	}

	sriovPciAddresses := make(map[string][]string)
	for key, value := range c.SRIOVDevices {
		sriovPciAddresses[key] = append([]string{}, value...)
	}

	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		net, isExist := networks[iface.Name]
		if !isExist {
			return fmt.Errorf("failed to find network %s", iface.Name)
		}

		if iface.SRIOV != nil {
			var pciAddr string
			pciAddr, sriovPciAddresses, err = popSRIOVPCIAddress(iface.Name, sriovPciAddresses)
			if err != nil {
				return err
			}

			dbsfFields, err := util.ParsePciAddress(pciAddr)
			if err != nil {
				return err
			}

			hostDev := HostDevice{
				Source: HostDeviceSource{
					Address: &Address{
						Type:     "pci",
						Domain:   "0x" + dbsfFields[0],
						Bus:      "0x" + dbsfFields[1],
						Slot:     "0x" + dbsfFields[2],
						Function: "0x" + dbsfFields[3],
					},
				},
				Type:    "pci",
				Managed: "yes",
			}
			if iface.BootOrder != nil {
				hostDev.BootOrder = &BootOrder{Order: *iface.BootOrder}
			}
			log.Log.Infof("SR-IOV PCI device allocated: %s", pciAddr)
			domain.Spec.Devices.HostDevices = append(domain.Spec.Devices.HostDevices, hostDev)
		} else {
			ifaceType := getInterfaceType(&iface)
			domainIface := Interface{
				Model: &Model{
					Type: ifaceType,
				},
				Alias: &Alias{
					Name: iface.Name,
				},
			}

			// if UseEmulation unset and at least one NIC model is virtio,
			// /dev/vhost-net must be present as we should have asked for it.
			if ifaceType == "virtio" && virtioNetProhibited {
				return fmt.Errorf("In-kernel virtio-net device emulation '/dev/vhost-net' not present")
			} else if ifaceType == "virtio" && virtioNetMQRequested {
				domainIface.Driver = &InterfaceDriver{Name: "vhost", Queues: numQueues}
			}

			// Add a pciAddress if specifed
			if iface.PciAddress != "" {
				addr, err := decoratePciAddressField(iface.PciAddress)
				if err != nil {
					return fmt.Errorf("failed to configure interface %s: %v", iface.Name, err)
				}
				domainIface.Address = addr
			}

			if iface.Bridge != nil || iface.Masquerade != nil {
				// TODO:(ihar) consider abstracting interface type conversion /
				// detection into drivers

				// use "ethernet" interface type, since we're using pre-configured tap devices
				// https://libvirt.org/formatdomain.html#elementsNICSEthernet
				domainIface.Type = "ethernet"
				if iface.BootOrder != nil {
					domainIface.BootOrder = &BootOrder{Order: *iface.BootOrder}
				}
			} else if iface.Slirp != nil {
				domainIface.Type = "user"

				// Create network interface
				if domain.Spec.QEMUCmd == nil {
					domain.Spec.QEMUCmd = &Commandline{}
				}

				if domain.Spec.QEMUCmd.QEMUArg == nil {
					domain.Spec.QEMUCmd.QEMUArg = make([]Arg, 0)
				}

				// TODO: (seba) Need to change this if multiple interface can be connected to the same network
				// append the ports from all the interfaces connected to the same network
				err := createSlirpNetwork(iface, *net, domain)
				if err != nil {
					return err
				}
			}
			domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, domainIface)
		}
	}

	// Add Ignition Command Line if present
	ignitiondata, _ := vmi.Annotations[v1.IgnitionAnnotation]
	if ignitiondata != "" && strings.Contains(ignitiondata, "ignition") {
		if domain.Spec.QEMUCmd == nil {
			domain.Spec.QEMUCmd = &Commandline{}
		}

		if domain.Spec.QEMUCmd.QEMUArg == nil {
			domain.Spec.QEMUCmd.QEMUArg = make([]Arg, 0)
		}
		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, Arg{Value: "-fw_cfg"})
		ignitionpath := fmt.Sprintf("%s/%s", ignition.GetDomainBasePath(c.VirtualMachine.Name, c.VirtualMachine.Namespace), ignition.IgnitionFile)
		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, Arg{Value: fmt.Sprintf("name=opt/com.coreos/config,file=%s", ignitionpath)})
	}

	if val := vmi.Annotations[v1.PlacePCIDevicesOnRootComplex]; val == "true" {
		if err := PlacePCIDevicesOnRootComplex(&domain.Spec); err != nil {
			return err
		}
	}

	return nil
}

func getVirtualMemory(vmi *v1.VirtualMachineInstance) *resource.Quantity {
	// In case that guest memory is explicitly set, return it
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		return vmi.Spec.Domain.Memory.Guest
	}

	// Otherwise, take memory from the memory-limit, if set
	if v, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory]; ok {
		return &v
	}

	// Otherwise, take memory from the requested memory
	v, _ := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
	return &v
}

func getCPUTopology(vmi *v1.VirtualMachineInstance) *CPUTopology {
	cores := uint32(1)
	threads := uint32(1)
	sockets := uint32(1)
	vmiCPU := vmi.Spec.Domain.CPU
	if vmiCPU != nil {
		if vmiCPU.Cores != 0 {
			cores = vmiCPU.Cores
		}

		if vmiCPU.Threads != 0 {
			threads = vmiCPU.Threads
		}

		if vmiCPU.Sockets != 0 {
			sockets = vmiCPU.Sockets
		}
	}

	if vmiCPU == nil || (vmiCPU.Cores == 0 && vmiCPU.Sockets == 0 && vmiCPU.Threads == 0) {
		//if cores, sockets, threads are not set, take value from domain resources request or limits and
		//set value into sockets, which have best performance (https://bugzilla.redhat.com/show_bug.cgi?id=1653453)
		resources := vmi.Spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuLimit.Value())
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			sockets = uint32(cpuRequests.Value())
		}
	}

	return &CPUTopology{
		Sockets: sockets,
		Cores:   cores,
		Threads: threads,
	}
}

func calculateRequestedVCPUs(cpuTopology *CPUTopology) uint32 {
	return cpuTopology.Cores * cpuTopology.Sockets * cpuTopology.Threads
}

func formatDomainCPUTune(vmi *v1.VirtualMachineInstance, domain *Domain, c *ConverterContext) error {
	if len(c.CPUSet) == 0 {
		return fmt.Errorf("failed for get pods pinned cpus")
	}
	vcpus := calculateRequestedVCPUs(domain.Spec.CPU.Topology)
	cpuTune := CPUTune{}
	for idx := 0; idx < int(vcpus); idx++ {
		vcpupin := CPUTuneVCPUPin{}
		vcpupin.VCPU = uint(idx)
		vcpupin.CPUSet = strconv.Itoa(c.CPUSet[idx])
		cpuTune.VCPUPin = append(cpuTune.VCPUPin, vcpupin)
	}
	domain.Spec.CPUTune = &cpuTune
	return nil
}

func appendDomainEmulatorThreadPin(domain *Domain, allocatedCpu int) {
	emulatorThread := CPUEmulatorPin{
		CPUSet: strconv.Itoa(allocatedCpu),
	}
	domain.Spec.CPUTune.EmulatorPin = &emulatorThread
}

func appendDomainIOThreadPin(domain *Domain, thread uint, cpuset string) {
	iothreadPin := CPUTuneIOThreadPin{}
	iothreadPin.IOThread = thread
	iothreadPin.CPUSet = cpuset
	domain.Spec.CPUTune.IOThreadPin = append(domain.Spec.CPUTune.IOThreadPin, iothreadPin)
}

func formatDomainIOThreadPin(vmi *v1.VirtualMachineInstance, domain *Domain, c *ConverterContext) error {
	iothreads := int(domain.Spec.IOThreads.IOThreads)
	vcpus := int(calculateRequestedVCPUs(domain.Spec.CPU.Topology))

	if vmi.IsCPUDedicated() && vmi.Spec.Domain.CPU.IsolateEmulatorThread {
		// pin the IOThread on the same pCPU as the emulator thread
		cpuset := fmt.Sprintf("%d", *c.EmulatorThreadCpu)
		appendDomainIOThreadPin(domain, uint(1), cpuset)
	} else if iothreads >= vcpus {
		// pin an IOThread on a CPU
		for thread := 1; thread <= iothreads; thread++ {
			cpuset := fmt.Sprintf("%d", c.CPUSet[thread%vcpus])
			appendDomainIOThreadPin(domain, uint(thread), cpuset)
		}
	} else {
		// the following will pin IOThreads to a set of cpus of a balanced size
		// for example, for 3 threads and 8 cpus the output will look like:
		// thread cpus
		//   1    0,1,2
		//   2    3,4,5
		//   3    6,7
		series := vcpus % iothreads
		curr := 0
		for thread := 1; thread <= iothreads; thread++ {
			remainder := vcpus/iothreads - 1
			if thread <= series {
				remainder += 1
			}
			end := curr + remainder
			slice := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(c.CPUSet[curr:end+1])), ","), "[]")
			appendDomainIOThreadPin(domain, uint(thread), slice)
			curr = end + 1
		}
	}
	return nil
}

func createSlirpNetwork(iface v1.Interface, network v1.Network, domain *Domain) error {
	qemuArg := Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}

	err := configVMCIDR(&qemuArg, iface, network)
	if err != nil {
		return err
	}

	err = configDNSSearchName(&qemuArg)
	if err != nil {
		return err
	}

	err = configPortForward(&qemuArg, iface)
	if err != nil {
		return err
	}

	domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, Arg{Value: "-netdev"})
	domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, qemuArg)

	return nil
}

func configPortForward(qemuArg *Arg, iface v1.Interface) error {
	if iface.Ports == nil {
		return nil
	}

	// Can't be duplicated ports forward or the qemu process will crash
	configuredPorts := make(map[string]struct{}, 0)
	for _, forwardPort := range iface.Ports {

		if forwardPort.Port == 0 {
			return fmt.Errorf("Port must be configured")
		}

		if forwardPort.Protocol == "" {
			forwardPort.Protocol = DefaultProtocol
		}

		portConfig := fmt.Sprintf("%s-%d", forwardPort.Protocol, forwardPort.Port)
		if _, ok := configuredPorts[portConfig]; !ok {
			qemuArg.Value += fmt.Sprintf(",hostfwd=%s::%d-:%d", strings.ToLower(forwardPort.Protocol), forwardPort.Port, forwardPort.Port)
			configuredPorts[portConfig] = struct{}{}
		}
	}

	return nil
}

func configVMCIDR(qemuArg *Arg, iface v1.Interface, network v1.Network) error {
	vmNetworkCIDR := ""
	if network.Pod.VMNetworkCIDR != "" {
		_, _, err := net.ParseCIDR(network.Pod.VMNetworkCIDR)
		if err != nil {
			return fmt.Errorf("Failed parsing CIDR %s", network.Pod.VMNetworkCIDR)
		}
		vmNetworkCIDR = network.Pod.VMNetworkCIDR
	} else {
		vmNetworkCIDR = DefaultVMCIDR
	}

	// Insert configuration to qemu commandline
	qemuArg.Value += fmt.Sprintf(",net=%s", vmNetworkCIDR)

	return nil
}

func configDNSSearchName(qemuArg *Arg) error {
	_, dnsDoms, err := GetResolvConfDetailsFromPod()
	if err != nil {
		return err
	}

	for _, dom := range dnsDoms {
		qemuArg.Value += fmt.Sprintf(",dnssearch=%s", dom)
	}
	return nil
}

func SecretToLibvirtSecret(vmi *v1.VirtualMachineInstance, secretName string) string {
	return fmt.Sprintf("%s-%s-%s---", secretName, vmi.Namespace, vmi.Name)
}

func QuantityToByte(quantity resource.Quantity) (Memory, error) {
	memorySize, _ := quantity.AsInt64()
	if memorySize < 0 {
		return Memory{Unit: "b"}, fmt.Errorf("Memory size '%s' must be greater than or equal to 0", quantity.String())
	}
	return Memory{
		Value: uint64(memorySize),
		Unit:  "b",
	}, nil
}

func QuantityToMebiByte(quantity resource.Quantity) (uint64, error) {
	q := int64(float64(0.953674) * float64(quantity.ScaledValue(resource.Mega)))
	if q < 0 {
		return 0, fmt.Errorf("Quantity '%s' must be greate tan or equal to 0", quantity.String())
	}
	return uint64(q), nil
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

// returns nameservers [][]byte, searchdomains []string, error
func GetResolvConfDetailsFromPod() ([][]byte, []string, error) {
	b, err := ioutil.ReadFile(resolvConf)
	if err != nil {
		return nil, nil, err
	}

	nameservers, err := dns.ParseNameservers(string(b))
	if err != nil {
		return nil, nil, err
	}

	searchDomains, err := dns.ParseSearchDomains(string(b))
	if err != nil {
		return nil, nil, err
	}

	log.Log.Reason(err).Infof("Found nameservers in %s: %s", resolvConf, bytes.Join(nameservers, []byte{' '}))
	log.Log.Reason(err).Infof("Found search domains in %s: %s", resolvConf, strings.Join(searchDomains, " "))

	return nameservers, searchDomains, err
}

func decoratePciAddressField(addressField string) (*Address, error) {
	dbsfFields, err := util.ParsePciAddress(addressField)
	if err != nil {
		return nil, err
	}
	decoratedAddrField := &Address{
		Type:     "pci",
		Domain:   "0x" + dbsfFields[0],
		Bus:      "0x" + dbsfFields[1],
		Slot:     "0x" + dbsfFields[2],
		Function: "0x" + dbsfFields[3],
	}
	return decoratedAddrField, nil
}

func createHostDevicesFromPCIAddresses(pcis []string) ([]HostDevice, error) {
	var hds []HostDevice
	for _, pciAddr := range pcis {
		address, err := decoratePciAddressField(pciAddr)
		if err != nil {
			return nil, err
		}

		hostDev := HostDevice{
			Source: HostDeviceSource{
				Address: address,
			},
			Type:    "pci",
			Managed: "yes",
		}

		hds = append(hds, hostDev)
	}

	return hds, nil
}

func createHostDevicesFromMdevUUIDList(mdevUuidList []string) ([]HostDevice, error) {
	var hds []HostDevice
	for _, mdevUuid := range mdevUuidList {
		decoratedAddrField := &Address{
			UUID: mdevUuid,
		}

		hostDev := HostDevice{
			Source: HostDeviceSource{
				Address: decoratedAddrField,
			},
			Type:  "mdev",
			Mode:  "subsystem",
			Model: "vfio-pci",
		}
		hds = append(hds, hostDev)
	}

	return hds, nil
}
