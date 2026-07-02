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
 * Copyright The KubeVirt Authors.
 *
 */

package services

import (
	"context"
	"fmt"
	"maps"
	"math/rand"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/openshift/library-go/pkg/build/naming"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubectl/pkg/cmd/util/podcmd"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/hypervisor"
	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/pkg/apimachinery"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/network/downwardapi"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/storage/types"
	storageutils "kubevirt.io/kubevirt/pkg/storage/utils"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/descheduler"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/pkg/vmitrait"
)

const (
	containerDisks          = "container-disks"
	hotplugDisks            = "hotplug-disks"
	hookSidecarSocks        = "hook-sidecar-sockets"
	pluginSocketsVolumeName = "kubevirt-plugin-sockets"
	pluginSocketsDir        = "/var/run/kubevirt-plugin"
	varRun                  = "/var/run"
	virtBinDir              = "virt-bin-share-dir"
	hotplugDisk             = "hotplug-disk"
	virtExporter            = "virt-exporter"
)

const K8sDevicePrefix = "devices.kubevirt.io"
const TunDevice = K8sDevicePrefix + "/tun"
const VhostNetDevice = K8sDevicePrefix + "/vhost-net"
const VhostVsockDevice = K8sDevicePrefix + "/vhost-vsock"
const PrDevice = K8sDevicePrefix + "/pr-helper"
const SevDeviceName = "sev"
const TdxDeviceName = "tdx"
const SevDevice = K8sDevicePrefix + "/" + SevDeviceName
const TdxDevice = K8sDevicePrefix + "/" + TdxDeviceName
const IOMMUFDDeviceName = "iommufd"
const IOMMUFDDevice = K8sDevicePrefix + "/" + IOMMUFDDeviceName

const debugLogs = "debugLogs"
const logVerbosity = "logVerbosity"
const virtiofsDebugLogs = "virtiofsdDebugLogs"

const qemuTimeoutJitterRange = 120

const (
	CAP_NET_BIND_SERVICE = "NET_BIND_SERVICE"
	CAP_SYS_NICE         = "SYS_NICE"
)

// LibvirtStartupDelay is added to custom liveness and readiness probes initial delay value.
// Libvirt needs roughly 10 seconds to start.
const LibvirtStartupDelay = 10

const IntelVendorName = "Intel"

const ENV_VAR_POD_NAME = "POD_NAME"

const ephemeralStorageOverheadSize = "50M"

const (
	// Default: limits.memory = 2*requests.memory
	DefaultMemoryLimitOverheadRatio = float64(2.0)

	FailedToRenderLaunchManifestErrFormat = "failed to render launch manifest: %v"
)

// Safety guards for the disjunctive cross-product merge below.
const (
	// maxNodeSelectorCrossProduct caps the size of an intermediate cross-product
	// before pruning. Checked in mergeNodeSelectorTerms before allocation.
	maxNodeSelectorCrossProduct = 1000
	// maxNodeSelectorFinalTerms caps the simplified disjunction attached to the
	// pod. Etcd's ~1.5MB object limit is the upstream constraint; this leaves
	// plenty of room for the rest of the pod spec.
	maxNodeSelectorFinalTerms = 100
)

type netMemoryCalculator interface {
	Calculate(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin) resource.Quantity
}

type annotationsGenerator interface {
	Generate(vmi *v1.VirtualMachineInstance) (map[string]string, error)
}

type targetAnnotationsGenerator interface {
	GenerateFromSource(vmi *v1.VirtualMachineInstance, sourcePod *k8sv1.Pod) (map[string]string, error)
}

type TemplateService struct {
	launcherImage              string
	exporterImage              string
	launcherQemuTimeout        int
	virtShareDir               string
	ephemeralDiskDir           string
	containerDiskDir           string
	hotplugDiskDir             string
	imagePullSecret            string
	persistentVolumeClaimStore cache.Store
	persistentVolumeStore      cache.Store
	virtClient                 kubecli.KubevirtClient
	clusterConfig              *virtconfig.ClusterConfig
	launcherSubGid             int64
	resourceQuotaStore         cache.Store
	namespaceStore             cache.Store

	sidecarCreators               []SidecarCreatorFunc
	netMemoryCalculator           netMemoryCalculator
	annotationsGenerators         []annotationsGenerator
	netTargetAnnotationsGenerator targetAnnotationsGenerator
	launcherHypervisorResources   hypervisor.LauncherHypervisorResources
}

func isFeatureStateEnabled(fs *v1.FeatureState) bool {
	return fs != nil && fs.Enabled != nil && *fs.Enabled
}

func setPersistentReservationAntiAffinity(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod, pvcStore cache.Store) error {
	prLabels, err := reservation.PersistentReservationPVCLabels(vmi, pvcStore)
	if err != nil {
		return err
	}
	if len(prLabels) == 0 {
		return nil
	}

	maps.Copy(pod.Labels, prLabels)

	terms := reservation.PersistentReservationPodAntiAffinityTerms(prLabels)
	if len(terms) == 0 {
		return nil
	}

	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &k8sv1.Affinity{}
	}
	if pod.Spec.Affinity.PodAntiAffinity == nil {
		pod.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{}
	}

	pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
		pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
		terms...,
	)

	return nil
}

func (t *TemplateService) getHotpluggedPVsForVMI(vmi *v1.VirtualMachineInstance) ([]*k8sv1.PersistentVolume, error) {
	var pvs []*k8sv1.PersistentVolume
	if !vmi.Spec.Domain.Devices.DisableHotplug {
		for _, volume := range vmi.Spec.Volumes {
			// Assume (for now it's always true) that PVC name matches DV name
			pvcName := ""
			if volume.DataVolume != nil && volume.DataVolume.Hotpluggable {
				pvcName = volume.DataVolume.Name
			} else if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.Hotpluggable {
				pvcName = volume.PersistentVolumeClaim.ClaimName
			}

			if pvcName == "" {
				continue
			}

			obj, exists, err := t.persistentVolumeClaimStore.GetByKey(vmi.Namespace + "/" + pvcName)
			if err != nil {
				return nil, err
			}
			if !exists {
				// We can't tell if the PVC doesn't exist or if it's just not in the cache yet so if we don't find it here, we just skip it
				// as there's nothing to take topology constraints from and creating the PVC after the VMI is not something we want to break
				continue
			}
			pvc, ok := obj.(*k8sv1.PersistentVolumeClaim)
			if !ok {
				return nil, fmt.Errorf("couldn't cast object to PersistentVolumeClaim: %+v", obj)
			}
			// Skip unbound PVCs (WaitForFirstConsumer) as there no topology constraints to enforce yet
			if pvc.Status.Phase != k8sv1.ClaimBound || pvc.Spec.VolumeName == "" {
				continue
			}

			pvName := pvc.Spec.VolumeName
			obj, exists, err = t.persistentVolumeStore.GetByKey(pvName)
			if err != nil {
				return nil, err
			}
			if !exists {
				// On the other hand, if the PVC exists and is Bound but we can't find the PV, it's definitely a cache timing issue so we should
				// just return an error and retry instead of skipping the PV
				return nil, fmt.Errorf("PersistentVolume %s not found in cache", pvName)
			}
			pv, ok := obj.(*k8sv1.PersistentVolume)
			if !ok {
				return nil, fmt.Errorf("couldn't cast object to PersistentVolume: %+v", obj)
			}
			pvs = append(pvs, pv)
		}
	}
	return pvs, nil
}

func (t *TemplateService) setNodeAffinityForPod(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) error {
	setNodeAffinityForHostModelCpuModel(vmi, pod)
	setNodeAffinityForbiddenFeaturePolicy(vmi, pod)

	hotpluggedVolumes, err := t.getHotpluggedPVsForVMI(vmi)
	if err != nil {
		return err
	}

	if len(hotpluggedVolumes) > 0 {
		if err := t.setNodeAffinityForHotpluggedVolumeTopology(hotpluggedVolumes, pod); err != nil {
			return err
		}
	}
	return nil
}

func setPreferredArchitectureAffinity(architecture string, pod *k8sv1.Pod) {
	if architecture == "" {
		return
	}
	preferredTerm := k8sv1.PreferredSchedulingTerm{
		Weight: 100,
		Preference: k8sv1.NodeSelectorTerm{
			MatchExpressions: []k8sv1.NodeSelectorRequirement{
				{
					Key:      k8sv1.LabelArchStable,
					Operator: k8sv1.NodeSelectorOpIn,
					Values:   []string{strings.ToLower(architecture)},
				},
			},
		},
	}
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &k8sv1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{}
	}
	pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
		pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
		preferredTerm,
	)
}

func setNodeAffinityForHostModelCpuModel(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" || vmi.Spec.Domain.CPU.Model == v1.CPUModeHostModel {
		pod.Spec.Affinity = modifyNodeAffinityToRejectLabel(pod.Spec.Affinity, v1.NodeHostModelIsObsoleteLabel)
	}
}

func setNodeAffinityForbiddenFeaturePolicy(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Features == nil {
		return
	}

	for _, feature := range vmi.Spec.Domain.CPU.Features {
		if feature.Policy == "forbid" {
			pod.Spec.Affinity = modifyNodeAffinityToRejectLabel(pod.Spec.Affinity, v1.CPUFeatureLabel+feature.Name)
		}
	}
}

func (t *TemplateService) setNodeAffinityForHotpluggedVolumeTopology(hotpluggedVolumes []*k8sv1.PersistentVolume, pod *k8sv1.Pod) error {
	var requiredNodeSelectorTerms [][]k8sv1.NodeSelectorTerm
	for _, vol := range hotpluggedVolumes {
		if vol.Spec.NodeAffinity != nil && vol.Spec.NodeAffinity.Required != nil && len(vol.Spec.NodeAffinity.Required.NodeSelectorTerms) > 0 {
			requiredNodeSelectorTerms = append(requiredNodeSelectorTerms, vol.Spec.NodeAffinity.Required.NodeSelectorTerms)
		}
	}

	if len(requiredNodeSelectorTerms) == 0 {
		return nil
	}

	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &k8sv1.Affinity{}
	}
	podAffinity := pod.Spec.Affinity

	if podAffinity.NodeAffinity == nil {
		podAffinity.NodeAffinity = &k8sv1.NodeAffinity{}
	}
	nodeAffinity := podAffinity.NodeAffinity

	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &k8sv1.NodeSelector{}
	}
	required := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution

	if len(required.NodeSelectorTerms) != 0 {
		requiredNodeSelectorTerms = append(requiredNodeSelectorTerms, required.NodeSelectorTerms)
	}

	mergedNodeSelectorTerms, err := mergeNestedNodeSelectorTerms(slices.Clone(requiredNodeSelectorTerms))
	if err != nil {
		return err
	}
	required.NodeSelectorTerms = mergedNodeSelectorTerms
	return nil
}

// Clones term and sorts In/NotIn for easy pruning of duplicate terms
func normalizeNodeSelectorTerms(term []k8sv1.NodeSelectorTerm) []k8sv1.NodeSelectorTerm {
	res := make([]k8sv1.NodeSelectorTerm, 0, len(term))
	for _, selector := range term {
		matchExpression := make([]k8sv1.NodeSelectorRequirement, 0, len(selector.MatchExpressions))
		for _, expr := range selector.MatchExpressions {
			if expr.Operator == k8sv1.NodeSelectorOpIn || expr.Operator == k8sv1.NodeSelectorOpNotIn {
				expr.Values = slices.Sorted(slices.Values(expr.Values))
			}
			matchExpression = append(matchExpression, expr)
		}
		matchFields := make([]k8sv1.NodeSelectorRequirement, 0, len(selector.MatchFields))
		for _, expr := range selector.MatchFields {
			if expr.Operator == k8sv1.NodeSelectorOpIn || expr.Operator == k8sv1.NodeSelectorOpNotIn {
				expr.Values = slices.Sorted(slices.Values(expr.Values))
			}
			matchFields = append(matchFields, expr)
		}
		res = append(res, k8sv1.NodeSelectorTerm{
			MatchExpressions: matchExpression,
			MatchFields:      matchFields,
		})
	}
	return res
}

