/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virthandler

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"
	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/client"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/workqueue"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/domainstats"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/migrationdomainstats"
)

func SetupMetrics(
	nodeName string, maxRequestsInFlight int,
	vmiInformer cache.SharedIndexInformer, machines []libvirtxml.CapsGuestMachine,
) error {
	if err := workqueue.SetupMetrics(); err != nil {
		return err
	}

	if err := client.SetupMetrics(); err != nil {
		return err
	}

	if err := operatormetrics.RegisterMetrics(versionMetrics, machineTypeMetrics); err != nil {
		return err
	}
	SetVersionInfo()
	ReportDeprecatedMachineTypes(machines, nodeName)

	domainstats.SetupDomainStatsCollector(maxRequestsInFlight, vmiInformer)

	if err := migrationdomainstats.SetupMigrationStatsCollector(vmiInformer); err != nil {
		return err
	}

	return operatormetrics.RegisterCollector(
		domainstats.Collector,
		domainstats.DomainDirtyRateStatsCollector,
		migrationdomainstats.MigrationStatsCollector,
	)
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}
