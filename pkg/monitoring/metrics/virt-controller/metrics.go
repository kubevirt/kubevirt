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
	"k8s.io/client-go/tools/cache"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var (
	metrics = [][]operatormetrics.Metric{
		operatorMetrics,
	}

	vmiInformer                 cache.SharedIndexInformer
	clusterInstanceTypeInformer cache.SharedIndexInformer
	instanceTypeInformer        cache.SharedIndexInformer
	clusterPreferenceInformer   cache.SharedIndexInformer
	preferenceInformer          cache.SharedIndexInformer
	vmiMigrationInformer        cache.SharedIndexInformer
	clusterConfig               *virtconfig.ClusterConfig
)

func SetupMetrics(
	vmi cache.SharedIndexInformer,
	clusterInstanceType cache.SharedIndexInformer,
	instanceType cache.SharedIndexInformer,
	clusterPreference cache.SharedIndexInformer,
	preference cache.SharedIndexInformer,
	vmiMigration cache.SharedIndexInformer,
	virtClusterConfig *virtconfig.ClusterConfig,
) error {
	vmiInformer = vmi
	clusterInstanceTypeInformer = clusterInstanceType
	instanceTypeInformer = instanceType
	clusterPreferenceInformer = clusterPreference
	preferenceInformer = preference
	vmiMigrationInformer = vmiMigration
	clusterConfig = virtClusterConfig

	if err := operatormetrics.RegisterMetrics(metrics...); err != nil {
		return err
	}

	return operatormetrics.RegisterCollector(
		migrationStatsCollector,
		vmiStatsCollector,
	)
}

func UpdateVMIMigrationInformer(informer cache.SharedIndexInformer) {
	vmiMigrationInformer = informer
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}
