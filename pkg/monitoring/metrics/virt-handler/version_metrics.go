package virt_handler

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"kubevirt.io/client-go/version"
)

var (
	versionMetrics = []operatormetrics.Metric{
		versionInfo,
	}

	versionInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_info",
			Help: "Version information.",
		},
		[]string{"goversion", "kubeversion"},
	)
)

func SetVersionInfo() {
	info := version.Get()
	versionInfo.WithLabelValues(info.GoVersion, info.GitVersion).Set(1)
}
