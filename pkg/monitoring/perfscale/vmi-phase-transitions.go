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

package perfscale

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
	transTimeErrFmt = "Error encountered during vmi transition time histogram calculation: %v"
	transTimeFail   = "Failed to get a histogram for a vmi lifecycle transition times"
)

func getTransitionTimeSeconds(fromCreation bool, fromDeletion bool, oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) (float64, error) {
	var oldTime *metav1.Time
	var newTime *metav1.Time

	if fromCreation || oldVMI == nil || (oldVMI.Status.Phase == v1.VmPhaseUnset) {
		oldTime = newVMI.CreationTimestamp.DeepCopy()
	} else if fromDeletion && newVMI.IsMarkedForDeletion() {
		oldTime = newVMI.DeletionTimestamp.DeepCopy()
	} else if fromDeletion && !newVMI.IsMarkedForDeletion() {
		return 0.0, fmt.Errorf("missing deletion timestamp")
	}

	for _, transitionTimestamp := range newVMI.Status.PhaseTransitionTimestamps {
		if newTime == nil && transitionTimestamp.Phase == newVMI.Status.Phase {
			newTime = transitionTimestamp.PhaseTransitionTimestamp.DeepCopy()
		} else if oldTime == nil && oldVMI != nil && transitionTimestamp.Phase == oldVMI.Status.Phase {
			oldTime = transitionTimestamp.PhaseTransitionTimestamp.DeepCopy()
		} else if oldTime != nil && newTime != nil {
			break
		}
	}

	if newTime == nil || oldTime == nil {
		// no phase transition timestamp found
		return 0.0, fmt.Errorf("missing phase transition timestamp")
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

func updateVMIPhaseTransitionTimeHistogramVec(histogramVec *prometheus.HistogramVec, oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) {
	if oldVMI == nil || oldVMI.Status.Phase == newVMI.Status.Phase {
		return
	}

	diffSeconds, err := getTransitionTimeSeconds(false, false, oldVMI, newVMI)
	if err != nil {
		log.Log.V(4).Infof(transTimeErrFmt, err)
		return
	}

	labels := []string{string(newVMI.Status.Phase), string(oldVMI.Status.Phase)}
	histogram, err := histogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(transTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func newVMIPhaseTransitionTimeHistogramVec(informer cache.SharedIndexInformer) *prometheus.HistogramVec {
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubevirt_vmi_phase_transition_time_seconds",
			Help:    "Histogram of VM phase transitions duration between different phases in seconds",
			Buckets: phaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi
			"phase",
			// last phase of the vmi
			"last_phase",
		},
	)

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMI, newVMI interface{}) {
			updateVMIPhaseTransitionTimeHistogramVec(histogramVec, oldVMI.(*v1.VirtualMachineInstance), newVMI.(*v1.VirtualMachineInstance))
		},
	})
	if err != nil {
		panic(err)
	}
	return histogramVec
}

func updateVMIPhaseTransitionTimeFromCreationTimeHistogramVec(histogramVec *prometheus.HistogramVec, oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) {
	if oldVMI == nil || oldVMI.Status.Phase == newVMI.Status.Phase {
		return
	}

	diffSeconds, err := getTransitionTimeSeconds(true, false, oldVMI, newVMI)
	if err != nil {
		log.Log.V(4).Infof(transTimeErrFmt, err)
		return
	}

	labels := []string{string(newVMI.Status.Phase)}
	histogram, err := histogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(transTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func updateVMIPhaseTransitionTimeFromDeletionTimeHistogramVec(histogramVec *prometheus.HistogramVec, oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) {
	if !newVMI.IsMarkedForDeletion() || !newVMI.IsFinal() {
		return
	}

	if oldVMI == nil || oldVMI.Status.Phase == newVMI.Status.Phase {
		return
	}

	diffSeconds, err := getTransitionTimeSeconds(false, true, oldVMI, newVMI)
	if err != nil {
		log.Log.V(4).Infof(transTimeErrFmt, err)
		return
	}

	labels := []string{string(newVMI.Status.Phase)}
	histogram, err := histogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(transTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func newVMIPhaseTransitionTimeFromCreationHistogramVec(informer cache.SharedIndexInformer) *prometheus.HistogramVec {
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubevirt_vmi_phase_transition_time_from_creation_seconds",
			Help:    "Histogram of VM phase transitions duration from creation time in seconds",
			Buckets: phaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi
			"phase",
		},
	)

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMI, newVMI interface{}) {
			updateVMIPhaseTransitionTimeFromCreationTimeHistogramVec(histogramVec, oldVMI.(*v1.VirtualMachineInstance), newVMI.(*v1.VirtualMachineInstance))
		},
	})
	if err != nil {
		panic(err)
	}
	return histogramVec
}

func newVMIPhaseTransitionTimeFromDeletionHistogramVec(informer cache.SharedIndexInformer) *prometheus.HistogramVec {
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubevirt_vmi_phase_transition_time_from_deletion_seconds",
			Help:    "Histogram of VM phase transitions duration from deletion time in seconds",
			Buckets: phaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi
			"phase",
		},
	)

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMI, newVMI interface{}) {
			// User is deleting a VM. Record the time from the
			// deletionTimestamp to when the VMI enters the final phase
			updateVMIPhaseTransitionTimeFromDeletionTimeHistogramVec(histogramVec, oldVMI.(*v1.VirtualMachineInstance), newVMI.(*v1.VirtualMachineInstance))
		},
	})
	if err != nil {
		panic(err)
	}
	return histogramVec
}
