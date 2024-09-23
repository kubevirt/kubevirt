package downwardmetrics

import (
	"errors"
	"path/filepath"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	DownwardMetricsSerialDeviceName = "org.github.vhostmd.1"
	DownwardMetricsChannelDir       = util.VirtPrivateDir + "/downwardmetrics-channel"
	DownwardMetricsChannelSocket    = DownwardMetricsChannelDir + "/downwardmetrics.sock"
)

var DownwardMetricsNotEnabledError = errors.New("DownwardMetrics feature is not enabled")

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

func HasDownwardMetricDisk(volumes []v1.Volume) bool {
	for _, volume := range volumes {
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

func IsDownwardMetricsConfigurationInvalid(clusterConfig *virtconfig.ClusterConfig, spec *v1.VirtualMachineInstanceSpec) bool {
	return !clusterConfig.IsDownwardMetricsFeatureEnabled() && (HasDevice(spec) || HasDownwardMetricDisk(spec.Volumes))
}
