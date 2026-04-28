/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtapi

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	v1 "kubevirt.io/api/core/v1"
)

var (
	vmMetrics = []operatormetrics.Metric{
		vmsCreatedCounter,
	}

	vmsCreatedCounter = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_created_by_pod_total",
			Help: "[Deprecated] The total number of VMs created by namespace and virt-api pod, since install.",
		},
		[]string{"namespace"},
	)
)

func NewVMCreated(vm *v1.VirtualMachine) {
	vmsCreatedCounter.WithLabelValues(vm.Namespace).Inc()
}