func nodeSelectorRequirementByKeyAndOperator(reqs []k8sv1.NodeSelectorRequirement) map[string]map[k8sv1.NodeSelectorOperator][]k8sv1.NodeSelectorRequirement {
	res := make(map[string]map[k8sv1.NodeSelectorOperator][]k8sv1.NodeSelectorRequirement)
	for _, req := range reqs {
		if _, exists := res[req.Key]; !exists {
			res[req.Key] = make(map[k8sv1.NodeSelectorOperator][]k8sv1.NodeSelectorRequirement)
		}
		if _, exists := res[req.Key][req.Operator]; !exists {
			res[req.Key][req.Operator] = make([]k8sv1.NodeSelectorRequirement, 0)
		}
		res[req.Key][req.Operator] = append(res[req.Key][req.Operator], req)
	}
	return res
}

// Assume terms are normalized already and simplify. At worst returns unique keys * 4 terms
func simplifyNodeSelectorRequirements(reqs []k8sv1.NodeSelectorRequirement) ([]k8sv1.NodeSelectorRequirement, bool, error) {
	stringsToSet := func(vals []string) map[string]struct{} {
		res := make(map[string]struct{})
		for _, val := range vals {
			res[val] = struct{}{}
		}
		return res
	}

	newTerms := make([]k8sv1.NodeSelectorRequirement, 0)
	byKeyAndOp := nodeSelectorRequirementByKeyAndOperator(reqs)
	keys := slices.Sorted(maps.Keys(byKeyAndOp))
	for _, key := range keys {
		ops := byKeyAndOp[key]
		// inValues are the intersection of all matches
		var inValues map[string]struct{} = nil
		inReqs, mustBeIn := ops[k8sv1.NodeSelectorOpIn]
		for idx, req := range inReqs {
			valsSet := stringsToSet(req.Values)
			if idx == 0 {
				inValues = valsSet
				continue
			}
			for val := range inValues {
				if _, exists := valsSet[val]; !exists {
					delete(inValues, val)
				}
			}
			if len(inValues) == 0 {
				break
			}
		}

		// notInValues are the union of all matches
		var notInValues map[string]struct{} = make(map[string]struct{})
		notInReqs, mustNotBeIn := ops[k8sv1.NodeSelectorOpNotIn]
		for _, req := range notInReqs {
			for _, val := range req.Values {
				notInValues[val] = struct{}{}
				delete(inValues, val)
			}
		}

		// greater than the greatest term
		var greaterThanValue int64 = 0
		gtReqs, mustBeGreaterThan := ops[k8sv1.NodeSelectorOpGt]
		for idx, req := range gtReqs {
			if len(req.Values) == 0 {
				return nil, false, fmt.Errorf("Gt requirement on key %q has no value", key)
			}
			val, err := strconv.ParseInt(req.Values[0], 10, 64)
			if err != nil {
				return nil, false, err
			}
			if idx == 0 {
				greaterThanValue = val
				continue
			}
			if val > greaterThanValue {
				greaterThanValue = val
			}
		}
		// If In and Gt are on the the same key, we can further simplify In
		// This would likely be a bug in the user affinity
		if mustBeGreaterThan && mustBeIn {
			for val := range inValues {
				valAsInt, err := strconv.ParseInt(val, 10, 64)
				// K8S will drop non-integer constraints when Gt is applied
				if err != nil || valAsInt <= greaterThanValue {
					delete(inValues, val)
				}
			}
		}

		// less than the least term
		var lessThanValue int64 = 0
		ltReqs, mustBeLessThan := ops[k8sv1.NodeSelectorOpLt]
		for idx, req := range ltReqs {
			if len(req.Values) == 0 {
				return nil, false, fmt.Errorf("Lt requirement on key %q has no value", key)
			}
			val, err := strconv.ParseInt(req.Values[0], 10, 64)
			if err != nil {
				return nil, false, err
			}
			if idx == 0 {
				lessThanValue = val
				continue
			}
			if val < lessThanValue {
				lessThanValue = val
			}
		}
		// If In and Lt are on the the same key, we can further simplify In
		// This would likely be a bug in the user affinity
		if mustBeLessThan && mustBeIn {
			for val := range inValues {
				valAsInt, err := strconv.ParseInt(val, 10, 64)
				// K8S will drop non-integer constraints when Lt is applied
				if err != nil || valAsInt >= lessThanValue {
					delete(inValues, val)
				}
			}
		}

		// After all possible pruning of inValues, assert there is anything left
		if mustBeIn && len(inValues) == 0 {
			return nil, false, nil
		}

		// Assert that Lt and Gt constraints are satisfiable together
		if mustBeLessThan && mustBeGreaterThan && lessThanValue <= greaterThanValue+1 {
			return nil, false, nil
		}

		// At this point since In values are canonical (they encode NotIn/Gt/Lt) and are satisfiable
		// we can just drop NotIn/Gt/Lt
		if mustBeIn {
			mustNotBeIn = false
			mustBeGreaterThan = false
			mustBeLessThan = false
		}

		// In/Gt/Lt imply Exists so we don't need an explicit selector
		_, hasExists := ops[k8sv1.NodeSelectorOpExists]
		mustExist := hasExists && !mustBeGreaterThan && !mustBeLessThan && !mustBeIn

		// Nothing implies DoesNotExist
		_, mustNotExist := ops[k8sv1.NodeSelectorOpDoesNotExist]

		// All the other selectors other than mustNotBeIn need to be false for this to be satisfiable
		if mustNotExist && (mustExist || mustBeLessThan || mustBeGreaterThan || mustBeIn) {
			return nil, false, nil
		}

		// Always canonize in order of In/NotIn/Gt/Lt/Exists/DoesNotExist and sort values
		if mustBeIn {
			vals := slices.Sorted(maps.Keys(inValues))
			newTerms = append(newTerms, k8sv1.NodeSelectorRequirement{
				Key:      key,
				Operator: k8sv1.NodeSelectorOpIn,
				Values:   vals,
			})
		}

		if mustNotBeIn {
			vals := slices.Sorted(maps.Keys(notInValues))
			newTerms = append(newTerms, k8sv1.NodeSelectorRequirement{
				Key:      key,
				Operator: k8sv1.NodeSelectorOpNotIn,
				Values:   vals,
			})
		}

		if mustBeGreaterThan {
			val := strconv.FormatInt(greaterThanValue, 10)
			newTerms = append(newTerms, k8sv1.NodeSelectorRequirement{
				Key:      key,
				Operator: k8sv1.NodeSelectorOpGt,
				Values:   []string{val},
			})
		}

		if mustBeLessThan {
			val := strconv.FormatInt(lessThanValue, 10)
			newTerms = append(newTerms, k8sv1.NodeSelectorRequirement{
				Key:      key,
				Operator: k8sv1.NodeSelectorOpLt,
				Values:   []string{val},
			})
		}

		if mustExist {
			newTerms = append(newTerms, k8sv1.NodeSelectorRequirement{
				Key:      key,
				Operator: k8sv1.NodeSelectorOpExists,
			})
		}

		if mustNotExist {
			newTerms = append(newTerms, k8sv1.NodeSelectorRequirement{
				Key:      key,
				Operator: k8sv1.NodeSelectorOpDoesNotExist,
			})
		}
	}

	return newTerms, true, nil
}

func nodeSelectorRequirementsCanBeMadeRedundant(reqs1 []k8sv1.NodeSelectorRequirement, reqs2 []k8sv1.NodeSelectorRequirement) (bool, error) {
	reqs1ByKeyOp := nodeSelectorRequirementByKeyAndOperator(reqs1)
	reqs2ByKeyOp := nodeSelectorRequirementByKeyAndOperator(reqs2)
	// If reqs1 is stricter (has more keys) then reqs2, it cannot made it redundant
	if len(reqs1ByKeyOp) > len(reqs2ByKeyOp) {
		return false, nil
	}
	// All keys in reqs1 must be in reqs2
	for key := range reqs1ByKeyOp {
		if _, exists := reqs2ByKeyOp[key]; !exists {
			return false, nil
		}
	}
	// Validate that we have reqs in canonical form
	for key := range reqs1ByKeyOp {
		for op, term := range reqs1ByKeyOp[key] {
			if len(term) > 1 {
				return false, fmt.Errorf("Received denormalized node selector terms for %v", op)
			}
		}
	}
	for key := range reqs2ByKeyOp {
		for op, term := range reqs2ByKeyOp[key] {
			if len(term) > 1 {
				return false, fmt.Errorf("Received denormalized node selector terms for %v", op)
			}
		}
	}

	type normRec struct {
		hasExists       bool
		hasDoesNotExist bool
		hasIn           bool
		hasNotIn        bool
		hasGt           bool
		hasLt           bool
		greaterThanVal  int64
		lessThanVal     int64
		inValuesSet     map[string]struct{}
		notInValuesSet  map[string]struct{}
	}

	toNormRec := func(m map[k8sv1.NodeSelectorOperator][]k8sv1.NodeSelectorRequirement) (normRec, error) {
		_, hasExists := m[k8sv1.NodeSelectorOpExists]
		_, hasDoesNotExist := m[k8sv1.NodeSelectorOpDoesNotExist]

		inTermSlice, hasIn := m[k8sv1.NodeSelectorOpIn]
		var InValuesSet map[string]struct{}
		if hasIn {
			InValuesSet = make(map[string]struct{})
			for _, val := range inTermSlice[0].Values {
				InValuesSet[val] = struct{}{}
			}
		}

		notInTermSlice, hasNotIn := m[k8sv1.NodeSelectorOpNotIn]
		var notInValuesSet map[string]struct{}
		if hasNotIn {
			notInValuesSet = make(map[string]struct{})
			for _, val := range notInTermSlice[0].Values {
				notInValuesSet[val] = struct{}{}
			}
		}

		gtTermSlice, hasGt := m[k8sv1.NodeSelectorOpGt]
		var gtValue int64
		if hasGt {
			valStr := gtTermSlice[0].Values[0]
			val, err := strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				return normRec{}, err
			}
			gtValue = val
		}

		ltTermSlice, hasLt := m[k8sv1.NodeSelectorOpLt]
		var ltValue int64
		if hasLt {
			valStr := ltTermSlice[0].Values[0]
			val, err := strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				return normRec{}, err
			}
			ltValue = val
		}
		return normRec{
			hasExists:       hasExists,
			hasDoesNotExist: hasDoesNotExist,
			hasIn:           hasIn,
			hasNotIn:        hasNotIn,
			hasGt:           hasGt,
			hasLt:           hasLt,
			inValuesSet:     InValuesSet,
			notInValuesSet:  notInValuesSet,
			greaterThanVal:  gtValue,
			lessThanVal:     ltValue,
		}, nil
	}

	// Note: This set of function require that b.has<Operator> is true otherwise the result will be incorrect
	impliesExists := func(a normRec, _ normRec) bool {
		// Returns true if a implies b for the Exists operator
		return a.hasExists || a.hasIn || a.hasGt || a.hasLt
	}

	impliesDoesNotExist := func(a normRec, _ normRec) bool {
		// Returns true if a implies b for the DoesNotExist operator
		return a.hasDoesNotExist
	}

	impliesIn := func(a normRec, b normRec) bool {
		// Returns true if a implies b for the In operator
		if a.hasIn {
			for val := range a.inValuesSet {
				if _, exists := b.inValuesSet[val]; !exists {
					return false
				}
			}
			return true
		}

		// A Gt+Lt-bounded range can imply In if every integer the range admits
		// (minus NotIn'd values) appears in b.inValuesSet. We verify this by
		// counting rather than enumerating, so a wide (Gt, Lt) is cheap.
		if a.hasGt && a.hasLt {
			rangeCount := a.lessThanVal - a.greaterThanVal - 1
			if rangeCount <= 0 {
				return false // simplify should have caught this; bail out conservatively
			}
			for sv := range a.notInValuesSet {
				n, err := strconv.ParseInt(sv, 10, 64)
				if err == nil && n > a.greaterThanVal && n < a.lessThanVal {
					rangeCount -= 1
				}
			}
			var matched int64
			for sv := range b.inValuesSet {
				n, err := strconv.ParseInt(sv, 10, 64)
				if err != nil || n <= a.greaterThanVal || n >= a.lessThanVal {
					continue
				}
				if _, denied := a.notInValuesSet[sv]; denied {
					continue
				}
				matched++
			}
			return matched >= rangeCount
		}

		return false
	}

	impliesNotIn := func(a normRec, b normRec) bool {
		// Returns true if a implies b for the NotIn operator
		if a.hasDoesNotExist {
			return true
		}

		for val := range b.notInValuesSet {
			if a.hasIn {
				if _, exists := a.inValuesSet[val]; exists {
					return false
				}
				continue
			}

			if a.hasNotIn {
				if _, exists := a.notInValuesSet[val]; exists {
					continue
				}
			}

			if a.hasLt || a.hasGt {
				valInt64, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					continue
				}
				if a.hasGt && valInt64 <= a.greaterThanVal {
					continue
				}
				if a.hasLt && valInt64 >= a.lessThanVal {
					continue
				}
			}

			return false
		}

		return true
	}

	impliesGt := func(a normRec, b normRec) bool {
		// Returns true if a implies b for the Gt operator
		if a.hasDoesNotExist || !(a.hasExists || a.hasIn || a.hasGt || a.hasLt) {
			return false
		}

		if a.hasIn {
			for val := range a.inValuesSet {
				valInt64, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return false
				}
				if valInt64 <= b.greaterThanVal {
					return false
				}
			}
			return true
		}

		if a.hasGt && a.greaterThanVal >= b.greaterThanVal {
			return true
		}

		return false
	}

	impliesLt := func(a normRec, b normRec) bool {
		// Returns true if a implies b for the Lt operator
		if a.hasDoesNotExist || !(a.hasExists || a.hasIn || a.hasGt || a.hasLt) {
			return false
		}

		if a.hasIn {
			for val := range a.inValuesSet {
				valInt64, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return false
				}
				if valInt64 >= b.lessThanVal {
					return false
				}
			}
			return true
		}

		if a.hasLt && a.lessThanVal <= b.lessThanVal {
			return true
		}

		return false
	}

	// For each (key, operator) in reqs1, if reqs2 at (key, operator) logically implies reqs1 at (key, operator)
	// that means that for every node that would satisfy reqs2, reqs1 is also satisfied meaning reqs2 is redundant
	// We only have to care about the operators of reqs1 since if all the operators in reqs1 are implied than
	// any extra bounds on reqs2 could only make the set of nodes that satisfy reqs2 smaller than reqs1
	for key := range reqs1ByKeyOp {
		normRec1, err := toNormRec(reqs1ByKeyOp[key])
		if err != nil {
			return false, err
		}
		normRec2, err := toNormRec(reqs2ByKeyOp[key])
		if err != nil {
			return false, err
		}
		if normRec1.hasExists {
			if !impliesExists(normRec2, normRec1) {
				return false, nil
			}
		}
		if normRec1.hasDoesNotExist {
			if !impliesDoesNotExist(normRec2, normRec1) {
				return false, nil
			}
		}
		if normRec1.hasIn {
			if !impliesIn(normRec2, normRec1) {
				return false, nil
			}
		}
		if normRec1.hasNotIn {
			if !impliesNotIn(normRec2, normRec1) {
				return false, nil
			}
		}
		if normRec1.hasGt {
			if !impliesGt(normRec2, normRec1) {
				return false, nil
			}
		}
		if normRec1.hasLt {
			if !impliesLt(normRec2, normRec1) {
				return false, nil
			}
		}
	}

	return true, nil
}

