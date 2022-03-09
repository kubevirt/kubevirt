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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package clone

import (
	"fmt"

	clone "kubevirt.io/api/clone/v1alpha1"
	k6tv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

type cloneSourceType string

const (
	sourceTypeVM cloneSourceType = "VirtualMachine"
)

type cloneTargetType string

const (
	targetTypeVM cloneTargetType = "VirtualMachine"
)

func (ctrl *VMCloneController) execute(key string) error {
	logger := log.Log

	obj, cloneExists, err := ctrl.vmCloneInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}

	var vmClone *clone.VirtualMachineClone
	if cloneExists {
		vmClone = obj.(*clone.VirtualMachineClone)
		logger = logger.Object(vmClone)
	} else {
		return nil
	}

	sourceInfo := vmClone.Spec.Source
	switch cloneSourceType(sourceInfo.Kind) {
	case sourceTypeVM:
		vmKey := getKey(sourceInfo.Name, vmClone.Namespace)
		obj, vmExists, err := ctrl.vmInformer.GetStore().GetByKey(vmKey)
		if err != nil {
			return fmt.Errorf("error getting VM %s in namespace %s from cache: %v", sourceInfo.Name, vmClone.Namespace, err)
		}
		if !vmExists {
			return fmt.Errorf("VM %s in namespace %s does not exist", sourceInfo.Name, vmClone.Namespace)
		}
		sourceVM := obj.(*k6tv1.VirtualMachine)

		err = ctrl.syncSourceVM(key, sourceVM, vmClone)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("clone %s is defined with an unknown source type %s", vmClone.Name, sourceInfo.Kind)
	}

	err = ctrl.updateStatus(vmClone)
	if err != nil {
		return err
	}

	return nil
}

func (ctrl *VMCloneController) syncSourceVM(key string, source *k6tv1.VirtualMachine, vmClone *clone.VirtualMachineClone) error {
	targetType := cloneTargetType(vmClone.Spec.Target.Kind)

	switch targetType {
	case targetTypeVM:
		return ctrl.syncSourceVMTargetVM(key, source, vmClone)

	default:
		return fmt.Errorf("target type is unknown: %s", targetType)
	}
}

func (ctrl *VMCloneController) syncSourceVMTargetVM(key string, source *k6tv1.VirtualMachine, vmClone *clone.VirtualMachineClone) error {
	return nil
}

func (ctrl *VMCloneController) updateStatus(vmClone *clone.VirtualMachineClone) error {
	return nil
}
