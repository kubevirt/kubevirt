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
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/log"
)

var (
	guestPanicMetrics = []operatormetrics.Metric{
		guestOSPanicTotal,
	}

	guestOSPanicTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_os_panic_total",
			Help: "Total number of guest OS panic events detected, partitioned by VMI and panic type.",
		},
		[]string{"namespace", "name", "type", "bugcheck_code"},
	)
)

func GetGuestOSPanicTotal() *operatormetrics.CounterVec {
	return guestOSPanicTotal
}

func IncGuestOSPanic(namespace, name, panicType string, bugcheckCode uint64) {
	code := "unknown"
	if bugcheckCode != 0 {
		code = fmt.Sprintf("0x%x", bugcheckCode)
	}
	counter, err := guestOSPanicTotal.GetMetricWithLabelValues(namespace, name, panicType, code)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get guest OS panic counter for vmi %s/%s", namespace, name)
		return
	}
	counter.Inc()
}
