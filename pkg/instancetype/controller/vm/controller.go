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
package vm

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/annotations"
	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/expand"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceannotations "kubevirt.io/kubevirt/pkg/instancetype/preference/annotations"
	preferenceapply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
	preferencefind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/instancetype/upgrade"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

type applyVMHandler interface {
	ApplyToVM(*virtv1.VirtualMachine) error
}

type instancetypeFindHandler interface {
	Find(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
}

type preferenceFindHandler interface {
	FindPreference(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

type expandHandler interface {
	Expand(*virtv1.VirtualMachine) (*virtv1.VirtualMachine, error)
}

type storeHandler interface {
	Store(*virtv1.VirtualMachine) error
}

type upgradeHandler interface {
	Upgrade(*virtv1.VirtualMachine) error
}

type controller struct {
	applyVMHandler
	storeHandler
	expandHandler
	upgradeHandler
	instancetypeFindHandler
	preferenceFindHandler

	clientset     kubecli.KubevirtClient
	clusterConfig *virtconfig.ClusterConfig
	recorder      record.EventRecorder
}

func New(
	instancetypeStore, clusterInstancetypeStore, preferenceStore, clusterPreferenceStore, revisionStore cache.Store,
	virtClient kubecli.KubevirtClient, clusterConfig *virtconfig.ClusterConfig, recorder record.EventRecorder,
) *controller {
	finder := find.NewSpecFinder(instancetypeStore, clusterInstancetypeStore, revisionStore, virtClient)
	prefFinder := preferencefind.NewSpecFinder(preferenceStore, clusterPreferenceStore, revisionStore, virtClient)
	return &controller{
		instancetypeFindHandler: finder,
		preferenceFindHandler:   prefFinder,
		applyVMHandler:          apply.NewVMApplier(finder, prefFinder),
		storeHandler:            revision.New(instancetypeStore, clusterInstancetypeStore, preferenceStore, clusterPreferenceStore, virtClient),
		expandHandler:           expand.New(clusterConfig, finder, prefFinder),
		upgradeHandler:          upgrade.New(revisionStore, virtClient),
		clientset:               virtClient,
		clusterConfig:           clusterConfig,
		recorder:                recorder,
	}
}

const (
	storeControllerRevisionErrFmt   = "error encountered while storing instancetype.kubevirt.io controllerRevisions: %v"
	upgradeControllerRevisionErrFmt = "error encountered while upgrading instancetype.kubevirt.io controllerRevisions: %v"
	cleanControllerRevisionErrFmt   = "error encountered cleaning controllerRevision %s after successfully expanding VirtualMachine %s: %v"
)

func (c *controller) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return vm, nil
	}

	// Before we sync ensure any referenced resources exist
	if syncErr := c.checkResourcesExist(vm); syncErr != nil {
		return vm, syncErr
	}

	referencePolicy := c.clusterConfig.GetInstancetypeReferencePolicy()
	switch referencePolicy {
	case virtv1.Reference:
		// Ensure we have controllerRevisions of any instancetype or preferences referenced by the VM
		if err := c.Store(vm); err != nil {
			log.Log.Object(vm).Errorf(storeControllerRevisionErrFmt, err)
			c.recorder.Eventf(vm, corev1.EventTypeWarning, common.FailedCreateVirtualMachineReason, storeControllerRevisionErrFmt, err)
			return vm, common.NewSyncError(fmt.Errorf(storeControllerRevisionErrFmt, err), common.FailedCreateVirtualMachineReason)
		}
	case virtv1.Expand, virtv1.ExpandAll:
		return c.handleExpand(vm, referencePolicy)
	}

	// If we have controllerRevisions make sure they are fully up to date before proceeding
	if err := c.Upgrade(vm); err != nil {
		log.Log.Object(vm).Reason(err).Errorf(upgradeControllerRevisionErrFmt, err)
		c.recorder.Eventf(vm, corev1.EventTypeWarning, common.FailedCreateVirtualMachineReason, upgradeControllerRevisionErrFmt, err)
		return vm, common.NewSyncError(fmt.Errorf(upgradeControllerRevisionErrFmt, err), common.FailedCreateVirtualMachineReason)
	}
	return vm, nil
}

func (c *controller) checkResourcesExist(vm *virtv1.VirtualMachine) error {
	const (
		failedFindInstancetype = "FailedFindInstancetype"
		failedFindPreference   = "FailedFindPreference"
	)
	if _, err := c.Find(vm); err != nil {
		return common.NewSyncError(err, failedFindInstancetype)
	}
	if _, err := c.FindPreference(vm); err != nil {
		return common.NewSyncError(err, failedFindPreference)
	}
	return nil
}

func (c *controller) handleExpand(
	vm *virtv1.VirtualMachine,
	referencePolicy virtv1.InstancetypeReferencePolicy,
) (*virtv1.VirtualMachine, error) {
	if referencePolicy == virtv1.Expand {
		if revision.HasControllerRevisionRef(vm.Status.InstancetypeRef) {
			log.Log.Object(vm).Infof("not expanding as instance type already has revisionName")
			return vm, nil
		}
		if revision.HasControllerRevisionRef(vm.Status.PreferenceRef) {
			log.Log.Object(vm).Infof("not expanding as preference already has revisionName")
			return vm, nil
		}
	}

	expandVMCopy, err := c.Expand(vm)
	if err != nil {
		return vm, fmt.Errorf("error encountered while expanding into VirtualMachine: %v", err)
	}

	// Only update the VM if we have changed something by applying an instance type and preference
	if !equality.Semantic.DeepEqual(vm, expandVMCopy) {
		updatedVM, err := c.clientset.VirtualMachine(expandVMCopy.Namespace).Update(
			context.Background(), expandVMCopy, metav1.UpdateOptions{})
		if err != nil {
			return vm, fmt.Errorf("error encountered when trying to update expanded VirtualMachine: %v", err)
		}
		updatedVM.Status = expandVMCopy.Status
		updatedVM, err = c.clientset.VirtualMachine(updatedVM.Namespace).UpdateStatus(
			context.Background(), updatedVM, metav1.UpdateOptions{})
		if err != nil {
			return vm, fmt.Errorf("error encountered when trying to update expanded VirtualMachine Status: %v", err)
		}
		// We should clean up any instance type or preference controllerRevisions after successfully expanding the VM
		if revision.HasControllerRevisionRef(vm.Status.InstancetypeRef) {
			if err = c.clientset.AppsV1().ControllerRevisions(vm.Namespace).Delete(
				context.Background(), vm.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.DeleteOptions{}); err != nil {
				return nil, common.NewSyncError(
					fmt.Errorf(cleanControllerRevisionErrFmt, vm.Status.InstancetypeRef.ControllerRevisionRef.Name, vm.Name, err),
					common.FailedCreateVirtualMachineReason,
				)
			}
		}

		if revision.HasControllerRevisionRef(vm.Status.PreferenceRef) {
			if err = c.clientset.AppsV1().ControllerRevisions(vm.Namespace).Delete(
				context.Background(), vm.Status.PreferenceRef.ControllerRevisionRef.Name, metav1.DeleteOptions{}); err != nil {
				return nil, common.NewSyncError(
					fmt.Errorf(cleanControllerRevisionErrFmt, vm.Status.PreferenceRef.ControllerRevisionRef.Name, vm.Name, err),
					common.FailedCreateVirtualMachineReason,
				)
			}
		}
		return updatedVM, nil
	}
	return vm, nil
}

func (c *controller) ApplyDevicePreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vm.Spec.Preference == nil {
		return nil
	}
	preferenceSpec, err := c.FindPreference(vm)
	if err != nil {
		return err
	}
	preferenceapply.ApplyDevicePreferences(preferenceSpec, &vmi.Spec)

	return nil
}

func (c *controller) ApplyToVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	instancetypeSpec, err := c.Find(vm)
	if err != nil {
		return err
	}

	preferenceSpec, err := c.FindPreference(vm)
	if err != nil {
		return err
	}

	if instancetypeSpec == nil && preferenceSpec == nil {
		return nil
	}

	annotations.Set(vm, vmi)
	preferenceannotations.Set(vm, vmi)

	if conflicts := apply.NewVMIApplier().ApplyToVMI(
		k8sfield.NewPath("spec"),
		instancetypeSpec,
		preferenceSpec,
		&vmi.Spec,
		&vmi.ObjectMeta,
	); len(conflicts) > 0 {
		return fmt.Errorf("VMI conflicts with instancetype spec in fields: [%s]", conflicts.String())
	}

	return nil
}
