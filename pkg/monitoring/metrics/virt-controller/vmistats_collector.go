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
	"strconv"
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
			vmiAddresses,
			vmiMigrationStartTime,
			vmiMigrationEndTime,
			vmiVnicInfo,
		},
		CollectCallback: vmiStatsCollectorCallback,
	}

	vmiInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_info",
			Help: "Information about VirtualMachineInstances.",
		},
		[]string{
			// Basic info
			"node", "namespace", "name",
			// Domain info
			"phase", "os", "workload", "flavor",
			// Instance type
			"instance_type", "preference",
			// Guest OS info
			"guest_os_kernel_release", "guest_os_machine", "guest_os_arch", "guest_os_name", "guest_os_version_id",
			// State info
			"evictable", "outdated",
			// Pod info
			"vmi_pod",
		},
	)

	vmiEvictionBlocker = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_non_evictable",
			Help: "Indication for a VirtualMachine that its eviction strategy is set to Live Migration but is not migratable.",
		},
		[]string{"node", "namespace", "name"},
	)

	vmiAddresses = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_status_addresses",
			Help: "The addresses of a VirtualMachineInstance. This metric provides the address of an available network " +
				"interface associated with the VMI in the 'address' label, and about the type of address, such as " +
				"internal IP, in the 'type' label.",
		},
		[]string{"node", "namespace", "name", "network_name", "address", "type"},
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

	vmiVnicInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_vnic_info",
			Help: "Details of VirtualMachineInstance (VMI) vNIC interfaces, such as vNIC name, binding type, " +
				"network name, and binding name for each vNIC of a running instance.",
		},
		[]string{"name", "namespace", "vnic_name", "binding_type", "network", "binding_name"},
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
	var crs []operatormetrics.CollectorResult

	for _, vmi := range vmis {
		crs = append(crs, collectVMIInfo(vmi))
		crs = append(crs, getEvictionBlocker(vmi))
		crs = append(crs, collectVMIInterfacesInfo(vmi)...)
		crs = append(crs, collectVMIMigrationTime(vmi)...)
		crs = append(crs, CollectVmisVnicInfo(vmi)...)
	}

	return crs
}

func collectVMIInfo(vmi *k6tv1.VirtualMachineInstance) operatormetrics.CollectorResult {
	os, workload, flavor := getSystemInfoFromAnnotations(vmi.Annotations)
	instanceType := getVMIInstancetype(vmi)
	preference := getVMIPreference(vmi)
	kernelRelease, guestOSMachineArch, name, versionID := getGuestOSInfo(vmi)
	guestOSMachineType := getVMIMachine(vmi)
	vmiPod := getVMIPod(vmi)

	return operatormetrics.CollectorResult{
		Metric: vmiInfo,
		Labels: []string{
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
			getVMIPhase(vmi), os, workload, flavor, instanceType, preference,
			kernelRelease, guestOSMachineType, guestOSMachineArch, name, versionID,
			strconv.FormatBool(isVMEvictable(vmi)),
			strconv.FormatBool(isVMIOutdated(vmi)),
			vmiPod,
		},
		Value: 1.0,
	}
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

func getGuestOSInfo(vmi *k6tv1.VirtualMachineInstance) (kernelRelease, guestOSMachineArch, name, versionID string) {

	if vmi.Status.GuestOSInfo == (k6tv1.VirtualMachineInstanceGuestOSInfo{}) {
		return
	}

	if vmi.Status.GuestOSInfo.KernelRelease != "" {
		kernelRelease = vmi.Status.GuestOSInfo.KernelRelease
	}

	if vmi.Status.GuestOSInfo.Machine != "" {
		guestOSMachineArch = vmi.Status.GuestOSInfo.Machine
	}

	if vmi.Status.GuestOSInfo.Name != "" {
		name = vmi.Status.GuestOSInfo.Name
	}

	if vmi.Status.GuestOSInfo.VersionID != "" {
		versionID = vmi.Status.GuestOSInfo.VersionID
	}

	return
}

func getVMIMachine(vmi *k6tv1.VirtualMachineInstance) (guestOSMachineType string) {
	if vmi.Status.Machine != nil {
		guestOSMachineType = vmi.Status.Machine.Type
	}

	return
}

func getVMIPod(vmi *k6tv1.VirtualMachineInstance) string {
	objs, err := kvPodInformer.GetIndexer().ByIndex(cache.NamespaceIndex, vmi.Namespace)
	if err != nil {
		return none
	}

	for _, obj := range objs {
		pod, ok := obj.(*k8sv1.Pod)
		if !ok {
			continue
		}

		if pod.Labels["kubevirt.io/created-by"] == string(vmi.UID) && pod.Status.Phase == k8sv1.PodRunning {
			if vmi.Status.NodeName == pod.Spec.NodeName {
				return pod.Name
			}
		}
	}

	return none
}

func getVMIInstancetype(vmi *k6tv1.VirtualMachineInstance) string {
	if instancetypeName, ok := vmi.Annotations[k6tv1.InstancetypeAnnotation]; ok {
		return fetchResourceName(instancetypeName, instancetypeMethods.InstancetypeStore)
	}

	if instancetypeName, ok := vmi.Annotations[k6tv1.ClusterInstancetypeAnnotation]; ok {
		return fetchResourceName(instancetypeName, instancetypeMethods.ClusterInstancetypeStore)
	}

	return none
}

func getVMIPreference(vmi *k6tv1.VirtualMachineInstance) string {
	if instancetypeName, ok := vmi.Annotations[k6tv1.PreferenceAnnotation]; ok {
		return fetchResourceName(instancetypeName, instancetypeMethods.PreferenceStore)
	}

	if instancetypeName, ok := vmi.Annotations[k6tv1.ClusterPreferenceAnnotation]; ok {
		return fetchResourceName(instancetypeName, instancetypeMethods.ClusterPreferenceStore)
	}

	return none
}

func fetchResourceName(name string, store cache.Store) string {
	obj, ok, err := store.GetByKey(name)
	if err != nil || !ok {
		return other
	}

	apiObj, ok := obj.(v1.Object)
	if !ok {
		return other
	}

	vendorName := apiObj.GetLabels()[instancetypeVendorLabel]
	if _, isWhitelisted := whitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
		return name
	}

	return other
}

