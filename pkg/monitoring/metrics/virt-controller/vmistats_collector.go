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
			vmiInfo,
			vmiEvictionBlocker,
			vmiMigrationStartTime,
			vmiMigrationEndTime,
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

	vmiMigrationStartTime = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_start_time_seconds",
			Help: "The time at which the migration started.",
		},
		[]string{"node", "namespace", "name", "migration_name"},
	)

	vmiMigrationEndTime = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_end_time_seconds",
			Help: "The time at which the migration ended.",
		},
		[]string{"node", "namespace", "name", "migration_name", "status"},
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

	results := make([]operatormetrics.CollectorResult, 0, len(phaseResults)+len(evictionBlockerResults)+(len(vmis)*2))
	results = append(results, phaseResults...)
	results = append(results, evictionBlockerResults...)

	for _, vmi := range vmis {
		results = append(results, collectVMIMigrationTime(vmi)...)
	}

	return results
}

func collectVMIInfo(vmis []*k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	for _, vmi := range vmis {
		os, workload, flavor := getSystemInfoFromAnnotations(vmi.Annotations)
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

func getSystemInfoFromAnnotations(annotations map[string]string) (os, workload, flavor string) {
	os = none
	workload = none
	flavor = none

	if val, ok := annotations[annotationPrefix+"os"]; ok {
		os = val
	}

	if val, ok := annotations[annotationPrefix+"workload"]; ok {
		workload = val
	}

	if val, ok := annotations[annotationPrefix+"flavor"]; ok {
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

func collectVMIMigrationTime(vmi *k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult
	var migrationName string

	if vmi.Status.MigrationState == nil {
		return cr
	}

	migrationName = getMigrationNameFromMigrationUID(vmi.Namespace, vmi.Status.MigrationState.MigrationUID)

	if vmi.Status.MigrationState.StartTimestamp != nil {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmiMigrationStartTime,
			Value:  float64(vmi.Status.MigrationState.StartTimestamp.Time.Unix()),
			Labels: []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, migrationName},
		})
	}

	if vmi.Status.MigrationState.EndTimestamp != nil {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmiMigrationEndTime,
			Value:  float64(vmi.Status.MigrationState.EndTimestamp.Time.Unix()),
			Labels: []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name, migrationName,
				calculateMigrationStatus(vmi.Status.MigrationState),
			},
		})
	}

	return cr
}

func calculateMigrationStatus(migrationState *k6tv1.VirtualMachineInstanceMigrationState) string {
	if !migrationState.Completed {
		return ""
	}

	if migrationState.Failed {
		return "failed"
	}

	return "succeeded"
}

func getMigrationNameFromMigrationUID(namespace string, migrationUID types.UID) string {
	objs, err := vmiMigrationInformer.GetIndexer().ByIndex(cache.NamespaceIndex, namespace)
	if err != nil {
		return none
	}

	for _, obj := range objs {
		curMigration := obj.(*k6tv1.VirtualMachineInstanceMigration)
		if curMigration.UID != migrationUID {
			continue
		}

		return curMigration.Name
	}

	return none
}
