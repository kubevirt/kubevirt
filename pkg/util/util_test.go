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

package util

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("HasDeclarativeVMState", func() {
	DescribeTable("should report whether the VMI opts into the declarative virtualMachineState API",
		func(state *v1.VirtualMachineStateSpec, expected bool) {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					VirtualMachineState: state,
				},
			}
			Expect(HasDeclarativeVMState(vmi)).To(Equal(expected))
		},
		Entry("VirtualMachineState has a volumeClaimTemplate", &v1.VirtualMachineStateSpec{VolumeClaimTemplate: &k8sv1.PersistentVolumeClaimTemplate{}}, true),
		Entry("VirtualMachineState has a source", &v1.VirtualMachineStateSpec{Source: &v1.VirtualMachineStateSource{Name: "pvc"}}, true),
		Entry("VirtualMachineState is empty", &v1.VirtualMachineStateSpec{}, false),
		Entry("VirtualMachineState is nil", nil, false),
	)
})

var _ = Describe("VMState canonical paths", func() {
	It("VMStateCanonicalEFIVarsPath returns the canonical EFI vars file", func() {
		Expect(VMStateCanonicalEFIVarsPath()).To(Equal(filepath.Join(VMStatePVCMountPath, VMStateDirEFI, VMStateEFIVarsFile)))
	})
})