func pruneRedundantNodeSelectorTerms(terms []k8sv1.NodeSelectorTerm) ([]k8sv1.NodeSelectorTerm, error) {
	// This logic cannot be better than n^2 but in all but the most pathological cases, this should be rather fast
	type nodeSelectorTermsWithKeyset struct {
		selector              k8sv1.NodeSelectorTerm
		keySetMatchExpression map[string]struct{}
		keySetMatchFields     map[string]struct{}
	}

	// For a set to be made redundant by another set with OR, it must have equal or less keys
	// Sorting first by keyset cardinality is a small optimization by comparing each set
	// first to sets that are more likely to able to make later constraints redundant
	termsWithKeyset := make([]nodeSelectorTermsWithKeyset, 0, len(terms))
	for _, term := range terms {
		keySetMatchExpression := make(map[string]struct{})
		keySetMatchFields := make(map[string]struct{})
		for _, req := range term.MatchExpressions {
			keySetMatchExpression[req.Key] = struct{}{}
		}
		for _, req := range term.MatchFields {
			keySetMatchFields[req.Key] = struct{}{}
		}
		termsWithKeyset = append(termsWithKeyset, nodeSelectorTermsWithKeyset{
			selector:              term,
			keySetMatchExpression: keySetMatchExpression,
			keySetMatchFields:     keySetMatchFields,
		})
	}
	slices.SortStableFunc(termsWithKeyset, func(a nodeSelectorTermsWithKeyset, b nodeSelectorTermsWithKeyset) int {
		return (len(a.keySetMatchExpression) + len(a.keySetMatchFields)) - (len(b.keySetMatchExpression) + len(b.keySetMatchFields))
	})

	newNodeTerms := make([]k8sv1.NodeSelectorTerm, 0)
	for _, candidateWithKeyset := range termsWithKeyset {
		candidateMatchExpressions := candidateWithKeyset.selector.MatchExpressions
		candidateMatchFields := candidateWithKeyset.selector.MatchFields

		canBeMadeRedundant := false
		for _, kept := range newNodeTerms {
			matchExpressionCanBeMadeRedundantByKept, err := nodeSelectorRequirementsCanBeMadeRedundant(kept.MatchExpressions, candidateMatchExpressions)
			if err != nil {
				return nil, err
			}
			matchFieldsCanBeMadeRedundantByKept, err := nodeSelectorRequirementsCanBeMadeRedundant(kept.MatchFields, candidateMatchFields)
			if err != nil {
				return nil, err
			}
			if matchExpressionCanBeMadeRedundantByKept && matchFieldsCanBeMadeRedundantByKept {
				canBeMadeRedundant = true
				break
			}
		}
		if canBeMadeRedundant {
			continue
		}
		// When cardinality of terms are the <= either matchExpressions/matchFields, this candidate can actually made redundant a node term from newNodeTerms
		filteredNewNodeTerms := make([]k8sv1.NodeSelectorTerm, 0, len(newNodeTerms))
		for _, kept := range newNodeTerms {
			matchExpressionCanMadeRedundantKept, err := nodeSelectorRequirementsCanBeMadeRedundant(candidateMatchExpressions, kept.MatchExpressions)
			if err != nil {
				return nil, err
			}
			matchFieldsCanMadeRedundantKept, err := nodeSelectorRequirementsCanBeMadeRedundant(candidateMatchFields, kept.MatchFields)
			if err != nil {
				return nil, err
			}
			if !matchExpressionCanMadeRedundantKept || !matchFieldsCanMadeRedundantKept {
				filteredNewNodeTerms = append(filteredNewNodeTerms, kept)
			}
		}
		newNodeTerms = append(filteredNewNodeTerms, candidateWithKeyset.selector)
	}

	return newNodeTerms, nil
}

func mergeNodeSelectorTerms(terms1 []k8sv1.NodeSelectorTerm, terms2 []k8sv1.NodeSelectorTerm) ([]k8sv1.NodeSelectorTerm, error) {
	if len(terms1)*len(terms2) > maxNodeSelectorCrossProduct {
		return nil, fmt.Errorf("node selector cross-product would exceed %d terms (%d × %d); affinity inputs are likely pathological or incompatible", maxNodeSelectorCrossProduct, len(terms1), len(terms2))
	}
	newTerms := make([]k8sv1.NodeSelectorTerm, 0, max(len(terms1), len(terms2)))
	for _, term1 := range terms1 {
		for _, term2 := range terms2 {
			matchExpressions, satisfiable, err := simplifyNodeSelectorRequirements(append(append([]k8sv1.NodeSelectorRequirement{}, term1.MatchExpressions...), term2.MatchExpressions...))
			if err != nil {
				return nil, err
			}
			if !satisfiable {
				continue
			}

			matchFields, satisfiable, err := simplifyNodeSelectorRequirements(append(append([]k8sv1.NodeSelectorRequirement{}, term1.MatchFields...), term2.MatchFields...))
			if err != nil {
				return nil, err
			}
			if !satisfiable {
				continue
			}

			newTerms = append(newTerms, k8sv1.NodeSelectorTerm{
				MatchExpressions: matchExpressions,
				MatchFields:      matchFields,
			})
		}
	}
	// Since nodeSelectorTerms are OR'd, only one term actually has to be satisfiable
	// If nothing in the cross product is satisfiable we should error
	if len(newTerms) == 0 {
		return nil, fmt.Errorf("all merged terms are unsatisfiable for cross product")
	}
	return pruneRedundantNodeSelectorTerms(newTerms)
}

func mergeNestedNodeSelectorTerms(nestedTerms [][]k8sv1.NodeSelectorTerm) ([]k8sv1.NodeSelectorTerm, error) {
	res := []k8sv1.NodeSelectorTerm{{}}
	for _, term := range nestedTerms {
		var err error
		res, err = mergeNodeSelectorTerms(res, normalizeNodeSelectorTerms(term))
		if err != nil {
			return nil, err
		}
	}
	if len(res) > maxNodeSelectorFinalTerms {
		return nil, fmt.Errorf("merged node selector produced %d terms (limit %d); disjuncts do not subsume each other", len(res), maxNodeSelectorFinalTerms)
	}
	return res, nil
}

func modifyNodeAffinityToRejectLabel(origAffinity *k8sv1.Affinity, labelToReject string) *k8sv1.Affinity {
	affinity := origAffinity.DeepCopy()
	requirement := k8sv1.NodeSelectorRequirement{
		Key:      labelToReject,
		Operator: k8sv1.NodeSelectorOpDoesNotExist,
	}
	term := k8sv1.NodeSelectorTerm{
		MatchExpressions: []k8sv1.NodeSelectorRequirement{requirement}}

	nodeAffinity := &k8sv1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
			NodeSelectorTerms: []k8sv1.NodeSelectorTerm{term},
		},
	}
	if affinity != nil && affinity.NodeAffinity != nil {
		if affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			terms := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			// Since NodeSelectorTerms are ORed , the anti affinity requirement will be added to each term.
			for i, selectorTerm := range terms {
				affinity.NodeAffinity.
					RequiredDuringSchedulingIgnoredDuringExecution.
					NodeSelectorTerms[i].MatchExpressions = append(selectorTerm.MatchExpressions, requirement)
			}
		} else {
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &k8sv1.NodeSelector{
				NodeSelectorTerms: []k8sv1.NodeSelectorTerm{term},
			}
		}

	} else if affinity != nil {
		affinity.NodeAffinity = nodeAffinity
	} else {
		affinity = &k8sv1.Affinity{
			NodeAffinity: nodeAffinity,
		}
	}
	return affinity
}

func sysprepVolumeSource(sysprepVolume v1.SysprepSource) (k8sv1.VolumeSource, error) {
	logger := log.DefaultLogger()
	if sysprepVolume.Secret != nil {
		return k8sv1.VolumeSource{
			Secret: &k8sv1.SecretVolumeSource{
				SecretName: sysprepVolume.Secret.Name,
			},
		}, nil
	} else if sysprepVolume.ConfigMap != nil {
		return k8sv1.VolumeSource{
			ConfigMap: &k8sv1.ConfigMapVolumeSource{
				LocalObjectReference: k8sv1.LocalObjectReference{
					Name: sysprepVolume.ConfigMap.Name,
				},
			},
		}, nil
	}
	errorStr := fmt.Sprintf("Sysprep must have Secret or ConfigMap reference set %v", sysprepVolume)
	logger.Errorf("%s", errorStr)
	return k8sv1.VolumeSource{}, fmt.Errorf("%s", errorStr)
}

