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
package instancetype_test

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"
	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"

	"kubevirt.io/kubevirt/pkg/instancetype/webhooks"
	"kubevirt.io/kubevirt/pkg/virtctl/create"
	. "kubevirt.io/kubevirt/pkg/virtctl/create/instancetype"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("create instancetype", func() {
	Context("should fail", func() {
		const (
			ioThreadErr   = "IOThread must be of value auto or shared"
			nameErr       = "name must be specified"
			deviceNameErr = "deviceName must be specified"
		)

		DescribeTable("because of required cpu and memory", func(extraArgs ...string) {
			_, err := runCmd(extraArgs...)
			Expect(err).To(MatchError(`required flag(s) "cpu", "memory" not set`))
		},
			Entry("VirtualMachineInstancetype", setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterInstancetype"),
		)

		DescribeTable("invalid cpu and memory", func(cpu, memory, errMsg string, extraArgs ...string) {
			args := append([]string{
				setFlag(CPUFlag, cpu),
				setFlag(MemoryFlag, memory),
			}, extraArgs...)
			_, err := runCmd(args...)
			Expect(err).To(MatchError(ContainSubstring(errMsg)))
		},
			Entry("VirtualMachineInstancetype invalid cpu string value",
				"two", "256Mi", `parsing "two": invalid syntax`, setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineInstancetype invalid cpu negative value",
				"-2", "256Mi", `parsing "-2": invalid syntax`, setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineInstancetype invalid memory value",
				"2", "256My", "quantities must match the regular expression", setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterInstancetype invalid cpu string value", "two", "256Mi", `parsing "two": invalid syntax`),
			Entry("VirtualMachineClusterInstancetype invalid cpu negative value", "-2", "256Mi", `parsing "-2": invalid syntax`),
			Entry("VirtualMachineClusterInstancetype invalid memory value", "2", "256My", "quantities must match the regular expression"),
		)

		DescribeTable("with invalid arguments", func(errMsg string, extraArgs ...string) {
			args := append([]string{
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
			}, extraArgs...)
			_, err := runCmd(args...)
			Expect(err).To(MatchError(ContainSubstring(errMsg)))
		},
			Entry("VirtualMachineInstancetype gpu missing name",
				nameErr,
				setFlag(GPUFlag, "devicename:nvidia"),
				setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineInstancetype gpu missing deviceName",
				deviceNameErr,
				setFlag(GPUFlag, "name:gpu1"),
				setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineInstancetype hostdevice missing name",
				nameErr,
				setFlag(HostDeviceFlag, "devicename:intel"),
				setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineInstancetype hostdevice missing deviceName",
				deviceNameErr,
				setFlag(HostDeviceFlag, "name:device1"),
				setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineInstancetype invalid IOThreadsPolicy",
				ioThreadErr,
				setFlag(IOThreadsPolicyFlag, "invalid-policy"),
				setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterInstancetype gpu missing name",
				nameErr,
				setFlag(GPUFlag, "devicename:nvidia")),
			Entry("VirtualMachineClusterInstancetype gpu missing deviceName",
				deviceNameErr,
				setFlag(GPUFlag, "name:gpu1")),
			Entry("VirtualMachineClusterInstancetype hostdevice missing name",
				nameErr,
				setFlag(HostDeviceFlag, "devicename:intel")),
			Entry("VirtualMachineClusterInstancetype hostdevice missing deviceName",
				deviceNameErr,
				setFlag(HostDeviceFlag, "name:device1")),
			Entry("VirtualMachineClusterInstancetype invalid IOThreadsPolicy",
				ioThreadErr,
				setFlag(IOThreadsPolicyFlag, "invalid-policy")),
		)
	})

	Context("should succeed", func() {
		DescribeTable("with defined cpu and memory", func(extraArgs ...string) {
			args := append([]string{
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			spec := getInstancetypeSpec(out)
			Expect(spec.CPU.Guest).To(Equal(uint32(2)))
			Expect(spec.Memory.Guest).To(Equal(resource.MustParse("256Mi")))
			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachineInstancetype", setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterInstancetype"),
		)

		DescribeTable("with namespaced flag", func(namespaced bool) {
			out, err := runCmd(
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(NamespacedFlag, strconv.FormatBool(namespaced)),
			)
			Expect(err).ToNot(HaveOccurred())

			decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), out)
			Expect(err).ToNot(HaveOccurred())

			var spec *instancetypev1beta1.VirtualMachineInstancetypeSpec
			if namespaced {
				instancetype, ok := decodedObj.(*instancetypev1beta1.VirtualMachineInstancetype)
				Expect(ok).To(BeTrue())
				spec = &instancetype.Spec
			} else {
				clusterInstancetype, ok := decodedObj.(*instancetypev1beta1.VirtualMachineClusterInstancetype)
				Expect(ok).To(BeTrue())
				spec = &clusterInstancetype.Spec
			}

			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachinePreference", true),
			Entry("VirtualMachineClusterPreference", false),
		)

		runDeviceTest := func(flagName, deviceParam string, extraArgs ...string) *instancetypev1beta1.VirtualMachineInstancetypeSpec {
			args := append([]string{
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(flagName, deviceParam),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			return getInstancetypeSpec(out)
		}

		DescribeTable("with defined GPU", func(extraArgs ...string) {
			spec := runDeviceTest(GPUFlag, "name:gpu1,devicename:nvidia", extraArgs...)
			Expect(spec.GPUs).To(HaveLen(1))
			Expect(spec.GPUs[0].Name).To(Equal("gpu1"))
			Expect(spec.GPUs[0].DeviceName).To(Equal("nvidia"))
			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachineInstancetype", setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterInstancetype"),
		)

		DescribeTable("with defined HostDevice", func(extraArgs ...string) {
			spec := runDeviceTest(HostDeviceFlag, "name:device1,devicename:intel", extraArgs...)
			Expect(spec.HostDevices).To(HaveLen(1))
			Expect(spec.HostDevices[0].Name).To(Equal("device1"))
			Expect(spec.HostDevices[0].DeviceName).To(Equal("intel"))
			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachineInstancetype", setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterInstancetype"),
		)

		DescribeTable("with valid IOThreadsPolicy", func(policy v1.IOThreadsPolicy, extraArgs ...string) {
			args := append([]string{
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(IOThreadsPolicyFlag, string(policy)),
			}, extraArgs...)
			out, err := runCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			spec := getInstancetypeSpec(out)
			Expect(*spec.IOThreadsPolicy).To(Equal(policy))
			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachineInstacetype set to auto", v1.IOThreadsPolicyAuto, setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineInstacetype set to shared", v1.IOThreadsPolicyShared, setFlag(NamespacedFlag, "true")),
			Entry("VirtualMachineClusterInstacetype set to auto", v1.IOThreadsPolicyAuto),
			Entry("VirtualMachineClusterInstacetype set to shared", v1.IOThreadsPolicyShared),
		)
	})

	It("should create namespaced object and apply namespace when namespace is specified", func() {
		const namespace = "my-namespace"
		out, err := runCmd(
			setFlag(CPUFlag, "1"),
			setFlag(MemoryFlag, "128Mi"),
			setFlag("namespace", namespace),
		)
		Expect(err).ToNot(HaveOccurred())

		decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), out)
		Expect(err).ToNot(HaveOccurred())

		instancetype, ok := decodedObj.(*instancetypev1beta1.VirtualMachineInstancetype)
		Expect(ok).To(BeTrue())
		Expect(instancetype.Namespace).To(Equal(namespace))
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func runCmd(extraArgs ...string) ([]byte, error) {
	args := append([]string{create.CREATE, "instancetype"}, extraArgs...)
	return testing.NewRepeatableVirtctlCommandWithOut(args...)()
}

func getInstancetypeSpec(bytes []byte) *instancetypev1beta1.VirtualMachineInstancetypeSpec {
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
	Expect(err).ToNot(HaveOccurred())

	switch obj := decodedObj.(type) {
	case *instancetypev1beta1.VirtualMachineInstancetype:
		Expect(strings.ToLower(obj.Kind)).To(Equal(instancetype.SingularResourceName))
		return &obj.Spec
	case *instancetypev1beta1.VirtualMachineClusterInstancetype:
		Expect(strings.ToLower(obj.Kind)).To(Equal(instancetype.ClusterSingularResourceName))
		return &obj.Spec
	default:
		Fail("object must be VirtualMachineInstance or VirtualMachineClusterInstancetype")
		return nil
	}
}

func validateInstancetypeSpec(spec *instancetypev1beta1.VirtualMachineInstancetypeSpec) []k8sv1.StatusCause {
	return webhooks.ValidateInstanceTypeSpec(field.NewPath("spec"), spec)
}
