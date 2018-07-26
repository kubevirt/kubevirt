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
	"fmt"
	"os"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"strconv"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	"kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/registry-disk"
)

const (
	CPUModeHostPassthrough = "host-passthrough"
	CPUModeHostModel       = "host-model"
)

type ConverterContext struct {
	AllowEmulation bool
	Secrets        map[string]*k8sv1.Secret
	VirtualMachine *v1.VirtualMachineInstance
}

func Convert_v1_Disk_To_api_Disk(diskDevice *v1.Disk, disk *Disk, devicePerBus map[string]int) error {

	if diskDevice.Disk != nil {
		disk.Device = "disk"
		disk.Target.Bus = diskDevice.Disk.Bus
		disk.Target.Device = makeDeviceName(diskDevice.Disk.Bus, devicePerBus)
		disk.ReadOnly = toApiReadOnly(diskDevice.Disk.ReadOnly)
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
		Name: "qemu",
	}
	disk.Alias = &Alias{Name: diskDevice.Name}
	if diskDevice.BootOrder != nil {
		disk.BootOrder = &BootOrder{Order: *diskDevice.BootOrder}
	}

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
	case "ide":
		prefix = "hd"
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

func Convert_v1_Volume_To_api_Disk(source *v1.Volume, disk *Disk, c *ConverterContext) error {

	if source.RegistryDisk != nil {
		return Convert_v1_RegistryDiskSource_To_api_Disk(source.Name, source.RegistryDisk, disk, c)
	}

	if source.CloudInitNoCloud != nil {
		return Convert_v1_CloudInitNoCloudSource_To_api_Disk(source.CloudInitNoCloud, disk, c)
	}

	if source.PersistentVolumeClaim != nil {
		return Covert_v1_FilesystemVolumeSource_To_api_Disk(source.Name, disk, c)
	}

	if source.Ephemeral != nil {
		return Convert_v1_EphemeralVolumeSource_To_api_Disk(source.Name, source.Ephemeral, disk, c)
	}
	if source.EmptyDisk != nil {
		return Convert_v1_EmptyDiskSource_To_api_Disk(source.Name, source.EmptyDisk, disk, c)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.Name)
}

func Covert_v1_FilesystemVolumeSource_To_api_Disk(volumeName string, disk *Disk, c *ConverterContext) error {

	disk.Type = "file"
	disk.Driver.Type = "raw"
	disk.Source.File = filepath.Join(
		"/var/run/kubevirt-private",
		"vmi-disks",
		volumeName,
		"disk.img")
	return nil
}

func Convert_v1_CloudInitNoCloudSource_To_api_Disk(source *v1.CloudInitNoCloudSource, disk *Disk, c *ConverterContext) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.Name)
	}

	disk.Source.File = fmt.Sprintf("%s/%s", cloudinit.GetDomainBasePath(c.VirtualMachine.Name, c.VirtualMachine.Namespace), cloudinit.NoCloudFile)
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

func Convert_v1_RegistryDiskSource_To_api_Disk(volumeName string, _ *v1.RegistryDiskSource, disk *Disk, c *ConverterContext) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.Name)
	}

	disk.Type = "file"
	diskPath, diskType, err := registrydisk.GetFilePath(c.VirtualMachine, volumeName)
	if err != nil {
		return err
	}
	disk.Driver.Type = diskType
	disk.Source.File = diskPath
	return nil
}

func Convert_v1_EphemeralVolumeSource_To_api_Disk(volumeName string, source *v1.EphemeralVolumeSource, disk *Disk, c *ConverterContext) error {
	disk.Type = "file"
	disk.Driver.Type = "qcow2"
	disk.Source.File = ephemeraldisk.GetFilePath(volumeName)
	disk.BackingStore = &BackingStore{}

	backingDisk := &Disk{Driver: &DiskDriver{}}
	err := Covert_v1_FilesystemVolumeSource_To_api_Disk(volumeName, backingDisk, c)
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
	return nil
}

