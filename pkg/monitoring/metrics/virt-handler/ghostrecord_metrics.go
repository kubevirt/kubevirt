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
 *
 */

package virthandler

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/log"
)

const (
	StaleVMIReconciliationCleaned = "cleaned"
	StaleVMIReconciliationError   = "error"
)

var (
	ghostRecordMetrics = []operatormetrics.Metric{
		staleVMIReconciliationsTotal,
	}

	staleVMIReconciliationsTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_virt_handler_stale_vmi_reconciliations_total",
			Help: "Number of terminal stale VMI reconciliation outcomes by result (cleaned, error).",
		},
		[]string{"result"},
	)
)

func GetStaleVMIReconciliationsTotal() *operatormetrics.CounterVec {
	return staleVMIReconciliationsTotal
}

func IncStaleVMIReconciliation(result string) {
	counter, err := staleVMIReconciliationsTotal.GetMetricWithLabelValues(result)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get stale VMI reconciliation counter for result %s", result)
		return
	}
	counter.Inc()
}