func (t *TemplateService) GetLauncherImage() string {
	return t.launcherImage
}

func (t *TemplateService) RenderLaunchManifestNoVm(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	backendStoragePVCName := ""
	if backendstorage.IsBackendStorageNeeded(vmi) {
		backendStoragePVC := backendstorage.PVCForVMI(t.persistentVolumeClaimStore, vmi)
		if backendStoragePVC == nil {
			return nil, fmt.Errorf("can't generate manifest without backend-storage PVC, waiting for the PVC to be created")
		}
		backendStoragePVCName = backendStoragePVC.Name
	}
	memoryOverhead := CalculateMemoryOverhead(t.clusterConfig, t.netMemoryCalculator, vmi, t.launcherHypervisorResources)
	return t.renderLaunchManifest(vmi, nil, backendStoragePVCName, true, memoryOverhead)
}

func (t *TemplateService) RenderMigrationManifest(vmi *v1.VirtualMachineInstance, migration *v1.VirtualMachineInstanceMigration, sourcePod *k8sv1.Pod) (*k8sv1.Pod, error) {
	reproducibleImageIDs, err := containerdisk.ExtractImageIDsFromSourcePod(vmi, sourcePod, t.clusterConfig.ImageVolumeEnabled())
	if err != nil {
		return nil, fmt.Errorf("can not proceed with the migration when no reproducible image digest can be detected: %v", err)
	}
	backendStoragePVCName := ""
	if backendstorage.IsBackendStorageNeeded(vmi) {
		backendStoragePVC := backendstorage.PVCForMigrationTarget(t.persistentVolumeClaimStore, migration)
		if backendStoragePVC == nil {
			return nil, fmt.Errorf("can't generate manifest without backend-storage PVC, waiting for the PVC to be created")
		}
		backendStoragePVCName = backendStoragePVC.Name
	}
	memoryOverhead := CalculateMemoryOverhead(t.clusterConfig, t.netMemoryCalculator, vmi, t.launcherHypervisorResources)
	targetPod, err := t.renderLaunchManifest(vmi, reproducibleImageIDs, backendStoragePVCName, false, memoryOverhead)
	if err != nil {
		return nil, err
	}

	if t.netTargetAnnotationsGenerator != nil {
		netAnnotations, err := t.netTargetAnnotationsGenerator.GenerateFromSource(vmi, sourcePod)
		if err != nil {
			return nil, err
		}

		maps.Copy(targetPod.Annotations, netAnnotations)
	}

	return targetPod, err
}

func (t *TemplateService) RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	backendStoragePVCName := ""
	if backendstorage.IsBackendStorageNeeded(vmi) {
		backendStoragePVC := backendstorage.PVCForVMI(t.persistentVolumeClaimStore, vmi)
		if backendStoragePVC == nil {
			return nil, fmt.Errorf("can't generate manifest without backend-storage PVC, waiting for the PVC to be created")
		}
		backendStoragePVCName = backendStoragePVC.Name
	}
	memoryOverhead := CalculateMemoryOverhead(t.clusterConfig, t.netMemoryCalculator, vmi, t.launcherHypervisorResources)
	return t.renderLaunchManifest(vmi, nil, backendStoragePVCName, false, memoryOverhead)
}

func generateQemuTimeoutWithJitter(qemuTimeoutBaseSeconds int) string {
	timeout := rand.Intn(qemuTimeoutJitterRange) + qemuTimeoutBaseSeconds

	return fmt.Sprintf("%ds", timeout)
}

func computePodSecurityContext(vmi *v1.VirtualMachineInstance, seccomp *k8sv1.SeccompProfile) *k8sv1.PodSecurityContext {
	psc := &k8sv1.PodSecurityContext{}

	// virtiofs container will run unprivileged even if the pod runs as root,
	// so we need to allow the NonRootUID for virtiofsd to be able to write into the PVC
	psc.FSGroup = pointer.P(int64(util.NonRootUID))

	if vmitrait.IsNonRoot(vmi) {
		nonRootUser := int64(util.NonRootUID)
		psc.RunAsUser = &nonRootUser
		psc.RunAsGroup = &nonRootUser
		psc.RunAsNonRoot = pointer.P(true)
	} else {
		rootUser := int64(util.RootUser)
		psc.RunAsUser = &rootUser
	}
	psc.SeccompProfile = seccomp

	return psc
}

