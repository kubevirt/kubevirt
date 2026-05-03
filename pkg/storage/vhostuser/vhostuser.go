package vhostuser

import (
	"path/filepath"
	"strconv"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const maxQueuesCount = 8

func IsVhostUserVolumeSource(volume *v1.Volume) bool {
	return volume != nil && volume.VhostUser != nil
}

func IsVhostUserVolume(vmi *v1.VirtualMachineInstance, name string) bool {
	return FindVolume(vmi, name) != nil
}

func FindVolume(vmi *v1.VirtualMachineInstance, name string) *v1.Volume {
	for i := range vmi.Spec.Volumes {
		volume := &vmi.Spec.Volumes[i]
		if volume.Name == name && IsVhostUserVolumeSource(volume) {
			return volume
		}
	}
	return nil
}

func BuildDiskMap(vmi *v1.VirtualMachineInstance) map[string]bool {
	result := make(map[string]bool)
	for i := range vmi.Spec.Volumes {
		volume := &vmi.Spec.Volumes[i]
		if IsVhostUserVolumeSource(volume) {
			result[volume.Name] = true
		}
	}
	return result
}

func socketRoot(diskXML *api.Disk) string {
	if diskXML.Source.Path != "" {
		return filepath.Dir(diskXML.Source.Path)
	}
	if diskXML.Source.File != "" {
		return filepath.Dir(diskXML.Source.File)
	}
	if diskXML.Source.Dev != "" {
		return filepath.Dir(diskXML.Source.Dev)
	}
	return ""
}

// SocketPath resolves the vhost-user socket path relative to the root of the associated PVC-backed volume.
func SocketPath(diskXML *api.Disk, vhost *v1.VhostUserVolumeSource) string {
	return filepath.Join(socketRoot(diskXML), vhost.Socket.Path)
}

func ApplyToDomainDisk(vmi *v1.VirtualMachineInstance, disk v1.Disk, volume v1.Volume, diskXML *api.Disk) {
	if !IsVhostUserVolumeSource(&volume) {
		return
	}

	vhost := volume.VhostUser
	numQueues := defaultQueues(vmi)
	if vhost.Queues != nil {
		numQueues = *vhost.Queues
	}
	if vcpuCount(vmi) > 0 && numQueues > vcpuCount(vmi) {
		log.Log.Warningf("vhost-user-blk(%q): requested number of queues %d is greater than vCPUs count %d",
			disk.Name,
			numQueues,
			vcpuCount(vmi),
		)
	}

	reconnectTimeout := uint(1)
	if vhost.ReconnectTimeoutSeconds != nil {
		reconnectTimeout = *vhost.ReconnectTimeoutSeconds
	}

	*diskXML = api.Disk{
		Device: "disk",
		Type:   "vhostuser",
		Source: api.DiskSource{
			Type: "unix",
			Path: SocketPath(diskXML, vhost),
			Reconnect: &api.VhostReconnect{
				Enabled: "yes",
				Timeout: strconv.FormatUint(uint64(reconnectTimeout), 10),
			},
		},
		Target: api.DiskTarget{
			Bus:    v1.DiskBusVirtio,
			Device: diskXML.Target.Device,
		},
		Driver: &api.DiskDriver{
			Name:   "qemu",
			Type:   "raw",
			Queues: &numQueues,
		},
		Alias:     diskXML.Alias,
		BootOrder: diskXML.BootOrder,
		Address:   diskXML.Address,
		Model:     diskXML.Model,
	}
}

func defaultQueues(vmi *v1.VirtualMachineInstance) uint {
	count := vcpuCount(vmi)
	if count == 0 {
		return 1
	}
	if count > maxQueuesCount {
		return maxQueuesCount
	}
	return count
}

func vcpuCount(vmi *v1.VirtualMachineInstance) uint {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Cores == 0 {
		return 1
	}
	return uint(vmi.Spec.Domain.CPU.Cores)
}
