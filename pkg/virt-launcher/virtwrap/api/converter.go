package api

import (
	"fmt"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"strconv"

	"os"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/emptydisk"
	"kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/registry-disk"
)

type ConverterContext struct {
	Secrets        map[string]*k8sv1.Secret
	VirtualMachine *v1.VirtualMachine
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
		disk.ReadOnly = toApiReadOnly(*diskDevice.CDRom.ReadOnly)
	}
	disk.Driver = &DiskDriver{
		Name: "qemu",
	}
	disk.Alias = &Alias{Name: diskDevice.Name}
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
		"vm-disks",
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

func Convert_v1_VirtualMachine_To_api_Domain(vm *v1.VirtualMachine, domain *Domain, c *ConverterContext) (err error) {
	precond.MustNotBeNil(vm)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	domain.Spec.Name = VMNamespaceKeyFunc(vm)
	domain.ObjectMeta.Name = vm.ObjectMeta.Name
	domain.ObjectMeta.Namespace = vm.ObjectMeta.Namespace

	// XXX Fix me properly we don't want automatic fallback to qemu
	// We will solve this properly in https://github.com/kubevirt/kubevirt/pull/804
	if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
		domain.Spec.Type = "qemu"
	} else if err != nil {
		return err
	}

	// Spec metadata
	domain.Spec.Metadata.KubeVirt.UID = vm.UID
	if vm.Spec.TerminationGracePeriodSeconds != nil {
		domain.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = *vm.Spec.TerminationGracePeriodSeconds
	}

	domain.Spec.SysInfo = &SysInfo{}
	if vm.Spec.Domain.Firmware != nil {
		domain.Spec.SysInfo.System = []Entry{
			{
				Name:  "uuid",
				Value: string(vm.Spec.Domain.Firmware.UUID),
			},
		}
	}

	if v, ok := vm.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; ok {
		domain.Spec.Memory = QuantityToMegaByte(v)
	}

	volumes := map[string]*v1.Volume{}
	for _, volume := range vm.Spec.Volumes {
		volumes[volume.Name] = volume.DeepCopy()
	}

	devicePerBus := make(map[string]int)
	for _, disk := range vm.Spec.Domain.Devices.Disks {
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

	if vm.Spec.Domain.Devices.Watchdog != nil {
		newWatchdog := &Watchdog{}
		err := Convert_v1_Watchdog_To_api_Watchdog(vm.Spec.Domain.Devices.Watchdog, newWatchdog, c)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Watchdog = newWatchdog
	}

	if vm.Spec.Domain.Clock != nil {
		clock := vm.Spec.Domain.Clock
		newClock := &Clock{}
		err := Convert_v1_Clock_To_api_Clock(clock, newClock, c)
		if err != nil {
			return err
		}
		domain.Spec.Clock = newClock
	}

	if vm.Spec.Domain.Features != nil {
		domain.Spec.Features = &Features{}
		err := Convert_v1_Features_To_api_Features(vm.Spec.Domain.Features, domain.Spec.Features, c)
		if err != nil {
			return err
		}
	}
	apiOst := &vm.Spec.Domain.Machine
	err = Convert_v1_Machine_To_api_OSType(apiOst, &domain.Spec.OS.Type, c)
	if err != nil {
		return err
	}

	if vm.Spec.Domain.CPU != nil {
		domain.Spec.CPU.Topology = &CPUTopology{
			Sockets: 1,
			Cores:   vm.Spec.Domain.CPU.Cores,
			Threads: 1,
		}
		domain.Spec.VCPU = &VCPU{
			Placement: "static",
			CPUs:      vm.Spec.Domain.CPU.Cores,
		}
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
				Path: fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-serial%d", vm.ObjectMeta.Namespace, vm.ObjectMeta.Name, serialPort),
			},
		},
	}

	// Add mandatory vnc device
	domain.Spec.Devices.Graphics = []Graphics{
		{
			Listen: &GraphicsListen{
				Type:   "socket",
				Socket: fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-vnc", vm.ObjectMeta.Namespace, vm.ObjectMeta.Name),
			},
			Type: "vnc",
		},
	}

	return nil
}

func SecretToLibvirtSecret(vm *v1.VirtualMachine, secretName string) string {
	return fmt.Sprintf("%s-%s-%s---", secretName, vm.Namespace, vm.Name)
}

func QuantityToMegaByte(quantity resource.Quantity) Memory {
	return Memory{
		Value: uint(quantity.ToDec().ScaledValue(6)),
		Unit:  "MB",
	}
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
