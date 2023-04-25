package downwardmetrics

import (
	"path/filepath"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	DownwardMetricsSerialDeviceName = "org.github.vhostmd.1"
	DownwardMetricsChannelDir       = util.VirtPrivateDir + "/downwardmetrics-channel"
	DownwardMetricsChannelSocket    = DownwardMetricsChannelDir + "/downwardmetrics.sock"
)

func CreateDownwardMetricDisk(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if volume.DownwardMetrics != nil {
			return vhostmd.NewMetricsIODisk(config.DownwardMetricDisk).Create()
		}
	}
	return nil
}

func FormatDownwardMetricPath(pid int) string {
	return filepath.Join("/proc", strconv.Itoa(pid), "/root", config.DownwardMetricDisk)
}

func HasDownwardMetricDisk(vmi *v1.VirtualMachineInstance) bool {
	for _, volume := range vmi.Spec.Volumes {
		if volume.DownwardMetrics != nil {
			return true
		}
	}
	return false
}

func ChannelSocketPathOnHost(pid int) string {
	return filepath.Join("/proc", strconv.Itoa(pid), "root", DownwardMetricsChannelSocket)
}
