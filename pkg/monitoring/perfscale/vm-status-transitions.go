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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/prometheus/client_golang/prometheus"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

func getVMTransitionTimeSeconds(newVM *v1.VirtualMachine) (float64, error) {
	var oldTime *metav1.Time
	var newTime *metav1.Time

	// if the vm was deleted, use the deletion timestamp as the latency baseline
	oldTime = newVM.DeletionTimestamp.DeepCopy()
	if oldTime == nil {
		oldTime = newVM.CreationTimestamp.DeepCopy()
	}
	for _, transitionTimestamp := range newVM.Status.StatusTransitionTimestamps {
		if transitionTimestamp.Status == newVM.Status.PrintableStatus {
			newTime = transitionTimestamp.StatusTransitionTimestamp.DeepCopy()
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

func updateVMStatusTransitionTimeFromCreationTimeHistogramVec(histogramVec *prometheus.HistogramVec, oldVM *v1.VirtualMachine, newVM *v1.VirtualMachine) {
	if oldVM == nil || oldVM.Status.PrintableStatus == newVM.Status.PrintableStatus {
		return
	}

	diffSeconds, err := getVMTransitionTimeSeconds(newVM)
	if err != nil {
		log.Log.V(4).Infof(transTimeErrFmt, err)
		return
	}

	labels := []string{string(newVM.Status.PrintableStatus)}
	histogram, err := histogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error(transTimeFail)
		return
	}

	histogram.Observe(diffSeconds)
}

func newVMStatusTransitionTimeFromCreationHistogramVec(informer cache.SharedIndexInformer) *prometheus.HistogramVec {
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubevirt_vm_status_transition_time_from_creation_seconds",
			Buckets: phaseTransitionTimeBuckets(),
		},
		[]string{
			// status of the vm
			"status",
		},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVM, newVM interface{}) {
			updateVMStatusTransitionTimeFromCreationTimeHistogramVec(histogramVec, oldVM.(*v1.VirtualMachine), newVM.(*v1.VirtualMachine))
		},
	})
	return histogramVec
}
