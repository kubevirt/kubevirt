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

package mutators_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clonebase "kubevirt.io/api/clone"
	clone "kubevirt.io/api/clone/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
)

var _ = Describe("Clone mutating webhook", func() {
	const (
		testSourceVirtualMachineName         = "test-source-vm"
		testSourceVirtualMachineSnapshotName = "test-snapshot"
	)

	DescribeTable("should mutate the spec", func(vmClone *clone.VirtualMachineClone) {
		admissionReview, err := newAdmissionReviewForVMCloneCreation(vmClone)
		Expect(err).ToNot(HaveOccurred())

		const expectedTargetSuffix = "12345"
		mutator := mutators.NewCloneCreateMutatorWithTargetSuffix(expectedTargetSuffix)

		expectedVirtualMachineCloneSpec := vmClone.Spec.DeepCopy()
		expectedVirtualMachineCloneSpec.Target = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(virtualMachineAPIGroup),
			Kind:     virtualMachineKind,
			Name:     fmt.Sprintf("clone-%s-%s", expectedVirtualMachineCloneSpec.Source.Name, expectedTargetSuffix),
		}

		expectedJSONPatch, err := patch.New(patch.WithReplace("/spec", expectedVirtualMachineCloneSpec)).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())

		Expect(mutator.Mutate(admissionReview)).To(Equal(
			&admissionv1.AdmissionResponse{
				Allowed:   true,
				PatchType: pointer.P(admissionv1.PatchTypeJSONPatch),
				Patch:     expectedJSONPatch,
			},
		))
	},
		Entry("When the source is a VirtualMachine and the target is nil",
			newVirtualMachineClone(
				withVirtualMachineSource(testSourceVirtualMachineName),
			),
		),
		Entry("When the source is a VirtualMachine and the target name is empty",
			newVirtualMachineClone(
				withVirtualMachineSource(testSourceVirtualMachineName),
				withVirtualMachineTarget(""),
			),
		),
		Entry("when source is a VirtualMachineSnapshot and target is nil",
			newVirtualMachineClone(
				withVirtualMachineSnapshotSource(testSourceVirtualMachineSnapshotName),
			),
		),
		Entry("when source is a VirtualMachineSnapshot and target name is missing",
			newVirtualMachineClone(
				withVirtualMachineSnapshotSource(testSourceVirtualMachineSnapshotName),
				withVirtualMachineTarget(""),
			),
		),
	)

	It("should not mutate the spec when the target is fully set", func() {
		const testTargetName = "my-vm"

		vmClone := newVirtualMachineClone(
			withVirtualMachineSource(testSourceVirtualMachineName),
			withVirtualMachineTarget(testTargetName),
		)

		admissionReview, err := newAdmissionReviewForVMCloneCreation(vmClone)
		Expect(err).ToNot(HaveOccurred())

		mutator := mutators.NewCloneCreateMutator()

		Expect(mutator.Mutate(admissionReview)).To(Equal(
			&admissionv1.AdmissionResponse{
				Allowed: true,
			},
		))
	})
})

type option func(vmClone *clone.VirtualMachineClone)

func newVirtualMachineClone(options ...option) *clone.VirtualMachineClone {
	newVMClone := &clone.VirtualMachineClone{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubevirt-test-default",
			Name:      "testclone",
		},
	}

	for _, optionFunc := range options {
		optionFunc(newVMClone)
	}

	return newVMClone
}

const (
	virtualMachineAPIGroup = "kubevirt.io"
	virtualMachineKind     = "VirtualMachine"

	virtualMachineSnapshotAPIGroup = "snapshot.kubevirt.io"
	virtualMachineSnapshotKind     = "VirtualMachineSnapshot"
)

func withVirtualMachineSource(virtualMachineName string) option {
	return func(vmClone *clone.VirtualMachineClone) {
		vmClone.Spec.Source = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(virtualMachineAPIGroup),
			Kind:     virtualMachineKind,
			Name:     virtualMachineName,
		}
	}
}

func withVirtualMachineSnapshotSource(virtualMachineSnapshotName string) option {
	return func(vmClone *clone.VirtualMachineClone) {
		vmClone.Spec.Source = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(virtualMachineSnapshotAPIGroup),
			Kind:     virtualMachineSnapshotKind,
			Name:     virtualMachineSnapshotName,
		}
	}
}

func withVirtualMachineTarget(virtualMachineName string) option {
	return func(vmClone *clone.VirtualMachineClone) {
		vmClone.Spec.Target = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(virtualMachineAPIGroup),
			Kind:     virtualMachineKind,
			Name:     virtualMachineName,
		}
	}
}

func newAdmissionReviewForVMCloneCreation(vmClone *clone.VirtualMachineClone) (*admissionv1.AdmissionReview, error) {
	cloneBytes, err := json.Marshal(vmClone)
	if err != nil {
		return nil, err
	}

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    clonebase.GroupName,
				Resource: clonebase.ResourceVMClonePlural,
			},
			Object: runtime.RawExtension{
				Raw: cloneBytes,
			},
		},
	}

	return ar, nil
}
