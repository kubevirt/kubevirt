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

	v1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	v1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
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
		var expectedInstancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec

		BeforeEach(func() {
			expectedInstancetypeSpec = &v1beta1.VirtualMachineInstancetypeSpec{
				CPU: v1beta1.CPUInstancetype{
					Guest: 4,
					// Set the following values to be compatible with objects converted from prior API versions
					Model:                 pointer.P(""),
					DedicatedCPUPlacement: pointer.P(false),
					IsolateEmulatorThread: pointer.P(false),
				},
				Memory: v1beta1.MemoryInstancetype{
					Guest: resource.MustParse("128Mi"),
				},
			}
		})

		generatev1alpah1InstancetypeSpec := func() v1alpha1.VirtualMachineInstancetypeSpec {
			return v1alpha1.VirtualMachineInstancetypeSpec{
				CPU: v1alpha1.CPUInstancetype{
					Guest: expectedInstancetypeSpec.CPU.Guest,
				},
				Memory: v1alpha1.MemoryInstancetype{
					Guest: expectedInstancetypeSpec.Memory.Guest,
				},
			}
		}

		generatev1alpah2InstancetypeSpec := func() v1alpha2.VirtualMachineInstancetypeSpec {
			return v1alpha2.VirtualMachineInstancetypeSpec{
				CPU: v1alpha2.CPUInstancetype{
					Guest: expectedInstancetypeSpec.CPU.Guest,
				},
				Memory: v1alpha2.MemoryInstancetype{
					Guest: expectedInstancetypeSpec.Memory.Guest,
				},
			}
		}

		generatev1beta1InstancetypeSpec := func() v1beta1.VirtualMachineInstancetypeSpec {
			return v1beta1.VirtualMachineInstancetypeSpec{
				CPU: v1beta1.CPUInstancetype{
					Guest: expectedInstancetypeSpec.CPU.Guest,
				},
				Memory: v1beta1.MemoryInstancetype{
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
			Entry("v1alpha1 VirtualMachineInstancetypeSpecRevision with APIVersion", func() []byte {
				instancetypeSpec := generatev1alpah1InstancetypeSpec()

				specBytes, err := json.Marshal(&instancetypeSpec)
				Expect(err).ToNot(HaveOccurred())

				specRevision := v1alpha1.VirtualMachineInstancetypeSpecRevision{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Spec:       specBytes,
				}
				specRevisionBytes, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				return specRevisionBytes
			}),
			Entry("v1alpha1 VirtualMachineInstancetypeSpecRevision without APIVersion", func() []byte {
				instancetypeSpec := generatev1alpah1InstancetypeSpec()

				specBytes, err := json.Marshal(&instancetypeSpec)
				Expect(err).ToNot(HaveOccurred())

				specRevision := v1alpha1.VirtualMachineInstancetypeSpecRevision{
					APIVersion: "",
					Spec:       specBytes,
				}
				specRevisionBytes, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				return specRevisionBytes
			}),
			Entry("v1alpha1 VirtualMachineInstancetype", func() []byte {
				instancetype := v1alpha1.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineInstancetype",
					},
					Spec: generatev1alpah1InstancetypeSpec(),
				}
				instancetypeBytes, err := json.Marshal(instancetype)
				Expect(err).ToNot(HaveOccurred())

				return instancetypeBytes
			}),
			Entry("v1alpha1 VirtualMachineClusterInstancetype", func() []byte {
				instancetype := v1alpha1.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineClusterInstancetype",
					},
					Spec: generatev1alpah1InstancetypeSpec(),
				}
				instancetypeBytes, err := json.Marshal(instancetype)
				Expect(err).ToNot(HaveOccurred())

				return instancetypeBytes
			}),
			Entry("v1alpha2 VirtualMachineInstancetype", func() []byte {
				instancetype := v1alpha2.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha2.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineInstancetype",
					},
					Spec: generatev1alpah2InstancetypeSpec(),
				}
				instancetypeBytes, err := json.Marshal(instancetype)
				Expect(err).ToNot(HaveOccurred())

				return instancetypeBytes
			}),
			Entry("v1alpha2 VirtualMachineClusterInstancetype", func() []byte {
				instancetype := v1alpha2.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha2.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterInstancetype",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineClusterInstancetype",
					},
					Spec: generatev1alpah2InstancetypeSpec(),
				}
				instancetypeBytes, err := json.Marshal(instancetype)
				Expect(err).ToNot(HaveOccurred())

				return instancetypeBytes
			}),
			Entry("v1beta1 VirtualMachineInstancetype", func() []byte {
				// Omit optional pointer fields with default values
				expectedInstancetypeSpec.CPU = v1beta1.CPUInstancetype{
					Guest: 4,
				}

				instancetype := v1beta1.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1beta1.SchemeGroupVersion.String(),
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
				expectedInstancetypeSpec.CPU = v1beta1.CPUInstancetype{
					Guest: 4,
				}

				instancetype := v1beta1.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1beta1.SchemeGroupVersion.String(),
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
		var expectedPreferenceSpec *v1beta1.VirtualMachinePreferenceSpec

		BeforeEach(func() {
			preferredCPUTopology := v1beta1.DeprecatedPreferCores
			expectedPreferenceSpec = &v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
					PreferredCPUTopology: &preferredCPUTopology,
				},
			}
		})

		generatev1alpah1PreferenceSpec := func() v1alpha1.VirtualMachinePreferenceSpec {
			return v1alpha1.VirtualMachinePreferenceSpec{
				CPU: &v1alpha1.CPUPreferences{
					PreferredCPUTopology: v1alpha1.PreferCores,
				},
			}
		}

		generatev1alpah2PreferenceSpec := func() v1alpha2.VirtualMachinePreferenceSpec {
			return v1alpha2.VirtualMachinePreferenceSpec{
				CPU: &v1alpha2.CPUPreferences{
					PreferredCPUTopology: v1alpha2.PreferCores,
				},
			}
		}

		generatev1beta1PreferenceSpec := func() v1beta1.VirtualMachinePreferenceSpec {
			preferredTopology := v1beta1.DeprecatedPreferCores
			return v1beta1.VirtualMachinePreferenceSpec{
				CPU: &v1beta1.CPUPreferences{
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
			Entry("v1alpha1 VirtualMachinePreferenceSpecRevision with APIVersion", func() []byte {
				preferenceSpec := generatev1alpah1PreferenceSpec()
				specBytes, err := json.Marshal(&preferenceSpec)
				Expect(err).ToNot(HaveOccurred())

				specRevision := v1alpha1.VirtualMachinePreferenceSpecRevision{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Spec:       specBytes,
				}
				specRevisionBytes, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				return specRevisionBytes
			}),
			Entry("v1alpha1 VirtualMachineInstancetypeSpecRevision without APIVersion", func() []byte {
				preferenceSpec := generatev1alpah1PreferenceSpec()
				specBytes, err := json.Marshal(&preferenceSpec)
				Expect(err).ToNot(HaveOccurred())

				specRevision := v1alpha1.VirtualMachinePreferenceSpecRevision{
					APIVersion: "",
					Spec:       specBytes,
				}
				specRevisionBytes, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				return specRevisionBytes
			}),
			Entry("v1alpha1 VirtualMachinePreference", func() []byte {
				preference := v1alpha1.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachinePreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachinePreference",
					},
					Spec: generatev1alpah1PreferenceSpec(),
				}
				preferenceBytes, err := json.Marshal(preference)
				Expect(err).ToNot(HaveOccurred())

				return preferenceBytes
			}),
			Entry("v1alpha1 VirtualMachineClusterPreference", func() []byte {
				preference := v1alpha1.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterPreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineClusterPreference",
					},
					Spec: generatev1alpah1PreferenceSpec(),
				}
				preferenceBytes, err := json.Marshal(preference)
				Expect(err).ToNot(HaveOccurred())

				return preferenceBytes
			}),
			Entry("v1alpha2 VirtualMachinePreference", func() []byte {
				preference := v1alpha2.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha2.SchemeGroupVersion.String(),
						Kind:       "VirtualMachinePreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachinePreference",
					},
					Spec: generatev1alpah2PreferenceSpec(),
				}
				preferenceBytes, err := json.Marshal(preference)
				Expect(err).ToNot(HaveOccurred())

				return preferenceBytes
			}),
			Entry("v1alpha2 VirtualMachineClusterInstancetype", func() []byte {
				preference := v1alpha2.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha2.SchemeGroupVersion.String(),
						Kind:       "VirtualMachineClusterPreference",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "VirtualMachineClusterPreference",
					},
					Spec: generatev1alpah2PreferenceSpec(),
				}
				preferenceBytes, err := json.Marshal(preference)
				Expect(err).ToNot(HaveOccurred())

				return preferenceBytes
			}),
			Entry("v1beta1 VirtualMachinePreference", func() []byte {
				preference := v1beta1.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1beta1.SchemeGroupVersion.String(),
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
			Entry("v1beta1 VirtualMachineClusterPreference", func() []byte {
				preference := v1beta1.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1beta1.SchemeGroupVersion.String(),
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
