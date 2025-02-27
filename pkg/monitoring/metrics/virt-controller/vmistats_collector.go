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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"

	k6tv1 "kubevirt.io/api/core/v1"

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
		[]string{"node", "phase", "os", "workload", "flavor", "instance_type", "preference", "guest_os_kernel_release", "guest_os_machine", "guest_os_name", "guest_os_version_id"},
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
	Phase                string
	OS                   string
	Workload             string
	Flavor               string
	InstanceType         string
	Preference           string
	NodeName             string
	GuestOSKernelRelease string
	GuestOSMachine       string
	GuestOSName          string
	GuestOSVersionID     string
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
			Labels: []string{vmc.NodeName, vmc.Phase, vmc.OS, vmc.Workload, vmc.Flavor, vmc.InstanceType, vmc.Preference,
				vmc.GuestOSKernelRelease, vmc.GuestOSMachine, vmc.GuestOSName, vmc.GuestOSVersionID},
			Value: float64(count),
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
		Phase:                strings.ToLower(string(vmi.Status.Phase)),
		OS:                   none,
		Workload:             none,
		Flavor:               none,
		InstanceType:         none,
		Preference:           none,
		GuestOSKernelRelease: none,
		GuestOSMachine:       none,
		GuestOSName:          none,
		GuestOSVersionID:     none,
		NodeName:             vmi.Status.NodeName,
	}

	updateFromAnnotations(&vmc, vmi)
	updateFromGuestOSInfo(&vmc, vmi.Status.GuestOSInfo)

	return vmc
}

func updateFromAnnotations(vmc *vmiCountMetric, vmi *k6tv1.VirtualMachineInstance) {
	if val, ok := vmi.Annotations[annotationPrefix+"os"]; ok {
		vmc.OS = val
	}

	if val, ok := vmi.Annotations[annotationPrefix+"workload"]; ok {
		vmc.Workload = val
	}

	if val, ok := vmi.Annotations[annotationPrefix+"flavor"]; ok {
		vmc.Flavor = val
	}

	vmc.InstanceType = getVMIInstancetype(vmi)
	vmc.Preference = getVMIPreference(vmi)
}

func getVMIInstancetype(vmi *k6tv1.VirtualMachineInstance) string {
	if instancetypeName, ok := vmi.Annotations[k6tv1.InstancetypeAnnotation]; ok {
		key := types.NamespacedName{
			Namespace: vmi.Namespace,
			Name:      instancetypeName,
		}
		return fetchResourceName(key.String(), instancetypeMethods.InstancetypeStore)
	}

	if clusterInstancetypeName, ok := vmi.Annotations[k6tv1.ClusterInstancetypeAnnotation]; ok {
		return fetchResourceName(clusterInstancetypeName, instancetypeMethods.ClusterInstancetypeStore)
	}

	return none
}

func getVMIPreference(vmi *k6tv1.VirtualMachineInstance) string {
	if preferenceName, ok := vmi.Annotations[k6tv1.PreferenceAnnotation]; ok {
		key := types.NamespacedName{
			Namespace: vmi.Namespace,
			Name:      preferenceName,
		}
		return fetchResourceName(key.String(), instancetypeMethods.PreferenceStore)
	}

	if clusterPreferenceName, ok := vmi.Annotations[k6tv1.ClusterPreferenceAnnotation]; ok {
		return fetchResourceName(clusterPreferenceName, instancetypeMethods.ClusterPreferenceStore)
	}

	return none
}

func updateFromGuestOSInfo(vmc *vmiCountMetric, guestOSInfo k6tv1.VirtualMachineInstanceGuestOSInfo) {
	if guestOSInfo == (k6tv1.VirtualMachineInstanceGuestOSInfo{}) {
		return
	}

	if guestOSInfo.KernelRelease != "" {
		vmc.GuestOSKernelRelease = guestOSInfo.KernelRelease
	}

	if guestOSInfo.Machine != "" {
		vmc.GuestOSMachine = guestOSInfo.Machine
	}

	if guestOSInfo.Name != "" {
		vmc.GuestOSName = guestOSInfo.Name
	}

	if guestOSInfo.VersionID != "" {
		vmc.GuestOSVersionID = guestOSInfo.VersionID
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

func fetchResourceName(key string, store cache.Store) string {
	obj, ok, err := store.GetByKey(key)
	if err != nil || !ok {
		return other
	}

	apiObj, ok := obj.(v1.Object)
	if !ok {
		return other
	}

	vendorName := apiObj.GetLabels()[instancetypeVendorLabel]
	if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
		return apiObj.GetName()
	}

	return other
}
