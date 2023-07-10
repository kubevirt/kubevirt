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
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	none  = "<none>"
	other = "<other>"
)

var (

	// Prefixes used when transforming K8s metadata into metric labels
	annotationPrefix = "vm.kubevirt.io/"

	// higher-level, telemetry-friendly metrics
	vmiCountDesc = prometheus.NewDesc(
		"kubevirt_vmi_phase_count",
		"VMI phase.",
		[]string{
			"node", "phase", "os", "workload", "flavor", "instance_type", "preference",
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

	instancetypeVendorLabel = "instancetype.kubevirt.io/vendor"

	// vendors whose instance types are whitelisted for telemetry
	whitelistedInstanceTypeVendors = map[string]bool{
		"kubevirt.io": true,
		"redhat.com":  true,
	}
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

type VMICollector struct {
	vmiInformer                 cache.SharedIndexInformer
	clusterInstanceTypeInformer cache.SharedIndexInformer
	instanceTypeInformer        cache.SharedIndexInformer
	clusterPreferenceInformer   cache.SharedIndexInformer
	preferenceInformer          cache.SharedIndexInformer
	clusterConfig               *virtconfig.ClusterConfig
}

func (co *VMICollector) Describe(_ chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
}

// does VMI informer stuff
func SetupVMICollector(
	vmiInformer cache.SharedIndexInformer,
	clusterInstanceTypeInformer cache.SharedIndexInformer, instanceTypeInformer cache.SharedIndexInformer,
	clusterPreferenceInformer cache.SharedIndexInformer, preferenceInformer cache.SharedIndexInformer,
	clusterConfig *virtconfig.ClusterConfig,
) {

	log.Log.Infof("Starting vmi collector")
	co := &VMICollector{
		vmiInformer:                 vmiInformer,
		clusterInstanceTypeInformer: clusterInstanceTypeInformer,
		instanceTypeInformer:        instanceTypeInformer,
		clusterPreferenceInformer:   clusterPreferenceInformer,
		preferenceInformer:          preferenceInformer,
		clusterConfig:               clusterConfig,
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

	co.updateVMIsPhase(vmis, ch)
	co.updateVMIMetrics(vmis, ch)
	return
}

func (co *VMICollector) UpdateFromAnnotations(vmc *vmiCountMetric, annotations map[string]string) {
	if val, ok := annotations[annotationPrefix+"os"]; ok {
		vmc.OS = val
	}

	if val, ok := annotations[annotationPrefix+"workload"]; ok {
		vmc.Workload = val
	}

	if val, ok := annotations[annotationPrefix+"flavor"]; ok {
		vmc.Flavor = val
	}

	co.setInstancetypeFromAnnotations(vmc, annotations)
	co.setPreferenceFromAnnotations(vmc, annotations)
}

func (co *VMICollector) setInstancetypeFromAnnotations(vmc *vmiCountMetric, annotations map[string]string) {
	if instancetypeName, ok := annotations[k6tv1.InstancetypeAnnotation]; ok {
		vmc.InstanceType = other

		obj, ok, err := co.instanceTypeInformer.GetIndexer().GetByKey(instancetypeName)
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

		obj, ok, err := co.clusterInstanceTypeInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachineClusterInstancetype).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			vmc.InstanceType = instancetypeName
		}
	}
}

func (co *VMICollector) setPreferenceFromAnnotations(vmc *vmiCountMetric, annotations map[string]string) {
	if instancetypeName, ok := annotations[k6tv1.PreferenceAnnotation]; ok {
		vmc.Preference = other

		obj, ok, err := co.preferenceInformer.GetIndexer().GetByKey(instancetypeName)
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

		obj, ok, err := co.clusterPreferenceInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachineClusterPreference).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			vmc.Preference = instancetypeName
		}
	}
}

func (co *VMICollector) newVMICountMetric(vmi *k6tv1.VirtualMachineInstance) vmiCountMetric {
	vmc := vmiCountMetric{
		Phase:        strings.ToLower(string(vmi.Status.Phase)),
		OS:           none,
		Workload:     none,
		Flavor:       none,
		InstanceType: none,
		Preference:   none,
		NodeName:     vmi.Status.NodeName,
	}

	co.UpdateFromAnnotations(&vmc, vmi.Annotations)

	return vmc
}

func (co *VMICollector) makeVMICountMetricMap(vmis []*k6tv1.VirtualMachineInstance) map[vmiCountMetric]uint64 {
	countMap := make(map[vmiCountMetric]uint64)

	for _, vmi := range vmis {
		vmc := co.newVMICountMetric(vmi)
		countMap[vmc]++
	}
	return countMap
}

func (co *VMICollector) updateVMIsPhase(vmis []*k6tv1.VirtualMachineInstance, ch chan<- prometheus.Metric) {
	log.Log.V(1).Infof("Updating VMIs phase metrics")
	countMap := co.makeVMICountMetricMap(vmis)
	log.Log.V(1).Infof("phase %+v", countMap)

	for vmc, count := range countMap {
		mv, err := prometheus.NewConstMetric(
			vmiCountDesc, prometheus.GaugeValue,
			float64(count),
			vmc.NodeName, vmc.Phase, vmc.OS, vmc.Workload, vmc.Flavor, vmc.InstanceType, vmc.Preference,
		)
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to create metric for VMIs phase")
			continue
		}
		ch <- mv
	}
}

func checkNonEvictableVMAndSetMetric(clusterConfig *virtconfig.ClusterConfig, vmi *k6tv1.VirtualMachineInstance) float64 {
	setVal := 0.0
	if migrations.VMIMigratableOnEviction(clusterConfig, vmi) {
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

func (co *VMICollector) updateVMIMetrics(vmis []*k6tv1.VirtualMachineInstance, ch chan<- prometheus.Metric) {
	for _, vmi := range vmis {
		mv, err := prometheus.NewConstMetric(
			vmiEvictionBlockerDesc, prometheus.GaugeValue,
			checkNonEvictableVMAndSetMetric(co.clusterConfig, vmi),
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
		)
		if err != nil {
			continue
		}
		ch <- mv
	}
}
