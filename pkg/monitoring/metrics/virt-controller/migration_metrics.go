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
 */

package virt_controller

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	migrationTransTimeErrFmt = "Error encountered during VMI migration transition time histogram calculation: %v"
	migrationTransTimeFail   = "Failed to get a histogram for a VMI migration lifecycle transition times"
)

var (
	migrationMetrics = []operatormetrics.Metric{
		vmiMigrationPhaseTransitionTimeFromCreation,
	}

	vmiMigrationPhaseTransitionTimeFromCreation = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_phase_transition_time_from_creation_seconds",
			Help: "Histogram of VM migration phase transitions duration from creation time in seconds.",
		},
		prometheus.HistogramOpts{
			Buckets: PhaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi migration
			"phase",
		},
	)
)

func CreateVMIMigrationHandler(informer cache.SharedIndexInformer) error {
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMIMigration, newVMIMigration interface{}) {
			updateVMIMigrationPhaseTransitionTimeFromCreationTime(oldVMIMigration.(*v1.VirtualMachineInstanceMigration), newVMIMigration.(*v1.VirtualMachineInstanceMigration))
		},
	})

	return err
}

func updateVMIMigrationPhaseTransitionTimeFromCreationTime(oldVMIMigration *v1.VirtualMachineInstanceMigration, newVMIMigration *v1.VirtualMachineInstanceMigration) {
	if oldVMIMigration == nil || oldVMIMigration.Status.Phase == newVMIMigration.Status.Phase {
		return
	}

	diffSeconds, err := getVMIMigrationTransitionTimeSeconds(newVMIMigration)
	if err != nil {
		log.Log.V(4).Infof(migrationTransTimeErrFmt, err)
		return
	}

	labels := []string{string(newVMIMigration.Status.Phase)}
	histogram, err := vmiMigrationPhaseTransitionTimeFromCreation.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(migrationTransTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func getVMIMigrationTransitionTimeSeconds(newVMIMigration *v1.VirtualMachineInstanceMigration) (float64, error) {
	var oldTime *metav1.Time
	var newTime *metav1.Time

	oldTime = newVMIMigration.CreationTimestamp.DeepCopy()
	for _, transitionTimestamp := range newVMIMigration.Status.PhaseTransitionTimestamps {
		if transitionTimestamp.Phase == newVMIMigration.Status.Phase {
			newTime = transitionTimestamp.PhaseTransitionTimestamp.DeepCopy()
		} else if newTime != nil {
			break
		}
	}

	return getTransitionTimeSeconds(oldTime, newTime)
}
