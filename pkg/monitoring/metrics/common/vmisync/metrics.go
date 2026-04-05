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

package vmisync

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/log"
)

const (
	VirtControllerComponent = "virt-controller"
	VirtHandlerComponent    = "virt-handler"
)

var (
	vmiSyncMetrics = []operatormetrics.Metric{
		vmiSyncTotal,
	}

	vmiSyncTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_sync_total",
			Help: "Total number of times virt-controller or virt-handler has synced a VirtualMachineInstance.",
		},
		[]string{"controller", "namespace", "name"},
	)
)

func SetupMetrics() error {
	return operatormetrics.RegisterMetrics(vmiSyncMetrics)
}

func IncVMISyncMetric(controller, namespace, name string) {
	counter, err := vmiSyncTotal.GetMetricWithLabelValues(controller, namespace, name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get vmi sync counter for controller %s, vmi %s/%s", controller, namespace, name)
		return
	}
	counter.Inc()
}

func DeleteVMISyncMetric(controller, namespace, name string) {
	vmiSyncTotal.DeleteLabelValues(controller, namespace, name)
}