func (t *TemplateService) renderLaunchManifest(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, backendStoragePVCName string, tempPod bool, memoryOverhead resource.Quantity) (*k8sv1.Pod, error) {
	precond.MustNotBeNil(vmi)
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())

	var userId int64 = util.RootUser

	nonRoot := vmitrait.IsNonRoot(vmi)
	if nonRoot {
		userId = util.NonRootUID
	}

	// Pad the virt-launcher grace period.
	// Ideally we want virt-handler to handle tearing down
	// the vmi without virt-launcher's termination forcing
	// the vmi down.
	const gracePeriodPaddingSeconds int64 = 15
	gracePeriodSeconds := gracePeriodInSeconds(vmi) + gracePeriodPaddingSeconds
	gracePeriodKillAfter := gracePeriodSeconds + gracePeriodPaddingSeconds

	imagePullSecrets := imgPullSecrets(vmi.Spec.Volumes...)
	if util.HasKernelBootContainerImage(vmi) && vmi.Spec.Domain.Firmware.KernelBoot.Container.ImagePullSecret != "" {
		imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, k8sv1.LocalObjectReference{
			Name: vmi.Spec.Domain.Firmware.KernelBoot.Container.ImagePullSecret,
		})
	}
	if t.imagePullSecret != "" {
		imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, k8sv1.LocalObjectReference{
			Name: t.imagePullSecret,
		})
	}

	var networkToResourceMap map[string]string
	if !t.clusterConfig.ExternalNetResourceInjectionEnabled() {
		var err error
		networkToResourceMap, err = multus.NetworkToResource(t.virtClient, vmi)
		if err != nil {
			return nil, err
		}
	}
	resourceRenderer, err := t.newResourceRenderer(vmi, networkToResourceMap, memoryOverhead)
	if err != nil {
		return nil, err
	}
	resources := resourceRenderer.ResourceRequirements()

	ovmfPath := t.clusterConfig.GetOVMFPath(vmi.Spec.Architecture)

	var requestedHookSidecarList hooks.HookSidecarList
	for _, sidecarCreator := range t.sidecarCreators {
		sidecars, err := sidecarCreator(vmi, t.clusterConfig.GetConfig())
		if err != nil {
			return nil, err
		}
		requestedHookSidecarList = append(requestedHookSidecarList, sidecars...)
	}

	var command []string
	if tempPod {
		logger := log.DefaultLogger()
		logger.Infof("RUNNING doppleganger pod for %s", vmi.Name)
		command = []string{"/bin/bash",
			"-c",
			"echo", "bound PVCs"}
	} else {
		command = []string{"/usr/bin/virt-launcher-monitor",
			"--qemu-timeout", generateQemuTimeoutWithJitter(t.launcherQemuTimeout),
			"--name", domain,
			"--uid", string(vmi.UID),
			"--namespace", namespace,
			"--kubevirt-share-dir", t.virtShareDir,
			"--ephemeral-disk-dir", t.ephemeralDiskDir,
			"--container-disk-dir", t.containerDiskDir,
			"--grace-period-seconds", strconv.Itoa(int(gracePeriodSeconds)),
			"--hook-sidecars", strconv.Itoa(len(requestedHookSidecarList)),
			"--ovmf-path", ovmfPath,
			"--disk-memory-limit", strconv.Itoa(int(t.clusterConfig.GetDiskVerification().MemoryLimit.Value())),
			"--hypervisor", t.clusterConfig.GetHypervisor().Name,
		}
		if nonRoot {
			command = append(command, "--run-as-nonroot")
		}
		if t.clusterConfig.ImageVolumeEnabled() {
			command = append(command, "--image-volume")
		}
		if t.clusterConfig.LibvirtHooksServerAndClientEnabled() {
			command = append(command, "--libvirt-hook-server-and-client")
		}
		if t.clusterConfig.PodSecondaryInterfaceNamingUpgradeEnabled() {
			command = append(command, "--upgrade-ordinal-ifaces")
		}
		if t.clusterConfig.VGPULiveMigrationEnabled() {
			command = append(command, "--vgpu-dedicated-hook")
		}
		if t.clusterConfig.VMStatsCollectorEnabled() {
			command = append(command, "--vm-stats-collector")
		}
		if t.clusterConfig.FirmwareAutoSelectionEnabled() {
			command = append(command, "--firmware-auto-selection")
		}
		if customDebugFilters, exists := vmi.Annotations[v1.CustomLibvirtLogFiltersAnnotation]; exists {
			log.Log.Object(vmi).Infof("Applying custom debug filters for vmi %s: %s", vmi.Name, customDebugFilters)
			command = append(command, "--libvirt-log-filters", customDebugFilters)
		}
	}

	if t.clusterConfig.AllowEmulation() {
		command = append(command, "--allow-emulation")
	}

	if t.clusterConfig.CrossArchitectureVirtualizationEnabled() {
		command = append(command, "--allow-cross-arch-emulation")
	}

	if checkForKeepLauncherAfterFailure(vmi) {
		command = append(command, "--keep-after-failure")
	}

	_, ok := vmi.Annotations[v1.FuncTestLauncherFailFastAnnotation]
	if ok {
		command = append(command, "--simulate-crash")
	}

	volumeRenderer, err := t.newVolumeRenderer(vmi, imageIDs, namespace, requestedHookSidecarList, backendStoragePVCName)
	if err != nil {
		return nil, err
	}

	compute := t.newContainerSpecRenderer(vmi, volumeRenderer, resources, userId).Render(command)

	virtLauncherLogVerbosity := t.clusterConfig.GetVirtLauncherVerbosity()

	if verbosity, isSet := vmi.Labels[logVerbosity]; isSet || virtLauncherLogVerbosity != virtconfig.DefaultVirtLauncherLogVerbosity {
		// Override the cluster wide verbosity level if a specific value has been provided for this VMI
		verbosityStr := fmt.Sprint(virtLauncherLogVerbosity)
		if isSet {
			verbosityStr = verbosity

			verbosityInt, err := strconv.Atoi(verbosity)
			if err != nil {
				return nil, fmt.Errorf("verbosity %s cannot cast to int: %v", verbosity, err)
			}

			virtLauncherLogVerbosity = uint(verbosityInt)
		}
		compute.Env = append(compute.Env, k8sv1.EnvVar{Name: util.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY, Value: verbosityStr})
	}

	if labelValue, ok := vmi.Labels[debugLogs]; (ok && strings.EqualFold(labelValue, "true")) || virtLauncherLogVerbosity > util.EXT_LOG_VERBOSITY_THRESHOLD {
		compute.Env = append(compute.Env, k8sv1.EnvVar{Name: util.ENV_VAR_LIBVIRT_DEBUG_LOGS, Value: "1"})
	}
	if labelValue, ok := vmi.Labels[virtiofsDebugLogs]; (ok && strings.EqualFold(labelValue, "true")) || virtLauncherLogVerbosity > util.EXT_LOG_VERBOSITY_THRESHOLD {
		compute.Env = append(compute.Env, k8sv1.EnvVar{Name: util.ENV_VAR_VIRTIOFSD_DEBUG_LOGS, Value: "1"})
	}

	compute.Env = append(compute.Env, k8sv1.EnvVar{
		Name: ENV_VAR_POD_NAME,
		ValueFrom: &k8sv1.EnvVarSource{
			FieldRef: &k8sv1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})

	// Make sure the compute container is always the first since the mutating webhook shipped with the sriov operator
	// for adding the requested resources to the pod will add them to the first container of the list
	containers := []k8sv1.Container{compute}
	if !t.clusterConfig.ImageVolumeEnabled() {
		containersDisks := containerdisk.GenerateContainers(vmi, t.clusterConfig, imageIDs, containerDisks, virtBinDir)
		containers = append(containers, containersDisks...)

		kernelBootContainer := containerdisk.GenerateKernelBootContainer(vmi, t.clusterConfig, imageIDs, containerDisks, virtBinDir)
		if kernelBootContainer != nil {
			log.Log.Object(vmi).Infof("kernel boot container generated")
			containers = append(containers, *kernelBootContainer)
		}
	}

	virtiofsContainers := generateVirtioFSContainers(vmi, t.launcherImage, t.clusterConfig)
	if virtiofsContainers != nil {
		containers = append(containers, virtiofsContainers...)
	}

	var sidecarVolumes []k8sv1.Volume
	for i, requestedHookSidecar := range requestedHookSidecarList {
		sidecarContainer := newSidecarContainerRenderer(
			sidecarContainerName(i), vmi, sidecarResources(vmi, t.clusterConfig), requestedHookSidecar, userId).Render(requestedHookSidecar.Command)

		if requestedHookSidecar.ConfigMap != nil {
			cm, err := t.virtClient.CoreV1().ConfigMaps(vmi.Namespace).Get(context.TODO(), requestedHookSidecar.ConfigMap.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			volumeSource := k8sv1.VolumeSource{
				ConfigMap: &k8sv1.ConfigMapVolumeSource{
					LocalObjectReference: k8sv1.LocalObjectReference{Name: cm.Name},
					DefaultMode:          pointer.P(int32(0755)),
				},
			}
			vol := k8sv1.Volume{
				Name:         cm.Name,
				VolumeSource: volumeSource,
			}
			sidecarVolumes = append(sidecarVolumes, vol)
		}
		if requestedHookSidecar.PVC != nil {
			volumeSource := k8sv1.VolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: requestedHookSidecar.PVC.Name,
				},
			}
			vol := k8sv1.Volume{
				Name:         requestedHookSidecar.PVC.Name,
				VolumeSource: volumeSource,
			}
			sidecarVolumes = append(sidecarVolumes, vol)
			if requestedHookSidecar.PVC.SharedComputePath != "" {
				containers[0].VolumeMounts = append(containers[0].VolumeMounts,
					k8sv1.VolumeMount{
						Name:      requestedHookSidecar.PVC.Name,
						MountPath: requestedHookSidecar.PVC.SharedComputePath,
					})
			}
		}
		containers = append(containers, sidecarContainer)
	}

	podAnnotations, err := t.generatePodAnnotations(vmi)
	if err != nil {
		return nil, err
	}
	if tempPod {
		// mark pod as temp - only used for provisioning
		podAnnotations[v1.EphemeralProvisioningObject] = "true"
	}

	if t.clusterConfig.VmiMemoryOverheadReportEnabled() {
		podAnnotations[v1.MemoryOverheadAnnotationBytes] = strconv.FormatInt(memoryOverhead.Value(), 10)
	}

	var initContainers []k8sv1.Container

	sconsolelogContainer := generateSerialConsoleLogContainer(vmi, t.launcherImage, t.clusterConfig, virtLauncherLogVerbosity)
	if sconsolelogContainer != nil {
		initContainers = append(initContainers, *sconsolelogContainer)
	}

	if !t.clusterConfig.ImageVolumeEnabled() && (HaveContainerDiskVolume(vmi.Spec.Volumes) || util.HasKernelBootContainerImage(vmi)) {
		initContainerCommand := []string{"/usr/bin/cp", "--preserve=all",
			"/usr/bin/container-disk",
			"/init/usr/bin/container-disk",
		}

		initContainers = append(
			initContainers,
			t.newInitContainerRenderer(vmi,
				initContainerVolumeMount(),
				initContainerResourceRequirementsForVMI(vmi, v1.ContainerDisk, t.clusterConfig),
				userId).Render(initContainerCommand))

		// this causes containerDisks to be pre-pulled before virt-launcher starts.
		initContainers = append(initContainers, containerdisk.GenerateInitContainers(vmi, t.clusterConfig, imageIDs, containerDisks, virtBinDir)...)

		kernelBootInitContainer := containerdisk.GenerateKernelBootInitContainer(vmi, t.clusterConfig, imageIDs, containerDisks, virtBinDir)
		if kernelBootInitContainer != nil {
			initContainers = append(initContainers, *kernelBootInitContainer)
		}
	} else if t.clusterConfig.ImageVolumeEnabled() {
		// TODO: Once the KEP https://github.com/kubernetes/enhancements/pull/5375 is fully implemented and stable
		// in all Kubernetes versions supported by KubeVirt, this entire init containers logic should be removed,
		// and the digest can be fetched directly from the Pod volume status.
		// Generate init containers for regular volumes
		for _, volume := range vmi.Spec.Volumes {
			containerDiskImageIDAlreadyExists := strings.Contains(imageIDs[volume.Name], "@sha256:")
			if volume.ContainerDisk == nil || containerDiskImageIDAlreadyExists {
				continue
			}
			initContainer := containerdisk.CreateImageVolumeInitContainer(
				vmi,
				t.clusterConfig,
				volume.Name,
				volume.ContainerDisk.Image,
				volume.ContainerDisk.ImagePullPolicy,
			)
			initContainers = append(initContainers, initContainer)
		}

		// Generate init container for kernel boot if needed
		kernelBootImageIDAlreadyExists := strings.Contains(imageIDs[containerdisk.KernelBootVolumeName], "@sha256:")
		if util.HasKernelBootContainerImage(vmi) && !kernelBootImageIDAlreadyExists {
			kernelBootContainer := vmi.Spec.Domain.Firmware.KernelBoot.Container
			initContainer := containerdisk.CreateImageVolumeInitContainer(
				vmi,
				t.clusterConfig,
				containerdisk.KernelBootVolumeName,
				kernelBootContainer.Image,
				kernelBootContainer.ImagePullPolicy,
			)
			initContainers = append(initContainers, initContainer)
		}
	}

	hostName := dns.SanitizeHostname(vmi)
	enableServiceLinks := false

	var podSeccompProfile *k8sv1.SeccompProfile = nil
	if seccompConf := t.clusterConfig.GetConfig().SeccompConfiguration; seccompConf != nil && seccompConf.VirtualMachineInstanceProfile != nil {
		vmProfile := seccompConf.VirtualMachineInstanceProfile
		if customProfile := vmProfile.CustomProfile; customProfile != nil {
			if customProfile.LocalhostProfile != nil {
				podSeccompProfile = &k8sv1.SeccompProfile{
					Type:             k8sv1.SeccompProfileTypeLocalhost,
					LocalhostProfile: customProfile.LocalhostProfile,
				}
			} else if customProfile.RuntimeDefaultProfile {
				podSeccompProfile = &k8sv1.SeccompProfile{
					Type: k8sv1.SeccompProfileTypeRuntimeDefault,
				}
			}
		}

	}
	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-",
			Labels:       podLabels(vmi, hostName),
			Annotations:  podAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
			},
		},
		Spec: k8sv1.PodSpec{
			Hostname:                      hostName,
			Subdomain:                     vmi.Spec.Subdomain,
			SecurityContext:               computePodSecurityContext(vmi, podSeccompProfile),
			TerminationGracePeriodSeconds: &gracePeriodKillAfter,
			RestartPolicy:                 k8sv1.RestartPolicyNever,
			Containers:                    containers,
			InitContainers:                initContainers,
			NodeSelector:                  t.newNodeSelectorRenderer(vmi).Render(),
			Volumes:                       volumeRenderer.Volumes(),
			ImagePullSecrets:              imagePullSecrets,
			DNSConfig:                     vmi.Spec.DNSConfig,
			DNSPolicy:                     vmi.Spec.DNSPolicy,
			ReadinessGates:                readinessGates(),
			EnableServiceLinks:            &enableServiceLinks,
			SchedulerName:                 vmi.Spec.SchedulerName,
			Tolerations:                   vmi.Spec.Tolerations,
			TopologySpreadConstraints:     vmi.Spec.TopologySpreadConstraints,
			ResourceClaims:                drautil.ToPodResourceClaims(vmi.Spec.ResourceClaims),
		},
	}

	alignPodMultiCategorySecurity(&pod, t.clusterConfig.GetSELinuxLauncherType(), t.clusterConfig.DockerSELinuxMCSWorkaroundEnabled())

	// If we have a runtime class specified, use it, otherwise don't set a runtimeClassName
	runtimeClassName := t.clusterConfig.GetDefaultRuntimeClass()
	if runtimeClassName != "" {
		pod.Spec.RuntimeClassName = &runtimeClassName
	}

	if vmi.Spec.PriorityClassName != "" {
		pod.Spec.PriorityClassName = vmi.Spec.PriorityClassName
	}

	if vmi.Spec.Affinity != nil {
		pod.Spec.Affinity = vmi.Spec.Affinity.DeepCopy()
	}

	if err := t.setNodeAffinityForPod(vmi, &pod); err != nil {
		return nil, err
	}
	if err := setPersistentReservationAntiAffinity(vmi, &pod, t.persistentVolumeClaimStore); err != nil {
		return nil, err
	}

	if t.clusterConfig.CrossArchitectureVirtualizationEnabled() {
		setPreferredArchitectureAffinity(vmi.Spec.Architecture, &pod)
		if vmi.Spec.Architecture != "" {
			if pod.Spec.NodeSelector == nil {
				pod.Spec.NodeSelector = map[string]string{}
			}
			pod.Spec.NodeSelector[v1.VMArchLabel+vmi.Spec.Architecture] = "true"
		}
	}

	serviceAccountVolumeName := storageutils.ServiceAccountNameFromVolumes(vmi.Spec.Volumes)
	if vmi.Spec.ServiceAccountName != "" {
		pod.Spec.ServiceAccountName = vmi.Spec.ServiceAccountName
	} else if serviceAccountVolumeName != "" {
		pod.Spec.ServiceAccountName = serviceAccountVolumeName
	}

	if serviceAccountVolumeName != "" || istio.ProxyInjectionEnabled(vmi) {
		automount := true
		pod.Spec.AutomountServiceAccountToken = &automount
	} else {
		automount := false
		pod.Spec.AutomountServiceAccountToken = &automount
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, sidecarVolumes...)

	return &pod, nil
}

func (t *TemplateService) newNodeSelectorRenderer(vmi *v1.VirtualMachineInstance) *NodeSelectorRenderer {
	var opts []NodeSelectorRendererOption
	if vmi.IsCPUDedicated() {
		opts = append(opts, WithDedicatedCPU())
	}
	if t.clusterConfig.HypervStrictCheckEnabled() {
		opts = append(opts, WithHyperv(vmi.Spec.Domain.Features))
	}

	if modelLabel, err := CPUModelLabelFromCPUModel(vmi); err == nil {
		opts = append(
			opts,
			WithModelAndFeatureLabels(modelLabel, CPUFeatureLabelsFromCPUFeatures(vmi)...),
		)
	}

	var machineType string
	if vmi.Status.Machine != nil && vmi.Status.Machine.Type != "" {
		machineType = vmi.Status.Machine.Type
	} else if vmi.Spec.Domain.Machine != nil && vmi.Spec.Domain.Machine.Type != "" {
		machineType = vmi.Spec.Domain.Machine.Type
	}

	if machineType != "" {
		opts = append(opts, WithMachineType(machineType))
	}

	if topology.IsManualTSCFrequencyRequired(vmi) {
		opts = append(opts, WithTSCTimer(vmi.Status.TopologyHints.TSCFrequency))
	}

	if vmi.IsRealtimeEnabled() {
		log.Log.V(4).Info("Add realtime node label selector")
		opts = append(opts, WithRealtime())
	}
	if util.IsSEVVMI(vmi) {
		log.Log.V(4).Info("Add SEV node label selector")
		opts = append(opts, WithSEVSelector())
	}
	if util.IsSEVESVMI(vmi) {
		log.Log.V(4).Info("Add SEV-ES node label selector")
		opts = append(opts, WithSEVESSelector())
	}

	if util.IsSEVSNPVMI(vmi) {
		log.Log.V(4).Info("Add SEV-SNP node label selector")
		opts = append(opts, WithSEVSNPSelector())
	}

	if util.IsSecureExecutionVMI(vmi) {
		log.Log.V(4).Info("Add Secure Execution node label selector")
		opts = append(opts, WithSecureExecutionSelector())
	}

	if util.IsTDXVMI(vmi) {
		log.Log.V(4).Info("Add TDX node label selector")
		opts = append(opts, WithTDXSelector())
	}

	if t.clusterConfig.CrossArchitectureVirtualizationEnabled() && vmi.Spec.Architecture != "" {
		opts = append(opts, WithoutNativeArchSelector())
	}

	return NewNodeSelectorRenderer(
		vmi.Spec.NodeSelector,
		t.clusterConfig.GetNodeSelectors(),
		vmi.Spec.Architecture,
		opts...,
	)
}

