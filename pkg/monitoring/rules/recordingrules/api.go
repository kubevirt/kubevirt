/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package recordingrules

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var apiRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "cluster:kubevirt_api_request_deprecated_total:sum",
			Help: "The total number of requests to deprecated KubeVirt APIs, by API verb (e.g., LIST, WATCH).",
		},
		MetricType: operatormetrics.CounterType,
		Expr: intstr.FromString(
			"group by (group,version,resource,subresource) " +
				"(apiserver_requested_deprecated_apis{group=\"kubevirt.io\"}) * " +
				"on (group,version,resource,subresource) group_right() " +
				"sum by (group,version,resource,subresource,verb) (apiserver_request_total)",
		),
	},
}