func getEvictionBlocker(vmi *k6tv1.VirtualMachineInstance) operatormetrics.CollectorResult {
	nonEvictable := 1.0
	if isVMEvictable(vmi) {
		nonEvictable = 0.0
	}

	return operatormetrics.CollectorResult{
		Metric: vmiEvictionBlocker,
		Labels: []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name},
		Value:  nonEvictable,
	}
}

func isVMEvictable(vmi *k6tv1.VirtualMachineInstance) bool {
	if migrations.VMIMigratableOnEviction(clusterConfig, vmi) {
		vmiIsMigratableCond := controller.NewVirtualMachineInstanceConditionManager().
			GetCondition(vmi, k6tv1.VirtualMachineInstanceIsMigratable)

		// As this metric is used for user alert we refer to be conservative - so if the VirtualMachineInstanceIsMigratable
		// condition is still not set we treat the VM as if it's "not migratable"
		if vmiIsMigratableCond == nil || vmiIsMigratableCond.Status == k8sv1.ConditionFalse {
			return false
		}

	}
	return true
}

func isVMIOutdated(vmi *k6tv1.VirtualMachineInstance) bool {
	_, hasOutdatedLabel := vmi.Labels[k6tv1.OutdatedLauncherImageLabel]
	return hasOutdatedLabel
}

func collectVMIInterfacesInfo(vmi *k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	for _, iface := range vmi.Status.Interfaces {
		if cr := collectVMIInterfaceInfo(vmi, iface); cr != nil {
			crs = append(crs, *cr)
		}
	}

	return crs
}

func collectVMIInterfaceInfo(vmi *k6tv1.VirtualMachineInstance, iface k6tv1.VirtualMachineInstanceNetworkInterface) *operatormetrics.CollectorResult {
	if iface.IP == "" && iface.Name == "" {
		return nil
	}

	return &operatormetrics.CollectorResult{
		Metric: vmiAddresses,
		Labels: []string{
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
			iface.Name, iface.IP, "InternalIP",
		},
		Value: 1.0,
	}
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

func CollectVmisVnicInfo(vmi *k6tv1.VirtualMachineInstance) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult

	interfaces := vmi.Spec.Domain.Devices.Interfaces
	networks := vmi.Spec.Networks

	for _, iface := range interfaces {
		bindingType, bindingName := getBinding(iface)
		networkName, matchFound := getNetworkName(iface.Name, networks)

		if !matchFound {
			continue
		}

		results = append(results, operatormetrics.CollectorResult{
			Metric: vmiVnicInfo,
			Labels: []string{
				vmi.Name,
				vmi.Namespace,
				iface.Name,
				bindingType,
				networkName,
				bindingName,
			},
			Value: 1.0,
		})
	}

	return results
}