func initContainerVolumeMount() k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      virtBinDir,
		MountPath: "/init/usr/bin",
	}
}

func newSidecarContainerRenderer(sidecarName string, vmiSpec *v1.VirtualMachineInstance, resources k8sv1.ResourceRequirements, requestedHookSidecar hooks.HookSidecar, userId int64) *ContainerSpecRenderer {
	sidecarOpts := []Option{
		WithResourceRequirements(resources),
		WithArgs(requestedHookSidecar.Args),
		WithExtraEnvVars([]k8sv1.EnvVar{
			k8sv1.EnvVar{
				Name:  hooks.ContainerNameEnvVar,
				Value: sidecarName,
			}}),
	}

	var mounts []k8sv1.VolumeMount
	mounts = append(mounts, sidecarVolumeMount(sidecarName))
	if requestedHookSidecar.DownwardAPI == v1.DeviceInfo {
		mounts = append(mounts, mountPath(downwardapi.NetworkInfoVolumeName, downwardapi.MountPath))
	}
	if requestedHookSidecar.ConfigMap != nil {
		mounts = append(mounts, configMapVolumeMount(*requestedHookSidecar.ConfigMap))
	}
	if requestedHookSidecar.PVC != nil {
		mounts = append(mounts, pvcVolumeMount(*requestedHookSidecar.PVC))
	}
	sidecarOpts = append(sidecarOpts, WithVolumeMounts(mounts...))

	if vmitrait.IsNonRoot(vmiSpec) {
		sidecarOpts = append(sidecarOpts, WithNonRoot(userId))
		sidecarOpts = append(sidecarOpts, WithDropALLCapabilities())
	}
	if requestedHookSidecar.Image == "" {
		requestedHookSidecar.Image = os.Getenv(operatorutil.SidecarShimImageEnvName)
	}

	return NewContainerSpecRenderer(
		sidecarName,
		requestedHookSidecar.Image,
		requestedHookSidecar.ImagePullPolicy,
		sidecarOpts...)
}

func (t *TemplateService) newInitContainerRenderer(vmiSpec *v1.VirtualMachineInstance, initContainerVolumeMount k8sv1.VolumeMount, initContainerResources k8sv1.ResourceRequirements, userId int64) *ContainerSpecRenderer {
	const containerDisk = "container-disk-binary"
	cpInitContainerOpts := []Option{
		WithVolumeMounts(initContainerVolumeMount),
		WithResourceRequirements(initContainerResources),
		WithNoCapabilities(),
	}

	if vmitrait.IsNonRoot(vmiSpec) {
		cpInitContainerOpts = append(cpInitContainerOpts, WithNonRoot(userId))
	}

	return NewContainerSpecRenderer(containerDisk, t.launcherImage, t.clusterConfig.GetImagePullPolicy(), cpInitContainerOpts...)
}

func (t *TemplateService) newContainerSpecRenderer(vmi *v1.VirtualMachineInstance, volumeRenderer *VolumeRenderer, resources k8sv1.ResourceRequirements, userId int64) *ContainerSpecRenderer {
	computeContainerOpts := []Option{
		WithVolumeDevices(volumeRenderer.VolumeDevices()...),
		WithVolumeMounts(volumeRenderer.Mounts()...),
		WithSharedFilesystems(volumeRenderer.SharedFilesystemPaths()...),
		WithResourceRequirements(resources),
		WithPorts(vmi),
		WithCapabilities(vmi),
	}
	if vmitrait.IsNonRoot(vmi) {
		computeContainerOpts = append(computeContainerOpts, WithNonRoot(userId))
		computeContainerOpts = append(computeContainerOpts, WithDropALLCapabilities())
	}
	if vmi.Spec.ReadinessProbe != nil {
		computeContainerOpts = append(computeContainerOpts, WithReadinessProbe(vmi))
	}

	if vmi.Spec.LivenessProbe != nil {
		computeContainerOpts = append(computeContainerOpts, WithLivelinessProbe(vmi))
	}

	const computeContainerName = "compute"
	containerRenderer := NewContainerSpecRenderer(
		computeContainerName, t.launcherImage, t.clusterConfig.GetImagePullPolicy(), computeContainerOpts...)
	return containerRenderer
}

func (t *TemplateService) newVolumeRenderer(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, namespace string, requestedHookSidecarList hooks.HookSidecarList, backendStoragePVCName string) (*VolumeRenderer, error) {
	imageVolumeFeatureGateEnabled := t.clusterConfig.ImageVolumeEnabled()
	volumeOpts := []VolumeRendererOption{
		withVMIConfigVolumes(vmi.Spec.Domain.Devices.Disks, vmi.Spec.Volumes),
		withVMIVolumes(t.persistentVolumeClaimStore, vmi.Spec.Volumes, vmi.Status.VolumeStatus),
		withAccessCredentials(vmi.Spec.AccessCredentials),
		withBackendStorage(vmi, backendStoragePVCName),
	}
	if imageVolumeFeatureGateEnabled {
		volumeOpts = append(volumeOpts, withImageVolumes(vmi))
	}
	if len(requestedHookSidecarList) != 0 {
		volumeOpts = append(volumeOpts, withSidecarVolumes(requestedHookSidecarList))
	}
	if t.clusterConfig.PluginsEnabled() {
		volumeOpts = append(volumeOpts, withPluginSocketVolume())
	}

	if hasHugePages(vmi) {
		volumeOpts = append(volumeOpts, withHugepages())
	}

	if !vmi.Spec.Domain.Devices.DisableHotplug {
		volumeOpts = append(volumeOpts, withHotplugSupport(t.hotplugDiskDir))
	}

	if vmispec.BindingPluginNetworkWithDeviceInfoExist(vmi.Spec.Domain.Devices.Interfaces, t.clusterConfig.GetNetworkBindings()) ||
		vmispec.SRIOVInterfaceExist(vmi.Spec.Domain.Devices.Interfaces) {
		volumeOpts = append(volumeOpts, func(renderer *VolumeRenderer) error {
			renderer.podVolumeMounts = append(renderer.podVolumeMounts, mountPath(downwardapi.NetworkInfoVolumeName, downwardapi.MountPath))
			return nil
		})
		volumeOpts = append(volumeOpts, withNetworkDeviceInfoMapAnnotation())
	}

	if util.IsVMIVirtiofsEnabled(vmi) {
		volumeOpts = append(volumeOpts, withVirioFS())
	}

	volumeRenderer, err := NewVolumeRenderer(
		t.clusterConfig,
		imageVolumeFeatureGateEnabled,
		t.launcherImage,
		imageIDs,
		namespace,
		t.ephemeralDiskDir,
		t.containerDiskDir,
		t.virtShareDir,
		volumeOpts...)

	if err != nil {
		return nil, err
	}
	return volumeRenderer, nil
}

func (t *TemplateService) newResourceRenderer(vmi *v1.VirtualMachineInstance, networkToResourceMap map[string]string, memoryOverhead resource.Quantity) (*ResourceRenderer, error) {
	vmiResources := vmi.Spec.Domain.Resources
	hypervisorResource := ConstructHypervisorResourceName(t.launcherHypervisorResources)
	baseOptions := []ResourceRendererOption{
		WithEphemeralStorageRequest(),
		WithVirtualizationResources(getRequiredResources(vmi, hypervisorResource, t.clusterConfig.AllowEmulation())),
	}

	if err := validatePermittedHostDevices(&vmi.Spec, t.clusterConfig); err != nil {
		return nil, err
	}

	options := append(baseOptions, t.VMIResourcePredicates(vmi, networkToResourceMap, memoryOverhead).Apply()...)
	return NewResourceRenderer(vmiResources.Limits, vmiResources.Requests, options...), nil
}

func ConstructHypervisorResourceName(l hypervisor.LauncherHypervisorResources) k8sv1.ResourceName {
	return k8sv1.ResourceName(K8sDevicePrefix + "/" + l.GetHypervisorDevice())
}

func sidecarVolumeMount(containerName string) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      hookSidecarSocks,
		MountPath: hooks.HookSocketsSharedDirectory,
		SubPath:   containerName,
	}
}

func configMapVolumeMount(v hooks.ConfigMap) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      v.Name,
		MountPath: v.HookPath,
		SubPath:   v.Key,
	}
}

func pvcVolumeMount(v hooks.PVC) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      v.Name,
		MountPath: v.VolumePath,
	}
}

func gracePeriodInSeconds(vmi *v1.VirtualMachineInstance) int64 {
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		return *vmi.Spec.TerminationGracePeriodSeconds
	}
	return v1.DefaultGracePeriodSeconds
}

func sidecarContainerName(i int) string {
	return fmt.Sprintf("hook-sidecar-%d", i)
}

func (t *TemplateService) RenderHotplugAttachmentPodTemplate(volumes []*v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, claimMap map[string]*k8sv1.PersistentVolumeClaim) (*k8sv1.Pod, error) {
	zero := int64(0)
	runUser := int64(util.NonRootUID)
	sharedMount := k8sv1.MountPropagationHostToContainer
	command := []string{"/bin/sh", "-c", "/usr/bin/container-disk --copy-path /path/hp"}

	tolerations := append(hotplugPodTolerations(), ownerPod.Spec.Tolerations...)

	// Remove duplicates
	sort.Slice(tolerations, func(i, j int) bool {
		return tolerations[i].Key < tolerations[j].Key
	})
	tolerations = slices.Compact(tolerations)

	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hp-volume-",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ownerPod, schema.GroupVersionKind{
					Group:   k8sv1.SchemeGroupVersion.Group,
					Version: k8sv1.SchemeGroupVersion.Version,
					Kind:    "Pod",
				}),
			},
			Labels: map[string]string{
				v1.AppLabel: hotplugDisk,
			},
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				{
					Name:      hotplugDisk,
					Image:     t.launcherImage,
					Command:   command,
					Resources: hotplugContainerResourceRequirementsForVMI(t.clusterConfig),
					SecurityContext: &k8sv1.SecurityContext{
						AllowPrivilegeEscalation: pointer.P(false),
						RunAsNonRoot:             pointer.P(true),
						RunAsUser:                &runUser,
						SeccompProfile: &k8sv1.SeccompProfile{
							Type: k8sv1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &k8sv1.Capabilities{
							Drop: []k8sv1.Capability{"ALL"},
						},
						SELinuxOptions: &k8sv1.SELinuxOptions{
							// If SELinux is enabled on the host, this level will be adjusted below to match the level
							// of its companion virt-launcher pod to allow it to consume our disk images.
							Type:  t.clusterConfig.GetSELinuxLauncherType(),
							Level: "s0",
						},
					},
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:             hotplugDisks,
							MountPath:        "/path",
							MountPropagation: &sharedMount,
						},
					},
				},
			},
			Affinity: &k8sv1.Affinity{
				NodeAffinity: &k8sv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
						NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
							{
								MatchExpressions: []k8sv1.NodeSelectorRequirement{
									{
										Key:      k8sv1.LabelHostname,
										Operator: k8sv1.NodeSelectorOpIn,
										Values:   []string{ownerPod.Spec.NodeName},
									},
								},
							},
						},
					},
				},
			},
			Tolerations:                   tolerations,
			Volumes:                       []k8sv1.Volume{emptyDirVolume(hotplugDisks)},
			TerminationGracePeriodSeconds: &zero,
		},
	}

	err := matchSELinuxLevelOfVMI(pod, vmi)
	if err != nil {
		return nil, err
	}

	hotplugVolumeStatusMap := make(map[string]v1.VolumePhase)
	for _, status := range vmi.Status.VolumeStatus {
		if status.HotplugVolume != nil {
			hotplugVolumeStatusMap[status.Name] = status.Phase
		}
	}
	for _, volume := range volumes {
		claimName := types.PVCNameFromVirtVolume(volume)
		if claimName == "" {
			continue
		}
		skipMount := false
		if hotplugVolumeStatusMap[volume.Name] == v1.VolumeReady || hotplugVolumeStatusMap[volume.Name] == v1.HotplugVolumeMounted {
			skipMount = true
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
			Name: volume.Name,
			VolumeSource: k8sv1.VolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
			},
		})
		pvc := claimMap[volume.Name]
		if pvc == nil {
			continue
		}
		if types.IsPVCBlock(pvc.Spec.VolumeMode) {
			pod.Spec.Containers[0].VolumeDevices = append(pod.Spec.Containers[0].VolumeDevices, k8sv1.VolumeDevice{
				Name:       volume.Name,
				DevicePath: fmt.Sprintf("/path/%s/%s", volume.Name, pvc.GetUID()),
			})
		} else {
			if !skipMount {
				pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
					Name:      volume.Name,
					MountPath: fmt.Sprintf("/%s", volume.Name),
				})
			}
		}
	}

	return pod, nil
}

