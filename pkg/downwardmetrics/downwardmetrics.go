package downwardmetrics

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd"
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
