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
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/client-go/log"
)

var (
	vmiSyncMetrics = []operatormetrics.Metric{
		vmiSyncTotal,
	}

	vmiSyncTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_sync_total",
			Help: "Total number of times a VirtualMachineInstance has been synced.",
		},
		[]string{"namespace", "name"},
	)
)

func SetupMetrics() error {
	return operatormetrics.RegisterMetrics(vmiSyncMetrics)
}

func VMISynced(namespace, name string) {
	counter, err := vmiSyncTotal.GetMetricWithLabelValues(namespace, name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get vmi sync counter for vmi %s/%s", namespace, name)
		return
	}
	counter.Inc()
}

func ResetVMISync(key string) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to parse key %s for vmi sync metric deletion", key)
		return
	}
	vmiSyncTotal.DeleteLabelValues(namespace, name)
}
