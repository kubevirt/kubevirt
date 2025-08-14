package libmonitoring

import (
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/workqueue"
	virtapi "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	virtcontroller "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	virthandler "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
	virtoperator "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-operator"
	"kubevirt.io/kubevirt/pkg/monitoring/rules"
)

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

	if err := virthandler.SetupMetrics("", "", 0, nil, nil); err != nil {
		return err
	}

	if err := rules.SetupRules(""); err != nil {
		return err
	}

	// Create dummy worqueue metrics
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
