package vhostmd

import "kubevirt.io/kubevirt/pkg/safepath"

func NewMetricsIODisk(filePath *safepath.Path) *vhostmd {
	return &vhostmd{filePath: filePath}
}
