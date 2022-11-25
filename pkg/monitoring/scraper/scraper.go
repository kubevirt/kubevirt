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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package scraper

import (
	"github.com/prometheus/client_golang/prometheus"
	"kubevirt.io/client-go/log"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type Scraper interface {
	Scrape()
	PushConstMetric(*prometheus.Desc, prometheus.ValueType, float64, ...string)
}

type PrometheusScraper struct {
	Ch            chan<- prometheus.Metric
	ClusterConfig *virtconfig.ClusterConfig
}

func (ps *PrometheusScraper) Scrape() {}

func (ps *PrometheusScraper) PushConstMetric(desc *prometheus.Desc, valueType prometheus.ValueType, value float64, labelValues ...string) {
	mv, err := prometheus.NewConstMetric(
		desc,
		valueType,
		value,
		labelValues...,
	)
	if err != nil {
		log.Log.V(4).Warningf("Error creating the new const metric for %s: %s", desc, err)
		return
	}
	ps.Ch <- mv
}
