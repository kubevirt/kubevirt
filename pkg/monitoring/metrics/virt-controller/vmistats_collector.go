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
	"strings"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/client-go/log"

	k6tv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/migrations"
)

const (
	none  = "<none>"
	other = "<other>"

	annotationPrefix        = "vm.kubevirt.io/"
	instancetypeVendorLabel = "instancetype.kubevirt.io/vendor"
)

var (
	whitelistedInstanceTypeVendors = map[string]bool{
		"kubevirt.io": true,
		"redhat.com":  true,
	}

	vmiStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			vmiCount,
			vmiEvictionBlocker,
		},
		CollectCallback: vmiStatsCollectorCallback,
	}

	vmiCount = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_phase_count",
			Help: "Sum of VMIs per phase and node. `phase` can be one of the following: [`Pending`, `Scheduling`, `Scheduled`, `Running`, `Succeeded`, `Failed`, `Unknown`].",
		},
		[]string{"node", "phase", "os", "workload", "flavor", "instance_type", "preference"},
	)

	vmiEvictionBlocker = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_non_evictable",
			Help: "Indication for a VirtualMachine that its eviction strategy is set to Live Migration but is not migratable.",
		},
		[]string{"node", "namespace", "name"},
	)
)

type vmiCountMetric struct {
	Phase        string
	OS           string
	Workload     string
	Flavor       string
	InstanceType string
	Preference   string
	NodeName     string
}

func vmiStatsCollectorCallback() []operatormetrics.CollectorResult {
	cachedObjs := vmiInformer.GetIndexer().List()
	if len(cachedObjs) == 0 {
		log.Log.V(4).Infof("No VMIs detected")
		return []operatormetrics.CollectorResult{}
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))

	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	return reportVmisStats(vmis)
}

func reportVmisStats(vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	phaseResults := getVmisPhase(vmis)
	evictionBlockerResults := getEvictionBlocker(vmis)

	results := make([]operatormetrics.CollectorResult, 0, len(phaseResults)+len(evictionBlockerResults))
	results = append(results, phaseResults...)
	results = append(results, evictionBlockerResults...)
	return results
}

func getVmisPhase(vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	countMap := makeVMICountMetricMap(vmis)

	for vmc, count := range countMap {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmiCount,
			Labels: []string{vmc.NodeName, vmc.Phase, vmc.OS, vmc.Workload, vmc.Flavor, vmc.InstanceType, vmc.Preference},
			Value:  float64(count),
		})
	}

	return cr
}

func makeVMICountMetricMap(vmis []*k6tv1.VirtualMachineInstance) map[vmiCountMetric]uint64 {
	countMap := make(map[vmiCountMetric]uint64)

	for _, vmi := range vmis {
		vmc := newVMICountMetric(vmi)
		countMap[vmc]++
	}
	return countMap
}

func newVMICountMetric(vmi *k6tv1.VirtualMachineInstance) vmiCountMetric {
	vmc := vmiCountMetric{
		Phase:        strings.ToLower(string(vmi.Status.Phase)),
		OS:           none,
		Workload:     none,
		Flavor:       none,
		InstanceType: none,
		Preference:   none,
		NodeName:     vmi.Status.NodeName,
	}

	updateFromAnnotations(&vmc, vmi.Annotations)

	return vmc
}

func updateFromAnnotations(vmc *vmiCountMetric, annotations map[string]string) {
	if val, ok := annotations[annotationPrefix+"os"]; ok {
		vmc.OS = val
	}

	if val, ok := annotations[annotationPrefix+"workload"]; ok {
		vmc.Workload = val
	}

	if val, ok := annotations[annotationPrefix+"flavor"]; ok {
		vmc.Flavor = val
	}

	setInstancetypeFromAnnotations(vmc, annotations)
	setPreferenceFromAnnotations(vmc, annotations)
}

func setInstancetypeFromAnnotations(vmc *vmiCountMetric, annotations map[string]string) {
	if instancetypeName, ok := annotations[k6tv1.InstancetypeAnnotation]; ok {
		vmc.InstanceType = other

		obj, ok, err := instanceTypeInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachineInstancetype).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			vmc.InstanceType = instancetypeName
		}
	}

	if instancetypeName, ok := annotations[k6tv1.ClusterInstancetypeAnnotation]; ok {
		vmc.InstanceType = other

		obj, ok, err := clusterInstanceTypeInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachineClusterInstancetype).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			vmc.InstanceType = instancetypeName
		}
	}
}

func setPreferenceFromAnnotations(vmc *vmiCountMetric, annotations map[string]string) {
	if instancetypeName, ok := annotations[k6tv1.PreferenceAnnotation]; ok {
		vmc.Preference = other

		obj, ok, err := preferenceInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachinePreference).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			vmc.Preference = instancetypeName
		}
	}

	if instancetypeName, ok := annotations[k6tv1.ClusterPreferenceAnnotation]; ok {
		vmc.Preference = other

		obj, ok, err := clusterPreferenceInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachineClusterPreference).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			vmc.Preference = instancetypeName
		}
	}
}

func getEvictionBlocker(vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	for _, vmi := range vmis {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmiEvictionBlocker,
			Labels: []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name},
			Value:  getNonEvictableVM(vmi),
		})
	}

	return cr
}

func getNonEvictableVM(vmi *k6tv1.VirtualMachineInstance) float64 {
	setVal := 0.0
	if migrations.VMIMigratableOnEviction(clusterConfig, vmi) {
		vmiIsMigratableCond := controller.NewVirtualMachineInstanceConditionManager().
			GetCondition(vmi, k6tv1.VirtualMachineInstanceIsMigratable)

		// As this metric is used for user alert we refer to be conservative - so if the VirtualMachineInstanceIsMigratable
		// condition is still not set we treat the VM as if it's "not migratable"
		if vmiIsMigratableCond == nil || vmiIsMigratableCond.Status == k8sv1.ConditionFalse {
			setVal = 1.0
		}

	}
	return setVal
}
