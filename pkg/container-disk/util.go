package containerdisk

import virtv1 "kubevirt.io/api/core/v1"

func IsHotplugContainerDisk(v *virtv1.Volume) bool {
	return v != nil && IsHotplugContainerDiskSource(v.VolumeSource)
}
func IsHotplugContainerDiskSource(vs virtv1.VolumeSource) bool {
	return vs.ContainerDisk != nil && vs.ContainerDisk.Hotpluggable
}
