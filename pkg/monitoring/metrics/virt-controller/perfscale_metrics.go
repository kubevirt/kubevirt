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
	"fmt"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	transTimeErrFmt = "Error encountered during VMI transition time histogram calculation: %v"
	transTimeFail   = "Failed to get a histogram for a VMI lifecycle transition times"
)

var (
	perfscaleMetrics = []operatormetrics.Metric{
		vmiPhaseTransition,
		vmiPhaseTransitionTimeFromCreation,
		vmiPhaseTransitionFromDeletion,
	}

	vmiPhaseTransition = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_phase_transition_time_seconds",
			Help: "Histogram of VM phase transitions duration between different phases in seconds.",
		},
		prometheus.HistogramOpts{
			Buckets: PhaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi
			"phase",
			// last phase of the vmi
			"last_phase",
		},
	)

	vmiPhaseTransitionTimeFromCreation = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_phase_transition_time_from_creation_seconds",
			Help: "Histogram of VM phase transitions duration from creation time in seconds.",
		},
		prometheus.HistogramOpts{
			Buckets: PhaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi
			"phase",
		},
	)

	vmiPhaseTransitionFromDeletion = operatormetrics.NewHistogramVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_phase_transition_time_from_deletion_seconds",
			Help: "Histogram of VM phase transitions duration from deletion time in seconds.",
		},
		prometheus.HistogramOpts{
			Buckets: PhaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi
			"phase",
		},
	)
)

func AddVMIPhaseTransitionHandlers(informer cache.SharedIndexInformer) error {
	err := addVMIPhaseTransitionHandler(informer)
	if err != nil {
		return err
	}

	err = addVMIPhaseTransitionTimeFromCreationHandler(informer)
	if err != nil {
		return err
	}

	err = addVMIPhaseTransitionTimeFromDeletionHandler(informer)
	if err != nil {
		return err
	}

	return nil
}

func addVMIPhaseTransitionHandler(informer cache.SharedIndexInformer) error {
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMI, newVMI interface{}) {
			updateVMIPhaseTransitionTime(oldVMI.(*v1.VirtualMachineInstance), newVMI.(*v1.VirtualMachineInstance))
		},
	})
	return err
}

func addVMIPhaseTransitionTimeFromCreationHandler(informer cache.SharedIndexInformer) error {
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMI, newVMI interface{}) {
			updateVMIPhaseTransitionTimeFromCreationTime(oldVMI.(*v1.VirtualMachineInstance), newVMI.(*v1.VirtualMachineInstance))
		},
	})
	return err
}

func addVMIPhaseTransitionTimeFromDeletionHandler(informer cache.SharedIndexInformer) error {
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMI, newVMI interface{}) {
			// User is deleting a VM. Record the time from the
			// deletionTimestamp to when the VMI enters the final phase
			updateVMIPhaseTransitionTimeFromDeletionTime(oldVMI.(*v1.VirtualMachineInstance), newVMI.(*v1.VirtualMachineInstance))
		},
	})
	return err
}

func updateVMIPhaseTransitionTime(oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) {
	if oldVMI == nil || oldVMI.Status.Phase == newVMI.Status.Phase {
		return
	}

	diffSeconds, err := getVMITransitionTimeSeconds(false, false, oldVMI, newVMI)
	if err != nil {
		log.Log.V(4).Infof(transTimeErrFmt, err)
		return
	}

	labels := []string{string(newVMI.Status.Phase), string(oldVMI.Status.Phase)}
	histogram, err := vmiPhaseTransition.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(transTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func updateVMIPhaseTransitionTimeFromCreationTime(oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) {
	if oldVMI == nil || oldVMI.Status.Phase == newVMI.Status.Phase {
		return
	}

	diffSeconds, err := getVMITransitionTimeSeconds(true, false, oldVMI, newVMI)
	if err != nil {
		log.Log.V(4).Infof(transTimeErrFmt, err)
		return
	}

	labels := []string{string(newVMI.Status.Phase)}
	histogram, err := vmiPhaseTransitionTimeFromCreation.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(transTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func updateVMIPhaseTransitionTimeFromDeletionTime(oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) {
	if !newVMI.IsMarkedForDeletion() || !newVMI.IsFinal() {
		return
	}

	if oldVMI == nil || oldVMI.Status.Phase == newVMI.Status.Phase {
		return
	}

	diffSeconds, err := getVMITransitionTimeSeconds(false, true, oldVMI, newVMI)
	if err != nil {
		log.Log.V(4).Infof(transTimeErrFmt, err)
		return
	}

	labels := []string{string(newVMI.Status.Phase)}
	histogram, err := vmiPhaseTransitionFromDeletion.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(transTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func getVMITransitionTimeSeconds(fromCreation bool, fromDeletion bool, oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance) (float64, error) {
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

	return getTransitionTimeSeconds(oldTime, newTime)
}
