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
 */

package render_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/render"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Pre-default preferences", func() {
	var (
		vm         *virtv1.VirtualMachine
		preference *v1beta1.VirtualMachinePreferenceSpec
	)

	BeforeEach(func() {
		vm = libvmi.NewVirtualMachine(libvmi.New())
		vm.Spec.DataVolumeTemplates = []virtv1.DataVolumeTemplateSpec{{
			Spec: cdiv1.DataVolumeSpec{PVC: &k8sv1.PersistentVolumeClaimSpec{}},
		}, {
			Spec: cdiv1.DataVolumeSpec{Storage: &cdiv1.StorageSpec{}},
		}, {}}
		preference = &v1beta1.VirtualMachinePreferenceSpec{
			Volumes: &v1beta1.VolumePreferences{PreferredStorageClassName: "preferred"},
		}
	})

	DescribeTable("applies preferred storage classes to PVC and Storage templates", func(withTemplate bool) {
		if !withTemplate {
			vm.Spec.Template = nil
		}
		originalPreference := preference.DeepCopy()

		render.ApplyPreDefaultPreferences(vm, preference)

		pvc := vm.Spec.DataVolumeTemplates[0].Spec.PVC
		storage := vm.Spec.DataVolumeTemplates[1].Spec.Storage
		Expect(pvc.StorageClassName).To(Equal(pointer.P("preferred")))
		Expect(storage.StorageClassName).To(Equal(pointer.P("preferred")))
		*pvc.StorageClassName = "changed"
		Expect(storage.StorageClassName).To(Equal(pointer.P("preferred")))
		Expect(preference).To(Equal(originalPreference))
	},
		Entry("with a VM instance template", true),
		Entry("without a VM instance template", false),
	)

	DescribeTable("preserves explicit storage classes", func(storageClass string) {
		vm.Spec.DataVolumeTemplates[0].Spec.PVC.StorageClassName = pointer.P(storageClass)
		vm.Spec.DataVolumeTemplates[1].Spec.Storage.StorageClassName = pointer.P(storageClass)

		render.ApplyPreDefaultPreferences(vm, preference)

		Expect(vm.Spec.DataVolumeTemplates[0].Spec.PVC.StorageClassName).To(Equal(pointer.P(storageClass)))
		Expect(vm.Spec.DataVolumeTemplates[1].Spec.Storage.StorageClassName).To(Equal(pointer.P(storageClass)))
	},
		Entry("an empty class", ""),
		Entry("a named class", "explicit"),
	)

	DescribeTable("leaves storage classes unset without a preference", func(preference *v1beta1.VirtualMachinePreferenceSpec) {
		originalVM := vm.DeepCopy()

		render.ApplyPreDefaultPreferences(vm, preference)

		Expect(vm).To(Equal(originalVM))
	},
		Entry("no preference spec", nil),
		Entry("no volume preferences", &v1beta1.VirtualMachinePreferenceSpec{}),
		Entry("an empty preferred class", &v1beta1.VirtualMachinePreferenceSpec{Volumes: &v1beta1.VolumePreferences{}}),
	)
})
