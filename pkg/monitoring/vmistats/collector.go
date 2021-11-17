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
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/prometheus/client_golang/prometheus"

	k6tv1 "kubevirt.io/api/core/v1"
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
