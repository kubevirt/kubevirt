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
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

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
	clusterInstanceTypeInformer   cache.SharedIndexInformer
	instanceTypeInformer          cache.SharedIndexInformer
	clusterPreferenceInformer     cache.SharedIndexInformer
	preferenceInformer            cache.SharedIndexInformer
	vmiMigrationInformer          cache.SharedIndexInformer
	kvPodInformer                 cache.SharedIndexInformer
	clusterConfig                 *virtconfig.ClusterConfig

	migrationCache *VmMigrationCache
)

func SetupMetrics(
	vm cache.SharedIndexInformer,
	vmi cache.SharedIndexInformer,
	pvc cache.SharedIndexInformer,
	clusterInstanceType cache.SharedIndexInformer,
	instanceType cache.SharedIndexInformer,
	clusterPreference cache.SharedIndexInformer,
	preference cache.SharedIndexInformer,
	vmiMigration cache.SharedIndexInformer,
	pod cache.SharedIndexInformer,
	virtClusterConfig *virtconfig.ClusterConfig,
) error {
	vmInformer = vm
	vmiInformer = vmi
	persistentVolumeClaimInformer = pvc
	clusterInstanceTypeInformer = clusterInstanceType
	instanceTypeInformer = instanceType
	clusterPreferenceInformer = clusterPreference
	preferenceInformer = preference
	vmiMigrationInformer = vmiMigration
	kvPodInformer = pod
	clusterConfig = virtClusterConfig

	cacheDir := "/root/projects/github/kubevirt.io/kubevirt/data"
	cacheFilename := "vm_migration_cache.json"
	fullPathToCacheFile := filepath.Join(cacheDir, cacheFilename)

	if _, err := os.Stat(cacheDir); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(cacheDir, 0755); err != nil {
			log.Fatalf("Failed to create directory for cache: %v", err)
		}
	}

	var err error
	migrationCache, err = NewVmMigrationCache(fullPathToCacheFile)
	if err != nil {
		log.Fatalf("Failed to initialize VM migration cache: %v", err)
	}

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
		(0.5 * time.Second.Seconds()),
		(1 * time.Second.Seconds()),
		(2 * time.Second.Seconds()),
		(5 * time.Second.Seconds()),
		(10 * time.Second.Seconds()),
		(20 * time.Second.Seconds()),
		(30 * time.Second.Seconds()),
		(40 * time.Second.Seconds()),
		(50 * time.Second.Seconds()),
		(60 * time.Second).Seconds(),
		(90 * time.Second).Seconds(),
		(2 * time.Minute).Seconds(),
		(3 * time.Minute).Seconds(),
		(5 * time.Minute).Seconds(),
		(10 * time.Minute).Seconds(),
		(20 * time.Minute).Seconds(),
		(30 * time.Minute).Seconds(),
		(1 * time.Hour).Seconds(),
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
