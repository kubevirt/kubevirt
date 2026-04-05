/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package infer

import (
	"errors"

	virtv1 "kubevirt.io/api/core/v1"
	api "kubevirt.io/api/instancetype"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

const logVerbosityLevel = 3

type handler struct {
	virtClient kubecli.KubevirtClient
}

func New(virtClient kubecli.KubevirtClient) *handler {
	return &handler{
		virtClient: virtClient,
	}
}

func shouldIgnoreFailure(ignoreFailurePolicy *virtv1.InferFromVolumeFailurePolicy) bool {
	return ignoreFailurePolicy != nil && *ignoreFailurePolicy == virtv1.IgnoreInferFromVolumeFailure
}

func (h *handler) Infer(vm *virtv1.VirtualMachine) error {
	if err := h.Instancetype(vm); err != nil {
		return err
	}
	if err := h.Preference(vm); err != nil {
		return err
	}
	return nil
}

func (h *handler) Instancetype(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Instancetype == nil {
		return nil
	}
	// Leave matcher unchanged when inference is disabled
	if vm.Spec.Instancetype.InferFromVolume == "" {
		return nil
	}

	ignoreFailure := shouldIgnoreFailure(vm.Spec.Instancetype.InferFromVolumeFailurePolicy)
	defaultName, defaultKind, err := h.fromVolumes(
		vm, vm.Spec.Instancetype.InferFromVolume, api.DefaultInstancetypeLabel, api.DefaultInstancetypeKindLabel)
	if err != nil {
		var ignoreableInferenceErr *IgnoreableInferenceError
		if errors.As(err, &ignoreableInferenceErr) && ignoreFailure {
			log.Log.Object(vm).V(logVerbosityLevel).Info("Ignored error during inference of instancetype, clearing matcher.")
			vm.Spec.Instancetype = nil
			return nil
		}
		return err
	}

	if ignoreFailure {
		vm.Spec.Template.Spec.Domain.Memory = nil
	}

	vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
		Name: defaultName,
		Kind: defaultKind,
	}
	return nil
}

func (h *handler) Preference(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Preference == nil {
		return nil
	}
	// Leave matcher unchanged when inference is disabled
	if vm.Spec.Preference.InferFromVolume == "" {
		return nil
	}

	ignoreFailure := shouldIgnoreFailure(vm.Spec.Preference.InferFromVolumeFailurePolicy)
	defaultName, defaultKind, err := h.fromVolumes(
		vm, vm.Spec.Preference.InferFromVolume, api.DefaultPreferenceLabel, api.DefaultPreferenceKindLabel)
	if err != nil {
		var ignoreableInferenceErr *IgnoreableInferenceError
		if errors.As(err, &ignoreableInferenceErr) && ignoreFailure {
			log.Log.Object(vm).V(logVerbosityLevel).Info("Ignored error during inference of preference, clearing matcher.")
			vm.Spec.Preference = nil
			return nil
		}
		return err
	}

	vm.Spec.Preference = &virtv1.PreferenceMatcher{
		Name: defaultName,
		Kind: defaultKind,
	}
	return nil
}