func (t *TemplateService) RenderHotplugAttachmentTriggerPodTemplate(volume *v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, pvcName string, isBlock bool, tempPod bool) (*k8sv1.Pod, error) {
	zero := int64(0)
	runUser := int64(util.NonRootUID)
	sharedMount := k8sv1.MountPropagationHostToContainer
	var command []string
	if tempPod {
		command = []string{"/bin/bash",
			"-c",
			"exit", "0"}
	} else {
		command = []string{"/bin/sh", "-c", "/usr/bin/container-disk --copy-path /path/hp"}
	}

	annotationsList := make(map[string]string)
	if tempPod {
		// mark pod as temp - only used for provisioning
		annotationsList[v1.EphemeralProvisioningObject] = "true"
	}

	tmpTolerations := make([]k8sv1.Toleration, len(ownerPod.Spec.Tolerations))
	copy(tmpTolerations, ownerPod.Spec.Tolerations)

	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hp-volume-",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ownerPod, schema.GroupVersionKind{
					Group:   k8sv1.SchemeGroupVersion.Group,
					Version: k8sv1.SchemeGroupVersion.Version,
					Kind:    "Pod",
				}),
			},
			Labels: map[string]string{
				v1.AppLabel: hotplugDisk,
			},
			Annotations: annotationsList,
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				{
					Name:      hotplugDisk,
					Image:     t.launcherImage,
					Command:   command,
					Resources: hotplugContainerResourceRequirementsForVMI(t.clusterConfig),
					SecurityContext: &k8sv1.SecurityContext{
						AllowPrivilegeEscalation: pointer.P(false),
						RunAsNonRoot:             pointer.P(true),
						RunAsUser:                &runUser,
						SeccompProfile: &k8sv1.SeccompProfile{
							Type: k8sv1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &k8sv1.Capabilities{
							Drop: []k8sv1.Capability{"ALL"},
						},
						SELinuxOptions: &k8sv1.SELinuxOptions{
							Level: "s0",
						},
					},
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:             hotplugDisks,
							MountPath:        "/path",
							MountPropagation: &sharedMount,
						},
					},
				},
			},
			Affinity: &k8sv1.Affinity{
				PodAffinity: &k8sv1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: ownerPod.GetLabels(),
							},
							TopologyKey: k8sv1.LabelHostname,
						},
					},
				},
			},
			Tolerations: tmpTolerations,
			Volumes: []k8sv1.Volume{
				{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
							ReadOnly:  false,
						},
					},
				},
				emptyDirVolume(hotplugDisks),
			},
			TerminationGracePeriodSeconds: &zero,
		},
	}

	err := matchSELinuxLevelOfVMI(pod, vmi)
	if err != nil {
		return nil, err
	}

	if isBlock {
		pod.Spec.Containers[0].VolumeDevices = []k8sv1.VolumeDevice{
			{
				Name:       volume.Name,
				DevicePath: "/dev/hotplugblockdevice",
			},
		}
		pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{
			RunAsUser: &[]int64{0}[0],
		}
	} else {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
			Name:      volume.Name,
			MountPath: "/pvc",
		})
	}
	return pod, nil
}

func (t *TemplateService) RenderExporterManifest(vmExport *exportv1.VirtualMachineExport, namePrefix string) *k8sv1.Pod {
	exporterPod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			// Use of DNS1035LabelMaxLength here to align with
			// VMExportController{}.getExportPodName
			Name:      naming.GetName(namePrefix, vmExport.Name, validation.DNS1035LabelMaxLength),
			Namespace: vmExport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmExport, schema.GroupVersionKind{
					Group:   exportv1.SchemeGroupVersion.Group,
					Version: exportv1.SchemeGroupVersion.Version,
					Kind:    "VirtualMachineExport",
				}),
			},
			Labels: map[string]string{
				v1.AppLabel: virtExporter,
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:            "exporter",
					Image:           t.exporterImage,
					ImagePullPolicy: t.clusterConfig.GetImagePullPolicy(),
					Env: []k8sv1.EnvVar{
						{
							Name: "POD_NAME",
							ValueFrom: &k8sv1.EnvVarSource{
								FieldRef: &k8sv1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
					},
					SecurityContext: &k8sv1.SecurityContext{
						AllowPrivilegeEscalation: pointer.P(false),
						Capabilities:             &k8sv1.Capabilities{Drop: []k8sv1.Capability{"ALL"}},
					},
					Resources: vmExportContainerResourceRequirements(t.clusterConfig),
				},
			},
		},
	}
	return exporterPod
}

func appendUniqueImagePullSecret(secrets []k8sv1.LocalObjectReference, newsecret k8sv1.LocalObjectReference) []k8sv1.LocalObjectReference {
	for _, oldsecret := range secrets {
		if oldsecret == newsecret {
			return secrets
		}
	}
	return append(secrets, newsecret)
}

func HaveContainerDiskVolume(volumes []v1.Volume) bool {
	for _, volume := range volumes {
		if volume.ContainerDisk != nil {
			return true
		}
	}
	return false
}

type templateServiceOption func(*TemplateService)

func NewTemplateService(launcherImage string,
	launcherQemuTimeout int,
	virtShareDir string,
	ephemeralDiskDir string,
	containerDiskDir string,
	hotplugDiskDir string,
	imagePullSecret string,
	persistentVolumeClaimCache cache.Store,
	persistentVolumeCache cache.Store,
	virtClient kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
	launcherSubGid int64,
	exporterImage string,
	resourceQuotaStore cache.Store,
	namespaceStore cache.Store,
	opts ...templateServiceOption,
) *TemplateService {

	precond.MustNotBeEmpty(launcherImage)
	log.Log.V(1).Infof("Exporter Image: %s", exporterImage)
	svc := TemplateService{
		launcherImage:               launcherImage,
		launcherQemuTimeout:         launcherQemuTimeout,
		virtShareDir:                virtShareDir,
		ephemeralDiskDir:            ephemeralDiskDir,
		containerDiskDir:            containerDiskDir,
		hotplugDiskDir:              hotplugDiskDir,
		imagePullSecret:             imagePullSecret,
		persistentVolumeClaimStore:  persistentVolumeClaimCache,
		persistentVolumeStore:       persistentVolumeCache,
		virtClient:                  virtClient,
		clusterConfig:               clusterConfig,
		launcherSubGid:              launcherSubGid,
		exporterImage:               exporterImage,
		resourceQuotaStore:          resourceQuotaStore,
		namespaceStore:              namespaceStore,
		launcherHypervisorResources: hypervisor.NewLauncherHypervisorResources(clusterConfig.GetHypervisor().Name),
	}

	for _, opt := range opts {
		opt(&svc)
	}

	return &svc
}

func copyProbe(probe *v1.Probe) *k8sv1.Probe {
	if probe == nil {
		return nil
	}
	return &k8sv1.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
		ProbeHandler: k8sv1.ProbeHandler{
			Exec:      probe.Exec,
			HTTPGet:   probe.HTTPGet,
			TCPSocket: probe.TCPSocket,
		},
	}
}

func wrapGuestAgentPingWithVirtProbe(vmi *v1.VirtualMachineInstance, probe *k8sv1.Probe) {
	pingCommand := []string{
		"virt-probe",
		"--domainName", api.VMINamespaceKeyFunc(vmi),
		"--timeoutSeconds", strconv.FormatInt(int64(probe.TimeoutSeconds), 10),
		"--guestAgentPing",
	}
	probe.ProbeHandler.Exec = &k8sv1.ExecAction{Command: pingCommand}
	// we add 1s to the pod probe to compensate for the additional steps in probing
	probe.TimeoutSeconds += 1
	return
}

func alignPodMultiCategorySecurity(pod *k8sv1.Pod, selinuxType string, dockerSELinuxMCSWorkaround bool) {
	if selinuxType == "" && !dockerSELinuxMCSWorkaround {
		// No SELinux type and no docker workaround, nothing to do
		return
	}

	if selinuxType != "" {
		if pod.Spec.SecurityContext == nil {
			pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{}
		}
		pod.Spec.SecurityContext.SELinuxOptions = &k8sv1.SELinuxOptions{Type: selinuxType}
	}

	if dockerSELinuxMCSWorkaround {
		// more info on https://github.com/kubernetes/kubernetes/issues/90759
		// Since the compute container needs to be able to communicate with the
		// rest of the pod, we loop over all the containers and remove their SELinux
		// categories.
		// This currently only affects Docker + SELinux use-cases, and requires a
		// feature gate to be set.
		for i := range pod.Spec.Containers {
			container := &pod.Spec.Containers[i]
			if container.Name != "compute" {
				generateContainerSecurityContext(selinuxType, container)
			}
		}
	}
}

func matchSELinuxLevelOfVMI(pod *k8sv1.Pod, vmi *v1.VirtualMachineInstance) error {
	if vmi.Status.SelinuxContext == "" {
		if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.SourceState != nil && vmi.Status.MigrationState.SourceState.SelinuxContext != "" {
			selinuxContext := vmi.Status.MigrationState.SourceState.SelinuxContext
			if selinuxContext != "none" {
				return setSELinuxContext(selinuxContext, pod)
			}
			return nil
		}
		return fmt.Errorf("VMI is missing SELinux context")
	} else if vmi.Status.SelinuxContext != "none" {
		return setSELinuxContext(vmi.Status.SelinuxContext, pod)
	}

	return nil
}

func setSELinuxContext(selinuxContext string, pod *k8sv1.Pod) error {
	ctx := strings.Split(selinuxContext, ":")
	if len(ctx) < 4 {
		return fmt.Errorf("VMI has invalid SELinux context: %s", selinuxContext)
	}
	pod.Spec.Containers[0].SecurityContext.SELinuxOptions.Level = strings.Join(ctx[3:], ":")
	return nil
}

func generateContainerSecurityContext(selinuxType string, container *k8sv1.Container) {
	if container.SecurityContext == nil {
		container.SecurityContext = &k8sv1.SecurityContext{}
	}
	if container.SecurityContext.SELinuxOptions == nil {
		container.SecurityContext.SELinuxOptions = &k8sv1.SELinuxOptions{}
	}
	container.SecurityContext.SELinuxOptions.Type = selinuxType
	container.SecurityContext.SELinuxOptions.Level = "s0"
}

