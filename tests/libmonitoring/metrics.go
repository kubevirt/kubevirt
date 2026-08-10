package libmonitoring

import (
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/workqueue"
	virtapi "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	virtcontroller "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	virthandler "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
	virtoperator "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-operator"
	"kubevirt.io/kubevirt/pkg/monitoring/rules"
)

// stubClusterConfig implements domainstats.ClusterConfig for metric registration
// in tests, where the guest device metrics feature gate is not relevant.
type stubClusterConfig struct{}

func (stubClusterConfig) GuestDeviceMetricsEnabled() bool { return false }

func RegisterAllMetrics() error {
	if err := virtcontroller.SetupMetrics(nil, nil, nil, nil); err != nil {
		return err
	}

	if err := virtcontroller.RegisterLeaderMetrics(); err != nil {
		return err
	}

	if err := virtapi.SetupMetrics(); err != nil {
		return err
	}

	if err := virtoperator.SetupMetrics(); err != nil {
		return err
	}

	if err := virtoperator.RegisterLeaderMetrics(); err != nil {
		return err
	}

	if err := virthandler.SetupMetrics("", 0, nil, nil, stubClusterConfig{}); err != nil {
		return err
	}

	if err := rules.SetupRules(""); err != nil {
		return err
	}

	// Create dummy workqueue metrics
	workqueueMetricsProvider := workqueue.NewPrometheusMetricsProvider()
	workqueueMetricsProvider.NewAddsMetric("")
	workqueueMetricsProvider.NewDepthMetric("")
	workqueueMetricsProvider.NewLatencyMetric("")
	workqueueMetricsProvider.NewWorkDurationMetric("")
	workqueueMetricsProvider.NewUnfinishedWorkSecondsMetric("")
	workqueueMetricsProvider.NewLongestRunningProcessorSecondsMetric("")
	workqueueMetricsProvider.NewRetriesMetric("")

	return nil
}
