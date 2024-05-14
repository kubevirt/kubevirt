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
			vmiInfo,
			vmiEvictionBlocker,
		},
		CollectCallback: vmiStatsCollectorCallback,
	}

	vmiInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_info",
			Help: "Information about VirtualMachineInstances.",
		},
		[]string{"node", "namespace", "name", "phase", "os", "workload", "flavor", "instance_type", "preference", "guest_os_kernel_release", "guest_os_machine", "guest_os_name", "guest_os_version_id"},
	)

	vmiEvictionBlocker = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_non_evictable",
			Help: "Indication for a VirtualMachine that its eviction strategy is set to Live Migration but is not migratable.",
		},
		[]string{"node", "namespace", "name"},
	)
)

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
	phaseResults := collectVMIInfo(vmis)
	evictionBlockerResults := getEvictionBlocker(vmis)

	results := make([]operatormetrics.CollectorResult, 0, len(phaseResults)+len(evictionBlockerResults))
	results = append(results, phaseResults...)
	results = append(results, evictionBlockerResults...)
	return results
}

func collectVMIInfo(vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	for _, vmi := range vmis {
		os, workload, flavor := getVMISystemInfo(vmi)
		instanceType := getVMIInstancetype(vmi)
		preference := getVMIPreference(vmi)
		kernelRelease, machine, name, versionID := getGuestOSInfo(vmi)

		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmiInfo,
			Labels: []string{
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				getVMIPhase(vmi), os, workload, flavor, instanceType, preference,
				kernelRelease, machine, name, versionID,
			},
			Value: 1.0,
		})
	}

	return cr
}

func getVMIPhase(vmi *k6tv1.VirtualMachineInstance) string {
	return strings.ToLower(string(vmi.Status.Phase))
}

func getVMISystemInfo(vmi *k6tv1.VirtualMachineInstance) (os, workload, flavor string) {
	os = none
	workload = none
	flavor = none

	if val, ok := vmi.Annotations[annotationPrefix+"os"]; ok {
		os = val
	}

	if val, ok := vmi.Annotations[annotationPrefix+"workload"]; ok {
		workload = val
	}

	if val, ok := vmi.Annotations[annotationPrefix+"flavor"]; ok {
		flavor = val
	}

	return
}

func getGuestOSInfo(vmi *k6tv1.VirtualMachineInstance) (kernelRelease, machine, name, versionID string) {

	if vmi.Status.GuestOSInfo == (k6tv1.VirtualMachineInstanceGuestOSInfo{}) {
		return
	}

	if vmi.Status.GuestOSInfo.KernelRelease != "" {
		kernelRelease = vmi.Status.GuestOSInfo.KernelRelease
	}

	if vmi.Status.GuestOSInfo.Machine != "" {
		machine = vmi.Status.GuestOSInfo.Machine
	}

	if vmi.Status.GuestOSInfo.Name != "" {
		name = vmi.Status.GuestOSInfo.Name
	}

	if vmi.Status.GuestOSInfo.VersionID != "" {
		versionID = vmi.Status.GuestOSInfo.VersionID
	}

	return
}

func getVMIInstancetype(vmi *k6tv1.VirtualMachineInstance) string {
	instanceType := none

	if instancetypeName, ok := vmi.Annotations[k6tv1.InstancetypeAnnotation]; ok {
		instanceType = other

		obj, ok, err := instanceTypeInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return instanceType
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachineInstancetype).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			instanceType = instancetypeName
		}
	}

	if instancetypeName, ok := vmi.Annotations[k6tv1.ClusterInstancetypeAnnotation]; ok {
		instanceType = other

		obj, ok, err := clusterInstanceTypeInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return instanceType
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachineClusterInstancetype).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			instanceType = instancetypeName
		}
	}

	return instanceType
}

func getVMIPreference(vmi *k6tv1.VirtualMachineInstance) string {
	preference := none

	if instancetypeName, ok := vmi.Annotations[k6tv1.PreferenceAnnotation]; ok {
		preference = other

		obj, ok, err := preferenceInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return preference
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachinePreference).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			preference = instancetypeName
		}
	}

	if instancetypeName, ok := vmi.Annotations[k6tv1.ClusterPreferenceAnnotation]; ok {
		preference = other

		obj, ok, err := clusterPreferenceInformer.GetIndexer().GetByKey(instancetypeName)
		if err != nil || !ok {
			return preference
		}

		vendorName := obj.(*instancetypev1beta1.VirtualMachineClusterPreference).Labels[instancetypeVendorLabel]
		if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
			preference = instancetypeName
		}
	}

	return preference
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
