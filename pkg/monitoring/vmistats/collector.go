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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package vmistats

import (
	"fmt"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/prometheus/client_golang/prometheus"

	k6tv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

var (

	// Preffixes used when transforming K8s metadata into metric labels
	annotationPrefix = "vm.kubevirt.io/"

	// higher-level, telemetry-friendly metrics
	vmiCountDesc = prometheus.NewDesc(
		"kubevirt_vmi_phase_count",
		"VMI phase.",
		[]string{
			"node", "phase", "os", "workload", "flavor",
		},
		nil,
	)

	vmiEvictionBlockerDesc = prometheus.NewDesc(
		"kubevirt_vmi_non_evictable",
		"Indication for a VirtualMachine that its eviction strategy is set to Live Migration but is not migratable.",
		[]string{
			"node", "namespace", "name",
		},
		nil,
	)
)

type vmiCountMetric struct {
	Phase    string
	OS       string
	Workload string
	Flavor   string
	NodeName string
}

type VMICollector struct {
	vmiInformer cache.SharedIndexInformer
}

func (co *VMICollector) Describe(_ chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
}

// does VMI informer stuff
func SetupVMICollector(vmiInformer cache.SharedIndexInformer) {
	log.Log.Infof("Starting vmi collector")
	co := &VMICollector{
		vmiInformer: vmiInformer,
	}

	prometheus.MustRegister(newVMIPhaseTransitionTimeHistogramVec(vmiInformer))
	prometheus.MustRegister(newVMIPhaseTransitionTimeFromCreationHistogramVec(vmiInformer))
	prometheus.MustRegister(co)
}

// Note that Collect could be called concurrently
func (co *VMICollector) Collect(ch chan<- prometheus.Metric) {
	cachedObjs := co.vmiInformer.GetIndexer().List()
	if len(cachedObjs) == 0 {
		log.Log.V(4).Infof("No VMIs detected")
		return
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	updateVMIsPhase(vmis, ch)
	updateVMIMetrics(vmis, ch)
	return
}

func (vmc *vmiCountMetric) UpdateFromAnnotations(annotations map[string]string) {
	if val, ok := annotations[annotationPrefix+"os"]; ok {
		vmc.OS = val
	}

	if val, ok := annotations[annotationPrefix+"workload"]; ok {
		vmc.Workload = val
	}

	if val, ok := annotations[annotationPrefix+"flavor"]; ok {
		vmc.Flavor = val
	}
}

func newVMICountMetric(vmi *k6tv1.VirtualMachineInstance) vmiCountMetric {
	vmc := vmiCountMetric{
		Phase:    strings.ToLower(string(vmi.Status.Phase)),
		OS:       "<none>",
		Workload: "<none>",
		Flavor:   "<none>",
		NodeName: vmi.Status.NodeName,
	}
	vmc.UpdateFromAnnotations(vmi.Annotations)
	return vmc
}

func makeVMICountMetricMap(vmis []*k6tv1.VirtualMachineInstance) map[vmiCountMetric]uint64 {
	countMap := make(map[vmiCountMetric]uint64)

	for _, vmi := range vmis {
		vmc := newVMICountMetric(vmi)
		countMap[vmc]++
	}
	return countMap
}

func updateVMIsPhase(vmis []*k6tv1.VirtualMachineInstance, ch chan<- prometheus.Metric) {
	countMap := makeVMICountMetricMap(vmis)

	for vmc, count := range countMap {
		mv, err := prometheus.NewConstMetric(
			vmiCountDesc, prometheus.GaugeValue,
			float64(count),
			vmc.NodeName, vmc.Phase, vmc.OS, vmc.Workload, vmc.Flavor,
		)
		if err != nil {
			continue
		}
		ch <- mv
	}
}

func checkNonEvictableVMAndSetMetric(vmi *k6tv1.VirtualMachineInstance) float64 {
	setVal := 0.0
	if vmi.IsEvictable() {
		vmiIsMigratableCond := controller.NewVirtualMachineInstanceConditionManager().
			GetCondition(vmi, k6tv1.VirtualMachineInstanceIsMigratable)

		//As this metric is used for user alert we refer to be conservative - so if the VirtualMachineInstanceIsMigratable
		//condition is still not set we treat the VM as if it's "not migratable"
		if vmiIsMigratableCond == nil || vmiIsMigratableCond.Status == k8sv1.ConditionFalse {
			setVal = 1.0
		}

	}
	return setVal
}

func updateVMIMetrics(vmis []*k6tv1.VirtualMachineInstance, ch chan<- prometheus.Metric) {
	for _, vmi := range vmis {
		updateVMIEvictionBlocker(vmi, ch)
	}
}

func updateVMIEvictionBlocker(vmi *k6tv1.VirtualMachineInstance, ch chan<- prometheus.Metric) {
	mv, err := prometheus.NewConstMetric(
		vmiEvictionBlockerDesc, prometheus.GaugeValue,
		checkNonEvictableVMAndSetMetric(vmi),
		vmi.Status.NodeName, vmi.Namespace, vmi.Name,
	)
	if err != nil {
		return
	}
	ch <- mv

}

func getTransitionTimeSeconds(fromCreation bool, oldVMI *k6tv1.VirtualMachineInstance, newVMI *k6tv1.VirtualMachineInstance) (float64, error) {

	var oldTime *metav1.Time
	var newTime *metav1.Time

	if fromCreation || oldVMI == nil || (oldVMI.Status.Phase == k6tv1.VmPhaseUnset) {
		oldTime = newVMI.CreationTimestamp.DeepCopy()
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
	}
}

func updateVMIPhaseTransitionTimeHistogramVec(histogramVec *prometheus.HistogramVec, oldVMI *k6tv1.VirtualMachineInstance, newVMI *k6tv1.VirtualMachineInstance) {
	if oldVMI == nil || oldVMI.Status.Phase == newVMI.Status.Phase {
		return
	}
	diffSeconds, err := getTransitionTimeSeconds(false, oldVMI, newVMI)
	if err != nil {
		log.Log.V(4).Infof("Error encountered during vmi transition time histogram calculation: %v", err)
		return
	}

	labels := []string{string(newVMI.Status.Phase), string(oldVMI.Status.Phase)}
	histogram, err := histogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error("Failed to get a histogram for a vmi lifecycle transition times")
		return
	}

	histogram.Observe(diffSeconds)
}

func newVMIPhaseTransitionTimeHistogramVec(informer cache.SharedIndexInformer) *prometheus.HistogramVec {
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubevirt_vmi_phase_transition_time_seconds",
			Buckets: phaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi
			"phase",
			// last phase of the vmi
			"last_phase",
		},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMI, newVMI interface{}) {
			updateVMIPhaseTransitionTimeHistogramVec(histogramVec, oldVMI.(*k6tv1.VirtualMachineInstance), newVMI.(*k6tv1.VirtualMachineInstance))
		},
	})
	return histogramVec
}

