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

package recordingrules

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var apiRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_api_request_deprecated_total",
			Help: "The total number of requests to deprecated KubeVirt APIs.",
		},
		MetricType: operatormetrics.CounterType,
		Expr:       intstr.FromString("group by (group,version,resource,subresource) (apiserver_requested_deprecated_apis{group=\"kubevirt.io\"}) * on (group,version,resource,subresource) group_right() sum by (group,version,resource,subresource,verb) (apiserver_request_total)"),
	},
}
