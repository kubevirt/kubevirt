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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package migration

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/prometheus/client_golang/prometheus"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	transTimeErrFmt = "Error encountered during vmi migration transition time histogram calculation: %v"
	transTimeFail   = "Failed to get a histogram for a vmi migration lifecycle transition times"
)

func getTransitionTimeSeconds(newVMIMigration *v1.VirtualMachineInstanceMigration) (float64, error) {
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

func phaseTransitionTimeBuckets() []float64 {
	return []float64{
		(0.1 * time.Second.Seconds()),
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
		(2 * time.Hour).Seconds(),
	}
}

func updateVMIMigrationPhaseTransitionTimeFromCreationTimeHistogramVec(histogramVec *prometheus.HistogramVec, oldVMIMigration *v1.VirtualMachineInstanceMigration, newVMIMigration *v1.VirtualMachineInstanceMigration) {
	if oldVMIMigration == nil || oldVMIMigration.Status.Phase == newVMIMigration.Status.Phase {
		return
	}

	diffSeconds, err := getTransitionTimeSeconds(newVMIMigration)
	if err != nil {
		log.Log.V(4).Infof(transTimeErrFmt, err)
		return
	}

	labels := []string{string(newVMIMigration.Status.Phase)}
	histogram, err := histogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(transTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func newVMIMigrationPhaseTransitionTimeFromCreationHistogramVec(informer cache.SharedIndexInformer) *prometheus.HistogramVec {
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubevirt_vmi_migration_phase_transition_time_from_creation_seconds",
			Help:    "Histogram of VM migration phase transitions duration from creation time in seconds",
			Buckets: phaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi migration
			"phase",
		},
	)

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMIMigration, newVMIMigration interface{}) {
			updateVMIMigrationPhaseTransitionTimeFromCreationTimeHistogramVec(histogramVec, oldVMIMigration.(*v1.VirtualMachineInstanceMigration), newVMIMigration.(*v1.VirtualMachineInstanceMigration))
		},
	})
	if err != nil {
		panic(err)
	}
	return histogramVec
}
