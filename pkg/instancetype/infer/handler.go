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
 * Copyright The KubeVirt Authors
 *
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
