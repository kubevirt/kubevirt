/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/requirements"
	"kubevirt.io/kubevirt/pkg/instancetype/preference/validation"
)

type instancetypeFinder interface {
	Find(*virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
}

type preferenceFinder interface {
	FindPreference(*virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

type requirementsChecker interface {
	Check(*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		*virtv1.VirtualMachineInstanceSpec,
	) (conflict.Conflicts, error)
}

type applyVMIHandler interface {
	ApplyToVMI(
		*k8sfield.Path,
		*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		*virtv1.VirtualMachineInstanceSpec,
		*metav1.ObjectMeta,
	) conflict.Conflicts
}

type admitter struct {
	instancetypeFinder
	preferenceFinder
	applyVMIHandler
	requirementsChecker
}

func NewAdmitter(virtClient kubecli.KubevirtClient) *admitter {
	return &admitter{
		instancetypeFinder:  find.NewSpecFinder(nil, nil, nil, virtClient),
		preferenceFinder:    preferenceFind.NewSpecFinder(nil, nil, nil, virtClient),
		requirementsChecker: requirements.New(),
		applyVMIHandler:     apply.NewVMIApplier(),
	}
}

func (a *admitter) ApplyToVM(vm *virtv1.VirtualMachine) (
	*v1beta1.VirtualMachineInstancetypeSpec,
	*v1beta1.VirtualMachinePreferenceSpec,
	[]metav1.StatusCause,
) {
	const ignoreFindFailureWarnFmt = "ignoring err %q when looking for %s"

	instancetypeSpec, err := a.Find(vm)
	if err != nil {
		log.Log.Object(vm).Warningf(ignoreFindFailureWarnFmt, err, "instance type")
	}

	preferenceSpec, err := a.FindPreference(vm)
	if err != nil {
		log.Log.Object(vm).Warningf(ignoreFindFailureWarnFmt, err, "preference")
	}

	if instancetypeSpec == nil && preferenceSpec == nil {
		return nil, nil, nil
	}

	if spreadConflict := validation.CheckSpreadCPUTopology(instancetypeSpec, preferenceSpec); spreadConflict != nil {
		return nil, nil, spreadConflict.StatusCauses()
	}

	conflicts := a.ApplyToVMI(
		k8sfield.NewPath("spec", "template", "spec"),
		instancetypeSpec,
		preferenceSpec,
		&vm.Spec.Template.Spec,
		&vm.Spec.Template.ObjectMeta,
	)

	if len(conflicts) > 0 {
		return nil, nil, conflicts.StatusCauses()
	}

	return instancetypeSpec, preferenceSpec, nil
}
