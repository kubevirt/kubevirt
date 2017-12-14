package api

import (
	"fmt"
	"net"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/registry-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cache"
)

type Context struct {
	Secrets        map[string]*k8sv1.Secret
	VirtualMachine *v1.VirtualMachine
}

func Convert_v1_Disk_To_api_Disk(diskDevice *v1.Disk, disk *Disk) error {

	var baseAtts *v1.DiskBaseTarget
	if diskDevice.Disk != nil {
		disk.Device = "disk"
		baseAtts = &diskDevice.Disk.DiskBaseTarget
		disk.ReadOnly = toApiReadOnly(diskDevice.Disk.ReadOnly)
	} else if diskDevice.LUN != nil {
		disk.Device = "lun"
		disk.ReadOnly = toApiReadOnly(diskDevice.LUN.ReadOnly)
		baseAtts = &diskDevice.LUN.DiskBaseTarget
	} else if diskDevice.Floppy != nil {
		disk.Device = "floppy"
		disk.Target.Tray = string(diskDevice.Floppy.Tray)
		disk.ReadOnly = toApiReadOnly(diskDevice.Floppy.ReadOnly)
		baseAtts = &diskDevice.Floppy.DiskBaseTarget
	} else if diskDevice.CDRom != nil {
		disk.Device = "cdrom"
		disk.Target.Tray = string(diskDevice.Floppy.Tray)
		disk.ReadOnly = toApiReadOnly(*diskDevice.CDRom.ReadOnly)
		baseAtts = &diskDevice.CDRom.DiskBaseTarget
	}
	disk.Target.Device = baseAtts.Device
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

func Convert_v1_Volume_To_api_Disk(source *v1.Volume, disk *Disk, c *Context) error {

	if source.RegistryDisk != nil {
		return Convert_v1_RegistryDiskSource_To_api_Disk(source.RegistryDisk, disk, c)
	}

	if source.CloudInitNoCloud != nil {
		return Convert_v1_CloudInitNoCloudSource_To_api_Disk(source.CloudInitNoCloud, disk, c)
	}

	if source.ISCSI != nil {
		return Convert_v1_ISCSIVolumeSource_To_api_Disk(source.ISCSI, disk, c)
	}

	return fmt.Errorf("disk %s references an unsupported source", disk.Alias.Name)
}
func Convert_v1_ISCSIVolumeSource_To_api_Disk(source *k8sv1.ISCSIVolumeSource, disk *Disk, c *Context) error {

	disk.Type = "network"
	disk.Driver.Type = "raw"

	disk.Source.Name = fmt.Sprintf("%s/%d", source.IQN, source.Lun)
	disk.Source.Protocol = "iscsi"

	hostPort := strings.Split(source.TargetPortal, ":")
	// FIXME ugly hack to resolve the IP from dns, since qemu is not in the right namespace
	// FIXME Move this out of the converter!
	ipAddrs, err := net.LookupIP(hostPort[0])
	if err != nil || len(ipAddrs) < 1 {
		return fmt.Errorf("unable to resolve host '%s': %s", hostPort[0], err)
	}

	disk.Source.Host = &DiskSourceHost{}
	disk.Source.Host.Name = ipAddrs[0].String()
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

func Convert_v1_CloudInitNoCloudSource_To_api_Disk(source *v1.CloudInitNoCloudSource, disk *Disk, c *Context) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.Name)
	}

	disk.Source.File = fmt.Sprintf("%s/%s", cloudinit.GetDomainBasePath(c.VirtualMachine.Name, c.VirtualMachine.Namespace), cloudinit.NoCloudFile)
	disk.Type = "file"
	disk.Device = "disk"
	disk.Driver.Type = "raw"
	return nil
}

func Convert_v1_RegistryDiskSource_To_api_Disk(source *v1.RegistryDiskSource, disk *Disk, c *Context) error {
	if disk.Type == "lun" {
		return fmt.Errorf("device %s is of type lun. Not compatible with a file based disk", disk.Alias.Name)
	}

	disk.Type = "file"
	disk.Device = "volume"
	diskPath, diskType, err := registrydisk.GetFilePath(c.VirtualMachine, disk.Alias.Name)
	if err != nil {
		return err
	}
	disk.Driver.Type = diskType
	disk.Source.File = diskPath
	return nil
}

func Convert_v1_VirtualMachine_To_api_Domain(vm *v1.VirtualMachine, domain *Domain, c *Context) (err error) {
	precond.MustNotBeNil(vm)
	precond.MustNotBeNil(domain)
	precond.MustNotBeNil(c)

	domName := cache.VMNamespaceKeyFunc(vm)
	domain.Spec.Name = domName
	domain.Spec.UUID = string(vm.GetObjectMeta().GetUID())
	domain.ObjectMeta.Name = vm.ObjectMeta.Name
	domain.ObjectMeta.Namespace = vm.ObjectMeta.Namespace
	domain.Spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	domain.Spec.Type = "qemu"
	domain.Spec.SysInfo.Type = "smbios"
	domain.Spec.SysInfo.System = []Entry{
		{
			Name:  "uuid",
			Value: string(vm.Spec.Domain.Firmware.UID),
		},
	}

	if v, ok := vm.Spec.Domain.Resources.Initial[k8sv1.ResourceMemory]; ok {
		domain.Spec.Memory = QuantityToMegaByte(v)
	}

	volumes := map[string]*v1.Volume{}
	for _, volume := range vm.Spec.Volumes {
		volumes[volume.Name] = &volume
	}

	for _, disk := range vm.Spec.Domain.Devices.Disks {
		newDisk := &Disk{}
		Convert_v1_Disk_To_api_Disk(&disk, newDisk)
		Convert_v1_Volume_To_api_Disk(volumes[disk.Name], newDisk, c)
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
