package downwardmetrics

import (
	"errors"
	"os"
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
	vmPath := filepath.Join("/proc", strconv.Itoa(pid), "/root")

	// Backwards compatibility
	//TODO: remove this block of code when we do not support updates from old versions.
	oldDownwardMetricDisk := filepath.Join(config.DownwardAPIDisksDir, config.VhostmdDiskName)
	_, err := os.Stat(filepath.Join(vmPath, oldDownwardMetricDisk))
	if !errors.Is(err, os.ErrNotExist) {
		// Updating from old version. Let's restore the old path
		return oldDownwardMetricDisk
	}
	return filepath.Join(vmPath, config.DownwardMetricDisk)
}

func HasDownwardMetricDisk(vmi *v1.VirtualMachineInstance) bool {
	for _, volume := range vmi.Spec.Volumes {
		if volume.DownwardMetrics != nil {
			return true
		}
	}
	return false
}

func HasDevice(spec *v1.VirtualMachineInstanceSpec) bool {
	return spec.Domain.Devices.DownwardMetrics != nil
}

func ChannelSocketPathOnHost(pid int) string {
	return filepath.Join("/proc", strconv.Itoa(pid), "root", DownwardMetricsChannelSocket)
}
