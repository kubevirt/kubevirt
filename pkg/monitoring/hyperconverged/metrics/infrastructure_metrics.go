package metrics

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	ioprometheusclient "github.com/prometheus/client_model/go"
)

const (
	singleStackIPv6True           = 1.0
	misconfiguredDeschedulerTrue  = 1.0
	misconfiguredDeschedulerFalse = 0.0
)

var (
	infrastructureMetrics = []operatormetrics.Metric{
		singleStackIpv6,
		misconfiguredDescheduler,
	}

	singleStackIpv6 = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_hco_single_stack_ipv6",
			Help: "Indicates whether the underlying cluster is single stack IPv6 (1) or not (0)",
		},
	)

	misconfiguredDescheduler = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_hco_misconfigured_descheduler",
			Help: "Indicates whether the optional descheduler is not properly configured (1) to work with KubeVirt or not (0)",
		},
	)
)

// SetHCOMetricSingleStackIPv6True sets the gauge to 1 (true)
func SetHCOMetricSingleStackIPv6True() {
	singleStackIpv6.Set(singleStackIPv6True)
}

func SetHCOMetricMisconfiguredDescheduler(misconfigured bool) {
	if misconfigured {
		misconfiguredDescheduler.Set(misconfiguredDeschedulerTrue)
	} else {
		misconfiguredDescheduler.Set(misconfiguredDeschedulerFalse)
	}
}

func GetHCOMetrictMisconfiguredDescheduler() (bool, error) {
	dto := &ioprometheusclient.Metric{}
	err := misconfiguredDescheduler.Write(dto)
	if err != nil {
		return false, err
	}

	return dto.Gauge.GetValue() == misconfiguredDeschedulerTrue, nil
}