func updateVMIPhaseTransitionTimeFromCreationTimeHistogramVec(histogramVec *prometheus.HistogramVec, oldVMI *k6tv1.VirtualMachineInstance, newVMI *k6tv1.VirtualMachineInstance) {
	if oldVMI == nil || oldVMI.Status.Phase == newVMI.Status.Phase {
		return
	}

	diffSeconds, err := getTransitionTimeSeconds(true, oldVMI, newVMI)
	if err != nil {
		log.Log.V(4).Infof("Error encountered during vmi transition time histogram calculation: %v", err)
		return
	}

	labels := []string{string(newVMI.Status.Phase)}
	histogram, err := histogramVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		log.Log.Reason(err).Error("Failed to get a histogram for a vmi lifecycle transition times")
		return
	}

	histogram.Observe(diffSeconds)

}

func newVMIPhaseTransitionTimeFromCreationHistogramVec(informer cache.SharedIndexInformer) *prometheus.HistogramVec {
	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubevirt_vmi_phase_transition_time_from_creation_seconds",
			Buckets: phaseTransitionTimeBuckets(),
		},
		[]string{
			// phase of the vmi
			"phase",
		},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldVMI, newVMI interface{}) {
			updateVMIPhaseTransitionTimeFromCreationTimeHistogramVec(histogramVec, oldVMI.(*k6tv1.VirtualMachineInstance), newVMI.(*k6tv1.VirtualMachineInstance))
		},
	})
	return histogramVec
}
