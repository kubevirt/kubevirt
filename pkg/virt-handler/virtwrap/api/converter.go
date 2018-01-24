package api

import (
	"fmt"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"strconv"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/registry-disk"
)

type ConverterContext struct {
	Secrets        map[string]*k8sv1.Secret
	VirtualMachine *v1.VirtualMachine
}

func Convert_v1_Disk_To_api_Disk(diskDevice *v1.Disk, disk *Disk) error {

	if diskDevice.Disk != nil {
		disk.Device = "disk"
		disk.Target.Device = diskDevice.Disk.Device
		disk.ReadOnly = toApiReadOnly(diskDevice.Disk.ReadOnly)
	} else if diskDevice.LUN != nil {
		disk.Device = "lun"
		disk.ReadOnly = toApiReadOnly(diskDevice.LUN.ReadOnly)
		disk.Target.Device = diskDevice.LUN.Device
	} else if diskDevice.Floppy != nil {
		disk.Device = "floppy"
		disk.Target.Tray = string(diskDevice.Floppy.Tray)
		disk.ReadOnly = toApiReadOnly(diskDevice.Floppy.ReadOnly)
		disk.Target.Device = diskDevice.Floppy.Device
	} else if diskDevice.CDRom != nil {
		disk.Device = "cdrom"
		disk.Target.Tray = string(diskDevice.CDRom.Tray)
		disk.ReadOnly = toApiReadOnly(*diskDevice.CDRom.ReadOnly)
		disk.Target.Device = diskDevice.CDRom.Device
	}
	disk.Driver = &DiskDriver{
		Name: "qemu",
	}
	disk.Alias = &Alias{Name: diskDevice.Name}
	return nil
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

	if source.ISCSI != nil {
		return Convert_v1_ISCSIVolumeSource_To_api_Disk(source.ISCSI, disk, c)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.Name)
}
func Convert_v1_ISCSIVolumeSource_To_api_Disk(source *k8sv1.ISCSIVolumeSource, disk *Disk, c *ConverterContext) error {

	disk.Type = "network"
	disk.Driver.Type = "raw"
	disk.Driver.Cache = "none"

	disk.Source.Name = fmt.Sprintf("%s/%d", source.IQN, source.Lun)
	disk.Source.Protocol = "iscsi"

	hostPort := strings.Split(source.TargetPortal, ":")

	disk.Source.Host = &DiskSourceHost{}
	disk.Source.Host.Name = hostPort[0]
	if len(hostPort) > 1 {
		disk.Source.Host.Port = hostPort[1]
	}

	// This iscsi device has auth associated with it.
	if source.SecretRef != nil && source.SecretRef.Name != "" {

		secret := c.Secrets[source.SecretRef.Name]
		if secret == nil {
			return fmt.Errorf("failed to find secret for disk auth %s", source.SecretRef.Name)
		}
		userValue, ok := secret.Data["node.session.auth.username"]
		if ok == false {
			return fmt.Errorf("failed to find username for disk auth %s", source.SecretRef.Name)
		}

		disk.Auth = &DiskAuth{
			Secret: &DiskSecret{
				Type:  "iscsi",
				Usage: SecretToLibvirtSecret(c.VirtualMachine, source.SecretRef.Name),
			},
			Username: string(userValue),
		}
	}
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
				Value: string(vm.Spec.Domain.Firmware.UID),
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

	for _, disk := range vm.Spec.Domain.Devices.Disks {
		newDisk := Disk{}
		err := Convert_v1_Disk_To_api_Disk(&disk, &newDisk)
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
	if vm.Spec.Domain.Machine != nil {
		apiOst := vm.Spec.Domain.Machine
		err := Convert_v1_Machine_To_api_OSType(apiOst, &domain.Spec.OS.Type, c)
		if err != nil {
			return err
		}
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
