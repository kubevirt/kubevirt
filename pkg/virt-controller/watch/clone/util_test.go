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

package clone

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	clone "kubevirt.io/api/clone/v1beta1"
)

var _ = Describe("Clone Utils", func() {

	Context("isOwnedByClone", func() {
		var (
			testObj          *metav1.ObjectMeta
			cloneKind        string
			cloneAPIVersion  string
			correctOwnerRef  metav1.OwnerReference
			incorrectKindRef metav1.OwnerReference
		)

		BeforeEach(func() {
			cloneKind = clone.VirtualMachineCloneKind.Kind
			cloneAPIVersion = clone.VirtualMachineCloneKind.GroupVersion().String()

			testObj = &metav1.ObjectMeta{
				Name:      "test-object",
				Namespace: "test-namespace",
			}

			correctOwnerRef = metav1.OwnerReference{
				Kind:       cloneKind,
				APIVersion: cloneAPIVersion,
				Name:       "my-clone",
				UID:        types.UID("12345"),
			}

			incorrectKindRef = metav1.OwnerReference{
				Kind:       "VirtualMachine",
				APIVersion: cloneAPIVersion,
				Name:       "my-vm",
				UID:        types.UID("67890"),
			}
		})

		It("should return true and the key when object is owned by a valid Clone", func() {
			testObj.OwnerReferences = []metav1.OwnerReference{correctOwnerRef}

			isOwned, key := isOwnedByClone(testObj)

			Expect(isOwned).To(BeTrue())
			Expect(key).To(Equal("test-namespace/my-clone"))
		})

		It("should return false when object has no owners", func() {
			testObj.OwnerReferences = []metav1.OwnerReference{}

			isOwned, key := isOwnedByClone(testObj)

			Expect(isOwned).To(BeFalse())
			Expect(key).To(BeEmpty())
		})

		It("should return false when object is owned by something else (wrong Kind)", func() {
			testObj.OwnerReferences = []metav1.OwnerReference{incorrectKindRef}

			isOwned, key := isOwnedByClone(testObj)

			Expect(isOwned).To(BeFalse())
			Expect(key).To(BeEmpty())
		})

		It("should return false when key matches Kind but has wrong API Version", func() {
			wrongVersionRef := correctOwnerRef
			wrongVersionRef.APIVersion = "api.group/v1alpha0" // Intentionally wrong
			testObj.OwnerReferences = []metav1.OwnerReference{wrongVersionRef}

			isOwned, key := isOwnedByClone(testObj)

			Expect(isOwned).To(BeFalse())
			Expect(key).To(BeEmpty())
		})

		It("should return true if multiple owners exist and one is a valid Clone", func() {
			// Mixed ownership: One VM and one Clone
			testObj.OwnerReferences = []metav1.OwnerReference{incorrectKindRef, correctOwnerRef}

			isOwned, key := isOwnedByClone(testObj)

			Expect(isOwned).To(BeTrue())
			Expect(key).To(Equal("test-namespace/my-clone"))
		})
	})
})
