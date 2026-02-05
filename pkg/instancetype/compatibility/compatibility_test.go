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
package compatibility

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	instancetypev1 "kubevirt.io/api/instancetype/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1beta1 "kubevirt.io/api/snapshot/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("compatibility", func() {
	generateUnknownObjectControllerRevision := func() *appsv1.ControllerRevision {
		unknownObject := snapshotv1beta1.VirtualMachineSnapshot{
			TypeMeta: metav1.TypeMeta{
				APIVersion: snapshotv1beta1.SchemeGroupVersion.String(),
				Kind:       "VirtualMachineSnapshot",
			},
		}
		unknownObjectBytes, err := json.Marshal(unknownObject)
		Expect(err).ToNot(HaveOccurred())
		return &appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name: "crName",
			},
			Data: runtime.RawExtension{
				Raw: unknownObjectBytes,
			},
		}
	}

	Context("instancetype", func() {
		var expectedInstancetypeSpec *instancetypev1.VirtualMachineInstancetypeSpec

		BeforeEach(func() {
			expectedInstancetypeSpec = &instancetypev1.VirtualMachineInstancetypeSpec{
				CPU: instancetypev1.CPUInstancetype{
					Guest: 4,
					// Set the following values to be compatible with objects converted from prior API versions
					Model:                 pointer.P(""),
					DedicatedCPUPlacement: pointer.P(false),
					IsolateEmulatorThread: pointer.P(false),
				},
				Memory: instancetypev1.MemoryInstancetype{
					Guest: resource.MustParse("128Mi"),
				},
			}
		})

		generatev1beta1InstancetypeSpec := func() instancetypev1.VirtualMachineInstancetypeSpec {
			return instancetypev1.VirtualMachineInstancetypeSpec{
				CPU: instancetypev1.CPUInstancetype{
					Guest: expectedInstancetypeSpec.CPU.Guest,
				},
				Memory: instancetypev1.MemoryInstancetype{
					Guest: expectedInstancetypeSpec.Memory.Guest,
				},
			}
		}

		DescribeTable("decode ControllerRevision containing ", func(getRevisionData func() []byte) {
			revision := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name: "crName",
				},
				Data: runtime.RawExtension{
					Raw: getRevisionData(),
				},
			}

			spec, err := GetInstancetypeSpec(revision)
			Expect(err).ToNot(HaveOccurred())
			Expect(*spec).To(Equal(*expectedInstancetypeSpec))
		},
			Entry("v1beta1 VirtualMachineInstancetype", func() []byte {
				// Omit optional pointer fields with default values
				expectedInstancetypeSpec.CPU = instancetypev1.CPUInstancetype{
					Guest: 4,
				}

				instancetype := instancetypev1.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineInstancetype",
					},
					Spec: generatev1beta1InstancetypeSpec(),
				}
				instancetypeBytes, err := json.Marshal(instancetype)
				Expect(err).ToNot(HaveOccurred())

				return instancetypeBytes
			}),
			Entry("v1beta1 VirtualMachineClusterInstancetype", func() []byte {
				// Omit optional pointer fields with default values
				expectedInstancetypeSpec.CPU = instancetypev1.CPUInstancetype{
					Guest: 4,
				}

				instancetype := instancetypev1.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineClusterInstancetype",
					},
					Spec: generatev1beta1InstancetypeSpec(),
				}
				instancetypeBytes, err := json.Marshal(instancetype)
				Expect(err).ToNot(HaveOccurred())

				return instancetypeBytes
			}),
		)
		It("decode ControllerRevision fails due to unknown object", func() {
			_, err := GetInstancetypeSpec(generateUnknownObjectControllerRevision())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unexpected type in ControllerRevision"))
		})
	})

	Context("preference", func() {
		var expectedPreferenceSpec *instancetypev1.VirtualMachinePreferenceSpec

		BeforeEach(func() {
			// After conversion from v1beta1, deprecated values should be converted to new values
			preferredCPUTopology := instancetypev1.Cores
			expectedPreferenceSpec = &instancetypev1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1.CPUPreferences{
					PreferredCPUTopology: &preferredCPUTopology,
				},
			}
		})

		generatev1beta1PreferenceSpec := func() instancetypev1beta1.VirtualMachinePreferenceSpec {
			// Use deprecated value in v1beta1 input to test conversion
			preferredTopology := instancetypev1beta1.DeprecatedPreferCores
			return instancetypev1beta1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: &preferredTopology,
				},
			}
		}

		DescribeTable("decode ControllerRevision containing ", func(getRevisionData func() []byte) {
			revision := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name: "crName",
				},
				Data: runtime.RawExtension{
					Raw: getRevisionData(),
				},
			}

			spec, err := GetPreferenceSpec(revision)
			Expect(err).ToNot(HaveOccurred())
			Expect(*spec).To(Equal(*expectedPreferenceSpec))
		},
			Entry("v1beta1 VirtualMachinePreference", func() []byte {
				preference := instancetypev1beta1.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachinePreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachinePreference",
					},
					Spec: generatev1beta1PreferenceSpec(),
				}
				preferenceBytes, err := json.Marshal(preference)
				Expect(err).ToNot(HaveOccurred())

				return preferenceBytes
			}),
			Entry("v1beta1 VirtualMachineClusterPreference", func() []byte {
				preference := instancetypev1beta1.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterPreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineClusterPreference",
					},
					Spec: generatev1beta1PreferenceSpec(),
				}
				preferenceBytes, err := json.Marshal(preference)
				Expect(err).ToNot(HaveOccurred())

				return preferenceBytes
			}),
		)
		It("decode ControllerRevision fails due to unknown object", func() {
			_, err := GetPreferenceSpec(generateUnknownObjectControllerRevision())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unexpected type in ControllerRevision"))
		})
	})
})
