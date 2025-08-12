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
package preference_test

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/webhooks"
	"kubevirt.io/kubevirt/pkg/virtctl/create"
	. "kubevirt.io/kubevirt/pkg/virtctl/create/preference"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("create preference", func() {
	DescribeTable("should fail with invalid CPU topology values", func(topology string, extraArgs ...string) {
		args := append([]string{
			setFlag(CPUTopologyFlag, topology),
		}, extraArgs...)
		_, err := runCmd(args...)
		Expect(err).To(MatchError(ContainSubstring("CPU topology must have a value of sockets, cores, threads or spread")))
	},
		Entry("VirtualMachinePreference", "invalidCPU", setFlag(NamespacedFlag, "true")),
		Entry("VirtualMachineClusterPreference", "clusterInvalidCPU"),
	)

	Context("should succeed", func() {
		It("without flags", func() {
			out, err := runCmd()
			Expect(err).ToNot(HaveOccurred())

			decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), out)
			Expect(err).ToNot(HaveOccurred())
			clusterPreference, ok := decodedObj.(*instancetypev1beta1.VirtualMachineClusterPreference)
			Expect(ok).To(BeTrue())
			Expect(validatePreferenceSpec(&clusterPreference.Spec)).To(BeEmpty())
		})

		DescribeTable("with namespaced flag", func(namespaced bool) {
			out, err := runCmd(setFlag(NamespacedFlag, strconv.FormatBool(namespaced)))
			Expect(err).ToNot(HaveOccurred())

			decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), out)
			Expect(err).ToNot(HaveOccurred())

			var spec *instancetypev1beta1.VirtualMachinePreferenceSpec
			if namespaced {
				preference, ok := decodedObj.(*instancetypev1beta1.VirtualMachinePreference)
				Expect(ok).To(BeTrue())
				spec = &preference.Spec
			} else {
				clusterPreference, ok := decodedObj.(*instancetypev1beta1.VirtualMachineClusterPreference)
				Expect(ok).To(BeTrue())
				spec = &clusterPreference.Spec
			}

			Expect(validatePreferenceSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachinePreference", true),
			Entry("VirtualMachineClusterPreference", false),
		)

		DescribeTable("with defined preferred storage class", func(storageClass string, extraArgs ...string) {
			args := append([]string{
				setFlag(VolumeStorageClassFlag, storageClass),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			spec := getPreferenceSpec(out)
			Expect(spec.Volumes.PreferredStorageClassName).To(Equal(storageClass))
			Expect(validatePreferenceSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachinePreference", "testing-storage", setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterPreference", "hostpath-provisioner"),
		)

		DescribeTable("with defined machine type", func(machineType string, extraArgs ...string) {
			args := append([]string{
				setFlag(MachineTypeFlag, machineType),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			spec := getPreferenceSpec(out)
			Expect(spec.Machine.PreferredMachineType).To(HaveValue(Equal(machineType)))
			Expect(validatePreferenceSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachinePreference", "pc-i440fx-2.10", setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterPreference", "pc-q35-2.10"),
		)

		DescribeTable("with defined CPU topology", func(topology instancetypev1beta1.PreferredCPUTopology, extraArgs ...string) {
			args := append([]string{
				setFlag(CPUTopologyFlag, string(topology)),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			spec := getPreferenceSpec(out)
			Expect(spec.CPU.PreferredCPUTopology).ToNot(BeNil())
			Expect(*spec.CPU.PreferredCPUTopology).To(Equal(topology))
			Expect(validatePreferenceSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachinePreference", instancetypev1beta1.DeprecatedPreferCores, setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterPreference", instancetypev1beta1.DeprecatedPreferThreads),
		)
	})

	It("should create namespaced object and apply namespace when namespace is specified", func() {
		const namespace = "my-namespace"
		out, err := runCmd(setFlag("namespace", namespace))
		Expect(err).ToNot(HaveOccurred())

		decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), out)
		Expect(err).ToNot(HaveOccurred())

		preference, ok := decodedObj.(*instancetypev1beta1.VirtualMachinePreference)
		Expect(ok).To(BeTrue())
		Expect(preference.Namespace).To(Equal(namespace))
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func runCmd(extraArgs ...string) ([]byte, error) {
	args := append([]string{create.CREATE, "preference"}, extraArgs...)
	return testing.NewRepeatableVirtctlCommandWithOut(args...)()
}

func getPreferenceSpec(bytes []byte) *instancetypev1beta1.VirtualMachinePreferenceSpec {
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
	Expect(err).ToNot(HaveOccurred())

	switch obj := decodedObj.(type) {
	case *instancetypev1beta1.VirtualMachinePreference:
		Expect(strings.ToLower(obj.Kind)).To(Equal(instancetype.SingularPreferenceResourceName))
		return &obj.Spec
	case *instancetypev1beta1.VirtualMachineClusterPreference:
		Expect(strings.ToLower(obj.Kind)).To(Equal(instancetype.ClusterSingularPreferenceResourceName))
		return &obj.Spec
	default:
		Fail("object must be VirtualMachinePreference or VirtualMachineClusterPreference")
		return nil
	}
}

func validatePreferenceSpec(spec *instancetypev1beta1.VirtualMachinePreferenceSpec) []k8sv1.StatusCause {
	return webhooks.ValidatePreferenceSpec(field.NewPath("spec"), spec)
}
