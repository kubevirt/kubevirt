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

package virt_handler

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/version"
)

var (
	versionMetrics = []operatormetrics.Metric{
		versionInfo,
	}

	versionInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_info",
			Help: "Version information.",
		},
		[]string{"goversion", "kubeversion"},
	)
)

func SetVersionInfo() {
	info := version.Get()
	versionInfo.WithLabelValues(info.GoVersion, info.GitVersion).Set(1)
}
