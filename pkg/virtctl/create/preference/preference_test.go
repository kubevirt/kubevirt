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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"

	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	"kubevirt.io/kubevirt/pkg/virtctl/create"
	. "kubevirt.io/kubevirt/pkg/virtctl/create/preference"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("create preference", func() {
	DescribeTable("should fail with invalid CPU topology values", func(topology string, namespaced bool) {
		args := []string{create.CREATE, Preference,
			setFlag(CPUTopologyFlag, topology),
		}
		if namespaced {
			args = append(args, setFlag(NamespacedFlag, "true"))
		}
		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(MatchError(ContainSubstring(CPUTopologyErr)))
	},
		Entry("VirtualMachinePreference", "invalidCPU", true),
		Entry("VirtualMachineClusterPreference", "clusterInvalidCPU", false),
	)

	Context("should succeed", func() {
		DescribeTable("without arguments", func(namespaced bool) {
			args := []string{create.CREATE, Preference}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			Expect(cmd()).To(Succeed())
		},
			Entry("VirtualMachinePreference", true),
			Entry("VirtualMachineClusterPreference", false),
		)

		DescribeTable("with defined preferred storage class", func(storageClass string, namespaced bool) {
			args := []string{create.CREATE, Preference,
				setFlag(VolumeStorageClassFlag, storageClass),
			}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(args...)()
			Expect(err).ToNot(HaveOccurred())

			spec, err := getPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.Volumes.PreferredStorageClassName).To(Equal(storageClass))
			Expect(validatePreferenceSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachinePreference", "testing-storage", true),
			Entry("VirtualMachineClusterPreference", "hostpath-provisioner", false),
		)

		DescribeTable("with defined machine type", func(machineType string, namespaced bool) {
			args := []string{create.CREATE, Preference,
				setFlag(MachineTypeFlag, machineType),
			}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(args...)()
			Expect(err).ToNot(HaveOccurred())

			spec, err := getPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.Machine.PreferredMachineType).To(Equal(machineType))
			Expect(validatePreferenceSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachinePreference", "pc-i440fx-2.10", true),
			Entry("VirtualMachineClusterPreference", "pc-q35-2.10", false),
		)

		DescribeTable("with defined CPU topology", func(topology instancetypev1beta1.PreferredCPUTopology, namespaced bool) {
			args := []string{create.CREATE, Preference,
				setFlag(CPUTopologyFlag, string(topology)),
			}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(args...)()
			Expect(err).ToNot(HaveOccurred())

			spec, err := getPreferenceSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.CPU.PreferredCPUTopology).ToNot(BeNil())
			Expect(*spec.CPU.PreferredCPUTopology).To(Equal(topology))
			Expect(validatePreferenceSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachinePreference", instancetypev1beta1.DeprecatedPreferCores, true),
			Entry("VirtualMachineClusterPreference", instancetypev1beta1.DeprecatedPreferThreads, false),
		)
	})

	It("should create namespaced object and apply namespace when namespace is specified", func() {
		const namespace = "my-namespace"
		bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create.CREATE, Preference,
			setFlag("namespace", namespace),
		)()
		Expect(err).ToNot(HaveOccurred())

		decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
		Expect(err).ToNot(HaveOccurred())

		preference, ok := decodedObj.(*instancetypev1beta1.VirtualMachinePreference)
		Expect(ok).To(BeTrue())
		Expect(preference.Namespace).To(Equal(namespace))
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func getPreferenceSpec(bytes []byte, namespaced bool) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
	Expect(err).ToNot(HaveOccurred())

	switch obj := decodedObj.(type) {
	case *instancetypev1beta1.VirtualMachinePreference:
		Expect(namespaced).To(BeTrue())
		Expect(strings.ToLower(obj.Kind)).To(Equal(instancetype.SingularPreferenceResourceName))
		return &obj.Spec, nil
	case *instancetypev1beta1.VirtualMachineClusterPreference:
		Expect(namespaced).To(BeFalse())
		Expect(strings.ToLower(obj.Kind)).To(Equal(instancetype.ClusterSingularPreferenceResourceName))
		return &obj.Spec, nil
	}

	return nil, fmt.Errorf("object must be VirtualMachinePreference or VirtualMachineClusterPreference")
}

func validatePreferenceSpec(spec *instancetypev1beta1.VirtualMachinePreferenceSpec) []k8sv1.StatusCause {
	return admitters.ValidatePreferenceSpec(field.NewPath("spec"), spec)
}
