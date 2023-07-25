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
 * Copyright 2023 Red Hat, Inc.
 *
 */
package preference_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/preference"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	create     = "create"
	namespaced = "--namespaced"
)

var _ = Describe("create", func() {
	Context("preference without arguments", func() {
		DescribeTable("should succeed", func(namespacedFlag string, namespaced bool) {
			err := clientcmd.NewRepeatableVirtctlCommand(create, Preference, namespacedFlag)()
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("VirtualMachinePreference", namespaced, true),
			Entry("VirtualMachineClusterPreference", "", false),
		)
	})

	Context("preference with arguments", func() {
		var preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec

		DescribeTable("should succeed with defined preferred storage class", func(namespacedFlag, PreferredstorageClass string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Preference, namespacedFlag,
				setFlag(VolumeStorageClassFlag, PreferredstorageClass),
			)()
			Expect(err).ToNot(HaveOccurred())

			preferenceSpec, err = getPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec.Volumes.PreferredStorageClassName).To(Equal(PreferredstorageClass))
		},
			Entry("VirtualMachinePreference", namespaced, "testing-storage", true),
			Entry("VirtualMachineClusterPreference", "", "hostpath-provisioner", false),
		)

		DescribeTable("should succeed with defined machine type", func(namespacedFlag, machineType string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Preference, namespacedFlag,
				setFlag(MachineTypeFlag, machineType),
			)()
			Expect(err).ToNot(HaveOccurred())

			preferenceSpec, err = getPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec.Machine.PreferredMachineType).To(Equal(machineType))
		},
			Entry("VirtualMachinePreference", namespaced, "pc-i440fx-2.10", true),
			Entry("VirtualMachineClusterPreference", "", "pc-q35-2.10", false),
		)

		DescribeTable("should succeed with defined CPU topology", func(namespacedFlag, CPUTopology string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Preference, namespacedFlag,
				setFlag(CPUTopologyFlag, CPUTopology),
			)()
			Expect(err).ToNot(HaveOccurred())

			preferenceSpec, err = getPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec.CPU.PreferredCPUTopology).ToNot(BeNil())
			Expect(*preferenceSpec.CPU.PreferredCPUTopology).To(Equal(instancetypev1beta1.PreferredCPUTopology(CPUTopology)))
		},
			Entry("VirtualMachinePreference", namespaced, "preferCores", true),
			Entry("VirtualMachineClusterPreference", "", "preferThreads", false),
		)

		It("should create namespaced object and apply namespace when namespace is specified", func() {
			const namespace = "my-namespace"
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Preference,
				setFlag("namespace", namespace),
			)()
			Expect(err).ToNot(HaveOccurred())

			decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
			Expect(err).ToNot(HaveOccurred())

			preference, ok := decodedObj.(*instancetypev1beta1.VirtualMachinePreference)
			Expect(ok).To(BeTrue())
			Expect(preference.Namespace).To(Equal(namespace))
		})

		DescribeTable("should fail with invalid CPU topology values", func(namespacedFlag, CPUTopology string, namespaced bool) {
			err := clientcmd.NewRepeatableVirtctlCommand(create, Preference, namespacedFlag,
				setFlag(CPUTopologyFlag, CPUTopology),
			)()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("failed to parse \"--cpu-topology\" flag: CPU topology must have a value of preferCores, preferSockets or preferThreads"))
		},
			Entry("VirtualMachinePreference", namespaced, "invalidCPU", true),
			Entry("VirtualMachineClusterPreference", "", "clusterInvalidCPU", false),
		)

	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func getPreferenceSpec(bytes []byte, namespaced bool) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	switch obj := decodedObj.(type) {
	case *instancetypev1beta1.VirtualMachinePreference:
		ExpectWithOffset(1, namespaced).To(BeTrue(), "expected VirtualMachinePreference to be created")
		ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachinePreference"))
		return &obj.Spec, nil
	case *instancetypev1beta1.VirtualMachineClusterPreference:
		ExpectWithOffset(1, namespaced).To(BeFalse(), "expected VirtualMachineClusterPreference to be created")
		ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachineClusterPreference"))
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("object must be VirtualMachinePreference or VirtualMachineClusterPreference")
	}
}
