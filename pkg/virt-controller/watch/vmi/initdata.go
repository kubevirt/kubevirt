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

package vmi

import (
	"encoding/json"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util"
)

// addInitData handles the addition of an InitData, enqueuing affected VMIs.
func (c *Controller) addInitData(obj interface{}) {
	initData := obj.(*v1.InitData)
	if initData.DeletionTimestamp != nil {
		return
	}
	vmis := c.listVMIsMatchingInitData(initData.Namespace, initData.Name)
	for _, vmi := range vmis {
		log.Log.V(4).Infof("InitData %s/%s created for vmi %s", initData.Namespace, initData.Name, vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

// deleteInitData handles the deletion of an InitData, enqueuing affected VMIs.
func (c *Controller) deleteInitData(obj interface{}) {
	initData, ok := obj.(*v1.InitData)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Log.Reason(fmt.Errorf(tombstoneGetObjectErrFmt, obj)).Error(deleteNotifFailed)
			return
		}
		initData, ok = tombstone.Obj.(*v1.InitData)
		if !ok {
			log.Log.Reason(fmt.Errorf("tombstone contained object that is not an InitData %#v", obj)).Error(deleteNotifFailed)
			return
		}
	}
	vmis := c.listVMIsMatchingInitData(initData.Namespace, initData.Name)
	for _, vmi := range vmis {
		log.Log.V(4).Infof("InitData %s/%s deleted for vmi %s", initData.Namespace, initData.Name, vmi.Name)
		c.enqueueVirtualMachine(vmi)
	}
}

// listVMIsMatchingInitData finds all VMIs that reference the given InitData name.
func (c *Controller) listVMIsMatchingInitData(namespace, name string) []*v1.VirtualMachineInstance {
	var result []*v1.VirtualMachineInstance
	objs := c.vmiIndexer.List()
	for _, obj := range objs {
		vmi := obj.(*v1.VirtualMachineInstance)
		if vmi.Namespace != namespace {
			continue
		}
		ref, hasRef := util.HasInitDataRef(vmi)
		if hasRef && ref == name {
			result = append(result, vmi.DeepCopy())
		}
	}
	return result
}

// getInitData returns the InitData CR for the given VMI, or nil if not found
// or not needed. Returns (initData, ready).
func (c *Controller) getInitData(vmi *v1.VirtualMachineInstance) (*v1.InitData, bool) {
	if !c.clusterConfig.InjectInitDataEnabled() {
		return nil, true
	}

	initDataRef, hasRef := util.HasInitDataRef(vmi)
	if !hasRef {
		return nil, true
	}

	key := vmi.Namespace + "/" + initDataRef
	obj, exists, err := c.initDataIndexer.GetByKey(key)
	if err != nil || !exists {
		log.Log.Object(vmi).V(4).Infof("InitData %s not found yet, waiting", key)
		c.recorder.Eventf(vmi, k8sv1.EventTypeNormal, controller.InitDataNotFoundReason,
			"InitData %s does not exist, waiting for it to appear", key)
		return nil, false
	}

	initData := obj.(*v1.InitData)

	if initData.Spec.MRConfigId == "" && initData.Spec.HostData == "" {
		log.Log.Object(vmi).Warningf("InitData %s has neither mrConfigId nor hostData", key)
		return nil, false
	}

	return initData, true
}

// injectInitDataEnvVars adds InitData values as environment variables to the
// first container (compute) of the pod template.
func injectInitDataEnvVars(pod *k8sv1.Pod, initData *v1.InitData) {
	if initData == nil || len(pod.Spec.Containers) == 0 {
		return
	}

	if initData.Spec.MRConfigId != "" {
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, k8sv1.EnvVar{
			Name:  v1.InitDataMRConfigIdEnvVar,
			Value: initData.Spec.MRConfigId,
		})
	}
	if initData.Spec.HostData != "" {
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, k8sv1.EnvVar{
			Name:  v1.InitDataHostDataEnvVar,
			Value: initData.Spec.HostData,
		})
	}
	if len(initData.Spec.OEMStrings) > 0 {
		oemJSON, _ := json.Marshal(initData.Spec.OEMStrings)
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, k8sv1.EnvVar{
			Name:  v1.InitDataOEMStringsEnvVar,
			Value: string(oemJSON),
		})
	}
}

// updateInitDataCondition sets the InitDataReady condition on the VMI status.
func updateInitDataCondition(vmi *v1.VirtualMachineInstance, vmiCopy *v1.VirtualMachineInstance, initDataIndexer cache.Indexer, clusterConfig clusterConfigProvider) {
	if !util.HasInitDataRefBool(vmi) {
		return
	}

	if clusterConfig == nil || !clusterConfig.InjectInitDataEnabled() {
		return
	}

	conditionManager := controller.NewVirtualMachineInstanceConditionManager()

	initDataRef, _ := util.HasInitDataRef(vmi)
	key := vmi.Namespace + "/" + initDataRef
	_, exists, err := initDataIndexer.GetByKey(key)

	condition := v1.VirtualMachineInstanceCondition{
		Type: v1.VirtualMachineInstanceInitDataReady,
	}

	if err == nil && exists {
		condition.Status = k8sv1.ConditionTrue
		condition.Reason = v1.VirtualMachineInstanceReasonInitDataReady
		condition.Message = "InitData CR resolved"
	} else {
		condition.Status = k8sv1.ConditionFalse
		condition.Reason = v1.VirtualMachineInstanceReasonInitDataNotFound
		condition.Message = "InitData CR not yet available"
	}

	conditionManager.UpdateCondition(vmiCopy, &condition)
}

type clusterConfigProvider interface {
	InjectInitDataEnabled() bool
}