func Convert_v1_VirtualMachine_To_api_Domain(vmi *v1.VirtualMachineInstance, domain *Domain, c *ConverterContext) (err error) {
	precond.MustNotBeNil(vmi)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	domain.Spec.Name = VMINamespaceKeyFunc(vmi)
	domain.ObjectMeta.Name = vmi.ObjectMeta.Name
	domain.ObjectMeta.Namespace = vmi.ObjectMeta.Namespace

	if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
		if c.AllowEmulation {
			logger := log.DefaultLogger()
			logger.Infof("Hardware emulation device '/dev/kvm' not present. Using software emulation.")
			domain.Spec.Type = "qemu"
		} else {
			return fmt.Errorf("hardware emulation device '/dev/kvm' not present")
		}
	} else if err != nil {
		return err
	}

	// Spec metadata
	domain.Spec.Metadata.KubeVirt.UID = vmi.UID
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		domain.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}

	domain.Spec.SysInfo = &SysInfo{}
	if vmi.Spec.Domain.Firmware != nil {
		domain.Spec.SysInfo.System = []Entry{
			{
				Name:  "uuid",
				Value: string(vmi.Spec.Domain.Firmware.UUID),
			},
		}
	}

	if v, ok := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
		if domain.Spec.Memory, err = QuantityToByte(v); err != nil {
			return err
		}
	}

	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		domain.Spec.MemoryBacking = &MemoryBacking{
			HugePages: &HugePages{},
		}
	}

	volumes := map[string]*v1.Volume{}
	for _, volume := range vmi.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
	}

	devicePerBus := make(map[string]int)
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		newDisk := Disk{}

		err := Convert_v1_Disk_To_api_Disk(&disk, &newDisk, devicePerBus)
		if err != nil {
			return err
		}
		volume := volumes[disk.VolumeName]
		if volume == nil {
			return fmt.Errorf("No matching volume with name %s found", disk.VolumeName)
		}
		err = Convert_v1_Volume_To_api_Disk(volume, &newDisk, c)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, newDisk)
	}

	if vmi.Spec.Domain.Devices.Watchdog != nil {
		newWatchdog := &Watchdog{}
		err := Convert_v1_Watchdog_To_api_Watchdog(vmi.Spec.Domain.Devices.Watchdog, newWatchdog, c)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Watchdog = newWatchdog
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
		// Set VM CPU cores
		if vmi.Spec.Domain.CPU.Cores != 0 {
			domain.Spec.CPU.Topology = &CPUTopology{
				Sockets: 1,
				Cores:   vmi.Spec.Domain.CPU.Cores,
				Threads: 1,
			}
			domain.Spec.VCPU = &VCPU{
				Placement: "static",
				CPUs:      vmi.Spec.Domain.CPU.Cores,
			}
		}

		// Set VM CPU model and vendor
		if vmi.Spec.Domain.CPU.Model != "" {
			if vmi.Spec.Domain.CPU.Model == CPUModeHostModel || vmi.Spec.Domain.CPU.Model == CPUModeHostPassthrough {
				domain.Spec.CPU.Mode = vmi.Spec.Domain.CPU.Model
			} else {
				domain.Spec.CPU.Mode = "custom"
				domain.Spec.CPU.Model = vmi.Spec.Domain.CPU.Model
			}
		}
	}

	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		domain.Spec.CPU.Mode = CPUModeHostModel
	}

	// Add mandatory console device
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
				Path: fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-serial%d", vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name, serialPort),
			},
		},
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
					Socket: fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-vnc", vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name),
				},
				Type: "vnc",
			},
		}
	}

	// Add mandatory interface
	interfaceType := "virtio"

	_, ok := vmi.ObjectMeta.Annotations[v1.InterfaceModel]
	if ok {
		interfaceType = vmi.ObjectMeta.Annotations[v1.InterfaceModel]
	}

	findNetwork := func(nets []v1.Network, name string) (*v1.Network, error) {
		for _, net := range nets {
			if net.Name == name {
				return &net, nil
			}
		}
		return nil, fmt.Errorf("failed to find network %s", name)
	}

	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		net, err := findNetwork(vmi.Spec.Networks, iface.Name)
		if err != nil {
			return err
		}
		if net.Pod == nil {
			return fmt.Errorf("network interface type not supported for %s", iface.Name)
		}
		// TODO:(ihar) consider abstracting interface type conversion /
		// detection into drivers
		domainIface := Interface{
			Model: &Model{
				Type: interfaceType,
			},
			Type: "bridge",
			Source: InterfaceSource{
				Bridge: DefaultBridgeName,
			},
			Alias: &Alias{
				Name: iface.Name,
			},
		}
		domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, domainIface)
	}

	return nil
}

func SecretToLibvirtSecret(vmi *v1.VirtualMachineInstance, secretName string) string {
	return fmt.Sprintf("%s-%s-%s---", secretName, vmi.Namespace, vmi.Name)
}

func QuantityToByte(quantity resource.Quantity) (Memory, error) {
	memorySize, _ := quantity.AsInt64()
	if memorySize < 0 {
		return Memory{Unit: "B"}, fmt.Errorf("Memory size '%s' must be greater than or equal to 0", quantity.String())
	}
	return Memory{
		Value: uint64(memorySize),
		Unit:  "B",
	}, nil
}

func boolToOnOff(value *bool, defaultOn bool) string {
	if value == nil {
		if defaultOn {
			return "on"
		}
		return "off"
	}

	if *value {
		return "on"
	}
	return "off"
}

func boolToYesNo(value *bool, defaultYes bool) string {
	if value == nil {
		if defaultYes {
			return "yes"
		}
		return "no"
	}

	if *value {
		return "yes"
	}
	return "no"
}
