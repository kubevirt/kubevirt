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
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"

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
			"node", "phase", "os", "workload", "flavor", "instance_type",
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

	instancetypeVendorLabel        = "instancetype.kubevirt.io/vendor"
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
	NodeName     string
}

type VMICollector struct {
	vmiInformer                 cache.SharedIndexInformer
	clusterInstanceTypeInformer cache.SharedIndexInformer
	instanceTypeInformer        cache.SharedIndexInformer
	clusterConfig               *virtconfig.ClusterConfig
}

type vmisInstanceTypes struct {
	clusterInstanceTypes []*instancetypev1alpha2.VirtualMachineClusterInstancetype
	instanceTypes        []*instancetypev1alpha2.VirtualMachineInstancetype
}

func (co *VMICollector) Describe(_ chan<- *prometheus.Desc) {
	// TODO: Use DescribeByCollect?
}

// does VMI informer stuff
func SetupVMICollector(vmiInformer cache.SharedIndexInformer, clusterInstanceTypeInformer cache.SharedIndexInformer, instanceTypeInformer cache.SharedIndexInformer, clusterConfig *virtconfig.ClusterConfig) {
	log.Log.Infof("Starting vmi collector")
	co := &VMICollector{
		vmiInformer:                 vmiInformer,
		clusterInstanceTypeInformer: clusterInstanceTypeInformer,
		instanceTypeInformer:        instanceTypeInformer,
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

	updateVMIsPhase(vmis, co.buildVmisInstanceTypes(), ch)
	co.updateVMIMetrics(vmis, ch)
	return
}

func (co *VMICollector) buildVmisInstanceTypes() *vmisInstanceTypes {
	cachedInstanceTypes := co.instanceTypeInformer.GetIndexer().List()
	instanceTypes := make([]*instancetypev1alpha2.VirtualMachineInstancetype, len(cachedInstanceTypes))
	for i, obj := range cachedInstanceTypes {
		instanceTypes[i] = obj.(*instancetypev1alpha2.VirtualMachineInstancetype)
	}

	cachedClusterInstanceTypes := co.clusterInstanceTypeInformer.GetIndexer().List()
	clusterInstanceTypes := make([]*instancetypev1alpha2.VirtualMachineClusterInstancetype, len(cachedClusterInstanceTypes))
	for i, obj := range cachedClusterInstanceTypes {
		clusterInstanceTypes[i] = obj.(*instancetypev1alpha2.VirtualMachineClusterInstancetype)
	}

	return &vmisInstanceTypes{
		clusterInstanceTypes: clusterInstanceTypes,
		instanceTypes:        instanceTypes,
	}
}

func (vmc *vmiCountMetric) UpdateFromAnnotations(annotations map[string]string, instanceTypes *vmisInstanceTypes) {
	if val, ok := annotations[annotationPrefix+"os"]; ok {
		vmc.OS = val
	}

	if val, ok := annotations[annotationPrefix+"workload"]; ok {
		vmc.Workload = val
	}

	if val, ok := annotations[annotationPrefix+"flavor"]; ok {
		vmc.Flavor = val
	}

	if val, ok := annotations[k6tv1.InstancetypeAnnotation]; ok {
		vmc.InstanceType = other

		for _, it := range instanceTypes.instanceTypes {
			if it.Name == val {
				vendor := it.Labels[instancetypeVendorLabel]
				if _, isWhitelisted := whitelistedInstanceTypeVendors[vendor]; isWhitelisted {
					vmc.InstanceType = val
					break
				}
			}
		}
	}

	if val, ok := annotations[k6tv1.ClusterInstancetypeAnnotation]; ok {
		vmc.InstanceType = other

		for _, it := range instanceTypes.clusterInstanceTypes {
			if it.Name == val {
				vendor := it.Labels[instancetypeVendorLabel]
				if _, isWhitelisted := whitelistedInstanceTypeVendors[vendor]; isWhitelisted {
					vmc.InstanceType = val
					break
				}
			}
		}
	}
}

func newVMICountMetric(vmi *k6tv1.VirtualMachineInstance, instanceTypes *vmisInstanceTypes) vmiCountMetric {
	vmc := vmiCountMetric{
		Phase:        strings.ToLower(string(vmi.Status.Phase)),
		OS:           none,
		Workload:     none,
		Flavor:       none,
		InstanceType: none,
		NodeName:     vmi.Status.NodeName,
	}
	vmc.UpdateFromAnnotations(vmi.Annotations, instanceTypes)
	return vmc
}

func makeVMICountMetricMap(vmis []*k6tv1.VirtualMachineInstance, instanceTypes *vmisInstanceTypes) map[vmiCountMetric]uint64 {
	countMap := make(map[vmiCountMetric]uint64)

	for _, vmi := range vmis {
		vmc := newVMICountMetric(vmi, instanceTypes)
		countMap[vmc]++
	}
	return countMap
}

func updateVMIsPhase(vmis []*k6tv1.VirtualMachineInstance, instanceTypes *vmisInstanceTypes, ch chan<- prometheus.Metric) {
	log.Log.V(1).Infof("Updating VMIs phase metrics")
	countMap := makeVMICountMetricMap(vmis, instanceTypes)
	log.Log.V(1).Infof("phase %+v", countMap)

	for vmc, count := range countMap {
		mv, err := prometheus.NewConstMetric(
			vmiCountDesc, prometheus.GaugeValue,
			float64(count),
			vmc.NodeName, vmc.Phase, vmc.OS, vmc.Workload, vmc.Flavor, vmc.InstanceType,
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
