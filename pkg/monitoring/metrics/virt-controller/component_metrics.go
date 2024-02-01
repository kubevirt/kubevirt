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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package virt_controller

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	ioprometheusclient "github.com/prometheus/client_model/go"
)

var (
	componentMetrics = []operatormetrics.Metric{
		virtControllerLeading,
		virtControllerReady,
	}

	virtControllerLeading = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_virt_controller_leading_status",
			Help: "Indication for an operating virt-controller.",
		},
	)

	virtControllerReady = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_virt_controller_ready_status",
			Help: "Indication for a virt-controller that is ready to take the lead.",
		},
	)
)

func GetVirtControllerMetric() (*ioprometheusclient.Metric, error) {
	dto := &ioprometheusclient.Metric{}
	err := virtControllerLeading.Write(dto)
	return dto, err
}

func SetVirtControllerLeading() {
	virtControllerLeading.Set(1)
}

func SetVirtControllerReady() {
	virtControllerReady.Set(1)
}

func SetVirtControllerNotReady() {
	virtControllerReady.Set(0)
}
