package domainstats

import "github.com/machadovilaca/operator-observability/pkg/operatormetrics"

var (
	filesystemCapacityBytes = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_filesystem_capacity_bytes",
			Help: "Total VM filesystem capacity in bytes.",
		},
	)

	filesystemUsedBytes = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_filesystem_used_bytes",
			Help: "Used VM filesystem capacity in bytes.",
		},
	)
)

type filesystemMetrics struct{}

func (filesystemMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		filesystemCapacityBytes,
		filesystemUsedBytes,
	}
}

func (filesystemMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	for _, fsStat := range vmiReport.vmiStats.FsStats.Items {
		fsLabels := map[string]string{
			"disk_name":        fsStat.DiskName,
			"mount_point":      fsStat.MountPoint,
			"file_system_type": fsStat.FileSystemType,
		}

		crs = append(crs, vmiReport.newCollectorResultWithLabels(filesystemCapacityBytes, float64(fsStat.TotalBytes), fsLabels))
		crs = append(crs, vmiReport.newCollectorResultWithLabels(filesystemUsedBytes, float64(fsStat.UsedBytes), fsLabels))
	}

	return crs
}