func (t *TemplateService) generatePodAnnotations(vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	annotationsSet := map[string]string{
		v1.DomainAnnotation: vmi.GetObjectMeta().GetName(),
	}
	maps.Copy(annotationsSet, filterVMIAnnotationsForPod(vmi.Annotations))

	annotationsSet[podcmd.DefaultContainerAnnotationName] = "compute"

	// Set this annotation now to indicate that the newly created virt-launchers will use
	// unix sockets as a transport for migration
	annotationsSet[v1.MigrationTransportUnixAnnotation] = "true"
	annotationsSet[descheduler.EvictOnlyAnnotation] = ""

	for _, generator := range t.annotationsGenerators {
		annotations, err := generator.Generate(vmi)
		if err != nil {
			return nil, err
		}

		maps.Copy(annotationsSet, annotations)
	}

	return annotationsSet, nil
}

func filterVMIAnnotationsForPod(vmiAnnotations map[string]string) map[string]string {
	annotationsList := map[string]string{}
	for k, v := range vmiAnnotations {
		if strings.HasPrefix(k, "kubectl.kubernetes.io") ||
			strings.HasPrefix(k, "kubevirt.io/storage-observed-api-version") ||
			strings.HasPrefix(k, "kubevirt.io/latest-observed-api-version") {
			continue
		}
		annotationsList[k] = v
	}
	return annotationsList
}

func checkForKeepLauncherAfterFailure(vmi *v1.VirtualMachineInstance) bool {
	keepLauncherAfterFailure := false
	for k, v := range vmi.Annotations {
		if strings.HasPrefix(k, v1.KeepLauncherAfterFailureAnnotation) {
			if v == "" || strings.HasPrefix(v, "true") {
				keepLauncherAfterFailure = true
				break
			}
		}
	}
	return keepLauncherAfterFailure
}

func (t *TemplateService) doesVMIRequireAutoCPULimits(vmi *v1.VirtualMachineInstance) bool {
	if t.doesVMIRequireAutoResourceLimits(vmi, k8sv1.ResourceCPU) {
		return true
	}

	labelSelector := t.clusterConfig.GetConfig().AutoCPULimitNamespaceLabelSelector
	_, limitSet := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]
	if labelSelector == nil || limitSet {
		return false
	}
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		log.DefaultLogger().Reason(err).Warning("invalid CPULimitNamespaceLabelSelector set, assuming none")
		return false
	}

	if t.namespaceStore == nil {
		log.DefaultLogger().Reason(err).Warning("empty namespace informer")
		return false
	}

	obj, exists, err := t.namespaceStore.GetByKey(vmi.Namespace)
	if err != nil {
		log.Log.Warning("Error retrieving namespace from informer")
		return false
	} else if !exists {
		log.Log.Warningf("namespace %s does not exist.", vmi.Namespace)
		return false
	}

	ns, ok := obj.(*k8sv1.Namespace)
	if !ok {
		log.Log.Errorf("couldn't cast object to Namespace: %+v", obj)
		return false
	}

	if selector.Matches(labels.Set(ns.Labels)) {
		return true
	}

	return false
}

func (t *TemplateService) VMIResourcePredicates(vmi *v1.VirtualMachineInstance, networkToResourceMap map[string]string, memoryOverhead resource.Quantity) VMIResourcePredicates {
	withCPULimits := t.doesVMIRequireAutoCPULimits(vmi)
	additionalCPUs := uint32(0)
	if vmi.Spec.Domain.IOThreadsPolicy != nil &&
		*vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool &&
		vmi.Spec.Domain.IOThreads != nil &&
		vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
		additionalCPUs = *vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount
	}
	return VMIResourcePredicates{
		vmi: vmi,
		resourceRules: []VMIResourceRule{
			// Run overcommit first to avoid overcommitting overhead memory
			NewVMIResourceRule(emptyMemoryRequest, WithMemoryRequests(vmi.Spec.Domain.Memory, t.clusterConfig.GetMemoryOvercommit())),
			NewVMIResourceRule(doesVMIRequireDedicatedCPU, WithCPUPinning(vmi, vmi.Annotations, additionalCPUs)),
			NewVMIResourceRule(not(doesVMIRequireDedicatedCPU), WithoutDedicatedCPU(vmi, t.clusterConfig.GetCPUAllocationRatio(), withCPULimits)),
			NewVMIResourceRule(hasHugePages, WithHugePages(vmi.Spec.Domain.Memory, memoryOverhead)),
			NewVMIResourceRule(not(hasHugePages), WithMemoryOverhead(vmi.Spec.Domain.Resources, memoryOverhead)),
			NewVMIResourceRule(t.doesVMIRequireAutoMemoryLimits, WithAutoMemoryLimits(vmi.Namespace, t.namespaceStore)),
			NewVMIResourceRule(func(*v1.VirtualMachineInstance) bool {
				return len(networkToResourceMap) > 0
			}, WithNetworkResources(networkToResourceMap)),
			NewVMIResourceRule(isGPUVMIDevicePlugins, WithGPUsDevicePlugins(vmi.Spec.Domain.Devices.GPUs)),
			NewVMIResourceRule(func(vmi *v1.VirtualMachineInstance) bool {
				return t.clusterConfig.GPUsWithDRAGateEnabled() && isGPUVMIDRA(vmi)
			}, WithGPUsDRA(vmi.Spec.Domain.Devices.GPUs)),
			NewVMIResourceRule(isHostDevVMIDevicePlugins, WithHostDevicesDevicePlugins(vmi.Spec.Domain.Devices.HostDevices)),
			NewVMIResourceRule(func(vmi *v1.VirtualMachineInstance) bool {
				return t.clusterConfig.HostDevicesWithDRAEnabled() && isHostDevVMIDRA(vmi)
			}, WithHostDevicesDRA(vmi.Spec.Domain.Devices.HostDevices)),
			NewVMIResourceRule(func(vmi *v1.VirtualMachineInstance) bool {
				return t.clusterConfig.NetworkDevicesWithDRAGateEnabled() && vmispec.HasDRANetwork(vmi.Spec.Networks)
			}, WithNetworksDRA(vmi.Spec.Networks)),
			NewVMIResourceRule(util.IsSEVVMI, WithSEV()),
			NewVMIResourceRule(util.IsTDXVMI, WithTDX()),
			NewVMIResourceRule(reservation.HasVMIPersistentReservation, WithPersistentReservation()),
			NewVMIResourceRule(func(vmi *v1.VirtualMachineInstance) bool {
				return t.clusterConfig.IOMMUFDEnabled()
			}, WithIOMMUFD()),
		},
	}
}

// TODO: Make this function private (calculateMemoryOverhead) once VmiMemoryOverheadReport feature gate is GA
// and we are sure that all VMIs include the MemoryOverhead status field
func CalculateMemoryOverhead(clusterConfig *virtconfig.ClusterConfig, netMemoryCalculator netMemoryCalculator, vmi *v1.VirtualMachineInstance, launcherHypervisorResources hypervisor.LauncherHypervisorResources) resource.Quantity {
	// Set default with vmi Architecture. compatible with multi-architecture hybrid environments
	vmiCPUArch := vmi.Spec.Architecture
	if vmiCPUArch == "" {
		vmiCPUArch = clusterConfig.GetClusterCPUArch()
	}

	memoryOverhead := launcherHypervisorResources.GetMemoryOverhead(vmi, vmiCPUArch, clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio)

	if netMemoryCalculator != nil {
		memoryOverhead.Add(
			netMemoryCalculator.Calculate(vmi, clusterConfig.GetNetworkBindings()),
		)
	}

	return memoryOverhead
}

func (t *TemplateService) doesVMIRequireAutoMemoryLimits(vmi *v1.VirtualMachineInstance) bool {
	return t.doesVMIRequireAutoResourceLimits(vmi, k8sv1.ResourceMemory)
}

func (t *TemplateService) doesVMIRequireAutoResourceLimits(vmi *v1.VirtualMachineInstance, resource k8sv1.ResourceName) bool {
	if _, resourceLimitsExists := vmi.Spec.Domain.Resources.Limits[resource]; resourceLimitsExists {
		return false
	}

	for _, obj := range t.resourceQuotaStore.List() {
		if resourceQuota, ok := obj.(*k8sv1.ResourceQuota); ok {
			if _, exists := resourceQuota.Spec.Hard["limits."+resource]; exists && resourceQuota.Namespace == vmi.Namespace {
				return true
			}
		}
	}

	return false
}

func (p VMIResourcePredicates) Apply() []ResourceRendererOption {
	var options []ResourceRendererOption
	for _, rule := range p.resourceRules {
		if rule.predicate(p.vmi) {
			options = append(options, rule.option)
		}
	}
	return options
}

func podLabels(vmi *v1.VirtualMachineInstance, hostName string) map[string]string {
	labels := map[string]string{}

	for k, v := range vmi.Labels {
		labels[k] = v
	}
	labels[v1.AppLabel] = "virt-launcher"
	labels[v1.CreatedByLabel] = string(vmi.UID)
	labels[v1.DeprecatedVirtualMachineNameLabel] = hostName
	labels[v1.VirtualMachineInstanceIDLabel] = apimachinery.CalculateVirtualMachineInstanceID(vmi.Name)
	if val, exists := vmi.Annotations[istio.InjectSidecarAnnotation]; exists {
		labels[istio.InjectSidecarLabel] = val
	}
	return labels
}

func readinessGates() []k8sv1.PodReadinessGate {
	return []k8sv1.PodReadinessGate{
		{
			ConditionType: v1.VirtualMachineUnpaused,
		},
	}
}

func WithNetMemoryCalculator(netMemoryCalculator netMemoryCalculator) templateServiceOption {
	return func(service *TemplateService) {
		service.netMemoryCalculator = netMemoryCalculator
	}
}

func WithAnnotationsGenerators(generators ...annotationsGenerator) templateServiceOption {
	return func(service *TemplateService) {
		service.annotationsGenerators = append(service.annotationsGenerators, generators...)
	}
}

func WithNetTargetAnnotationsGenerator(generator targetAnnotationsGenerator) templateServiceOption {
	return func(service *TemplateService) {
		service.netTargetAnnotationsGenerator = generator
	}
}

func hasHugePages(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil
}

// isGPUVMIDevicePlugins checks if a VMI has any GPUs configured for device plugins
func isGPUVMIDevicePlugins(vmi *v1.VirtualMachineInstance) bool {
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if isGPUDevicePlugin(gpu) {
			return true
		}
	}
	return false
}

func isGPUDevicePlugin(gpu v1.GPU) bool {
	return gpu.DeviceName != "" && gpu.ClaimRequest == nil
}

// isGPUVMIDRA checks if a VMI has any GPUs configured for Dynamic Resource Allocation
func isGPUVMIDRA(vmi *v1.VirtualMachineInstance) bool {
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if drautil.IsGPUDRA(gpu) {
			return true
		}
	}
	return false
}

// isHostDevVMIDevicePlugins checks if a VMI has any HostDevices configured for device plugins
func isHostDevVMIDevicePlugins(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.HostDevices == nil {
		return false
	}

	for _, hostDev := range vmi.Spec.Domain.Devices.HostDevices {
		if hostDev.DeviceName != "" && hostDev.ClaimRequest == nil {
			return true
		}
	}

	return false
}

// isHostDevVMIDRA checks if a VMI has any HostDevices configured for Dynamic Resource Allocation
func isHostDevVMIDRA(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.HostDevices == nil {
		return false
	}

	for _, hostDev := range vmi.Spec.Domain.Devices.HostDevices {
		if hostDev.DeviceName == "" && hostDev.ClaimRequest != nil {
			return true
		}
	}

	return false
}

func emptyMemoryRequest(vmi *v1.VirtualMachineInstance) bool {
	resources := &vmi.Spec.Domain.Resources
	return resources.Requests.Memory().IsZero()
}
