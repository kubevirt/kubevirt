/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package virt_handler

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"
	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/client"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/workqueue"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/domainstats"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/migrationdomainstats"
)

func SetupMetrics(nodeName string, MaxRequestsInFlight int, vmiInformer cache.SharedIndexInformer, machines []libvirtxml.CapsGuestMachine) error {
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

	domainstats.SetupDomainStatsCollector(MaxRequestsInFlight, vmiInformer)

	if err := migrationdomainstats.SetupMigrationStatsCollector(vmiInformer); err != nil {
		return err
	}

	return operatormetrics.RegisterCollector(
		domainstats.Collector,
		domainstats.DomainDirtyRateStatsCollector,
		migrationdomainstats.MigrationStatsCollector,
		domainstats.GuestAgentInfoCollector,
	)
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}
