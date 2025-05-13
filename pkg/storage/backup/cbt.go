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

package backup

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func vmMatchesChangedBlockTrackingSelectors(vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig, nsStore cache.Store) bool {
	labelSelectors := clusterConfig.GetConfig().ChangedBlockTrackingLabelSelectors
	if labelSelectors == nil {
		return false
	}
	logger := log.Log.Object(vm)
	vmSelector := labelSelectors.VirtualMachineLabelSelector
	namespaceSelector := labelSelectors.NamespaceLabelSelector

	return vmMatchesVMSelector(vmSelector, vm.Labels, logger) ||
		vmMatchesNamespaceSelector(namespaceSelector, vm.Namespace, nsStore, logger)
}

func vmMatchesVMSelector(labelSelector *metav1.LabelSelector, vmLabels map[string]string, logger *log.FilteredLogger) bool {
	if labelSelector == nil {
		return false
	}
	vmSelector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		logger.Reason(err).Warning("invalid changedBlockTracking virtualMachineSelector set, assuming none")
		return false
	}

	return vmSelector.Matches(labels.Set(vmLabels))
}

func vmMatchesNamespaceSelector(labelSelector *metav1.LabelSelector, namespace string, nsStore cache.Store, logger *log.FilteredLogger) bool {
	if labelSelector == nil {
		return false
	}
	nsSelector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		logger.Reason(err).Warning("invalid changedBlockTracking namespaceSelector set, assuming none")
		return false
	}
	if nsStore == nil {
		logger.Warning("empty namespace informer")
		return false
	}

	obj, exists, err := nsStore.GetByKey(namespace)
	if err != nil {
		logger.Warning("Error retrieving vm namespace from informer")
		return false
	}
	if !exists {
		logger.Warningf("namespace %s does not exist.", namespace)
		return false
	}

	ns, ok := obj.(*k8sv1.Namespace)
	if !ok {
		log.Log.Errorf("couldn't cast object to Namespace: %+v", obj)
		return false
	}

	return nsSelector.Matches(labels.Set(ns.Labels))
}

func SyncVMChangedBlockTrackingState(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, clusterConfig *virtconfig.ClusterConfig, nsStore cache.Store) {
	vmMatchesSelector := vmMatchesChangedBlockTrackingSelectors(vm, clusterConfig, nsStore)

	if vmMatchesSelector {
		if vmi != nil {
			enableChangedBlockTrackingVMIExists(vm, vmi)
		} else {
			enableChangedBlockTrackingNoVMI(vm)
		}
	} else {
		disableChangedBlockTracking(vm, vmi)
	}
}

func enableChangedBlockTrackingVMIExists(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	switch vm.Status.ChangedBlockTracking {
	case "":
		vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingPendingRestart
	case v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingDisabled:
		switch vmi.Status.ChangedBlockTracking {
		case v1.ChangedBlockTrackingInitializing:
			vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingInitializing
		case v1.ChangedBlockTrackingEnabled:
			vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingEnabled
		default:
			vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingPendingRestart
		}
	case v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingEnabled:
		switch vmi.Status.ChangedBlockTracking {
		case v1.ChangedBlockTrackingEnabled:
			vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingEnabled
		default:
			vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingInitializing
		}
	default:
		log.Log.Object(vm).Warning("invalid changedBlockTracking state, removing state")
		vm.Status.ChangedBlockTracking = ""
	}
}

func enableChangedBlockTrackingNoVMI(vm *v1.VirtualMachine) {
	switch vm.Status.ChangedBlockTracking {
	case "", v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingDisabled:
		vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingInitializing
	case v1.ChangedBlockTrackingEnabled:
		vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingEnabled
	default:
		log.Log.Object(vm).Warning("invalid changedBlockTracking state, removing state")
		vm.Status.ChangedBlockTracking = ""
	}
}

func disableChangedBlockTracking(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	cbtState := vm.Status.ChangedBlockTracking
	if cbtState == "" {
		return
	}

	if vmi == nil || cbtState == v1.ChangedBlockTrackingDisabled {
		vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingDisabled
		return
	}

	switch cbtState {
	case v1.ChangedBlockTrackingPendingRestart, v1.ChangedBlockTrackingInitializing, v1.ChangedBlockTrackingEnabled:
		if vmi.Status.ChangedBlockTracking != "" {
			vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingPendingRestart
		} else {
			vm.Status.ChangedBlockTracking = v1.ChangedBlockTrackingDisabled
		}
	default:
		log.Log.Object(vm).Warning("invalid changedBlockTracking state, removing state")
		vm.Status.ChangedBlockTracking = ""
	}
}
