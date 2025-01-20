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
 * Copyright the KubeVirt Authors.
 *
 */

package virt_controller

import (
	"fmt"
	"time"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/client"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/common/workqueue"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var (
	metrics = [][]operatormetrics.Metric{
		componentMetrics,
		migrationMetrics,
		perfscaleMetrics,
		vmiMetrics,
		vmSnapshotMetrics,
	}

	vmInformer                    cache.SharedIndexInformer
	vmiInformer                   cache.SharedIndexInformer
	persistentVolumeClaimInformer cache.SharedIndexInformer
	vmiMigrationInformer          cache.SharedIndexInformer
	kvPodInformer                 cache.SharedIndexInformer
	clusterConfig                 *virtconfig.ClusterConfig
	instancetypeMethods           *instancetype.InstancetypeMethods
)

func SetupMetrics(
	vm cache.SharedIndexInformer,
	vmi cache.SharedIndexInformer,
	pvc cache.SharedIndexInformer,
	vmiMigration cache.SharedIndexInformer,
	pod cache.SharedIndexInformer,
	virtClusterConfig *virtconfig.ClusterConfig,
	methods *instancetype.InstancetypeMethods,
) error {
	vmInformer = vm
	vmiInformer = vmi
	persistentVolumeClaimInformer = pvc
	vmiMigrationInformer = vmiMigration
	kvPodInformer = pod
	clusterConfig = virtClusterConfig
	instancetypeMethods = methods

	if err := client.SetupMetrics(); err != nil {
		return err
	}

	if err := workqueue.SetupMetrics(); err != nil {
		return err
	}

	if err := operatormetrics.RegisterMetrics(metrics...); err != nil {
		return err
	}

	return operatormetrics.RegisterCollector(
		migrationStatsCollector,
		vmiStatsCollector,
		vmStatsCollector,
	)
}

func RegisterLeaderMetrics() error {
	if err := operatormetrics.RegisterMetrics(leaderMetrics); err != nil {
		return err
	}

	return nil
}

func UpdateVMIMigrationInformer(informer cache.SharedIndexInformer) {
	vmiMigrationInformer = informer
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}

func PhaseTransitionTimeBuckets() []float64 {
	return []float64{
		0.5 * time.Second.Seconds(),
		1 * time.Second.Seconds(),
		2 * time.Second.Seconds(),
		5 * time.Second.Seconds(),
		10 * time.Second.Seconds(),
		20 * time.Second.Seconds(),
		30 * time.Second.Seconds(),
		40 * time.Second.Seconds(),
		50 * time.Second.Seconds(),
		60 * time.Second.Seconds(),
		90 * time.Second.Seconds(),
		2 * time.Minute.Seconds(),
		3 * time.Minute.Seconds(),
		5 * time.Minute.Seconds(),
		10 * time.Minute.Seconds(),
		20 * time.Minute.Seconds(),
		30 * time.Minute.Seconds(),
		1 * time.Hour.Seconds(),
	}
}

func getTransitionTimeSeconds(oldTime *metav1.Time, newTime *metav1.Time) (float64, error) {
	if newTime == nil || oldTime == nil {
		// no phase transition timestamp found
		return 0.0, fmt.Errorf("missing phase transition timestamp, newTime: %v, oldTime: %v", newTime, oldTime)
	}

	diffSeconds := newTime.Time.Sub(oldTime.Time).Seconds()

	// when transitions are very fast, we can encounter time skew. Make 0 the floor
	if diffSeconds < 0 {
		diffSeconds = 0.0
	}

	return diffSeconds, nil
}
