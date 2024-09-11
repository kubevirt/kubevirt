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
package instancetype_test

import (
	"errors"
	"fmt"
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
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"

	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	"kubevirt.io/kubevirt/pkg/virtctl/create"
	. "kubevirt.io/kubevirt/pkg/virtctl/create/instancetype"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("create instancetype", func() {
	Context("should fail", func() {
		DescribeTable("because of required cpu and memory", func(namespaced bool) {
			args := []string{create.CREATE, Instancetype}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			Expect(cmd()).To(MatchError(`required flag(s) "cpu", "memory" not set`))
		},
			Entry("VirtualMachineInstancetype", true),
			Entry("VirtualMachineClusterInstancetype", false),
		)

		DescribeTable("invalid cpu and memory", func(cpu, memory, errMsg string, namespaced bool) {
			cmd := clientcmd.NewRepeatableVirtctlCommand(create.CREATE, Instancetype,
				setFlag(CPUFlag, cpu),
				setFlag(MemoryFlag, memory),
			)
			Expect(cmd()).To(MatchError(ContainSubstring(errMsg)))
		},
			Entry("VirtualMachineInstancetype invalid cpu string value", "two", "256Mi", `parsing "two": invalid syntax`, true),
			Entry("VirtualMachineInstancetype invalid cpu negative value", "-2", "256Mi", `parsing "-2": invalid syntax`, true),
			Entry("VirtualMachineInstancetype invalid memory value", "2", "256My", "quantities must match the regular expression", true),
			Entry("VirtualMachineClusterInstancetype invalid cpu string value", "two", "256Mi", `parsing "two": invalid syntax`, false),
			Entry("VirtualMachineClusterInstancetype invalid cpu negative value", "-2", "256Mi", `parsing "-2": invalid syntax`, false),
			Entry("VirtualMachineClusterInstancetype invalid memory value", "2", "256My", "quantities must match the regular expression", false),
		)

		DescribeTable("with invalid arguments", func(flag, param, errMsg string, namespaced bool) {
			args := []string{create.CREATE, Instancetype,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(flag, param),
			}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			Expect(cmd()).To(MatchError(ContainSubstring(errMsg)))
		},
			Entry("VirtualMachineInstancetype gpu missing name", GPUFlag, "devicename:nvidia", NameErr, true),
			Entry("VirtualMachineInstancetype gpu missing deviceName", GPUFlag, "name:gpu1", DeviceNameErr, true),
			Entry("VirtualMachineInstancetype hostdevice missing name", HostDeviceFlag, "devicename:intel", NameErr, true),
			Entry("VirtualMachineInstancetype hostdevice missing deviceName", HostDeviceFlag, "name:device1", DeviceNameErr, true),
			Entry("VirtualMachineInstancetype invalid IOThreadsPolicy", IOThreadsPolicyFlag, "invalid-policy", IOThreadErr, true),
			Entry("VirtualMachineClusterInstancetype gpu missing name", GPUFlag, "devicename:nvidia", NameErr, false),
			Entry("VirtualMachineClusterInstancetype gpu missing deviceName", GPUFlag, "name:gpu1", DeviceNameErr, false),
			Entry("VirtualMachineClusterInstancetype hostdevice missing name", HostDeviceFlag, "devicename:intel", NameErr, false),
			Entry("VirtualMachineClusterInstancetype hostdevice missing deviceName", HostDeviceFlag, "name:device1", DeviceNameErr, false),
			Entry("VirtualMachineClusterInstancetype invalid IOThreadsPolicy", IOThreadsPolicyFlag, "invalid-policy", IOThreadErr, false),
		)
	})

	Context("should succeed", func() {
		DescribeTable("with defined cpu and memory", func(namespaced bool) {
			args := []string{create.CREATE, Instancetype,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
			}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(args...)()
			Expect(err).ToNot(HaveOccurred())

			spec, err := getInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.CPU.Guest).To(Equal(uint32(2)))
			Expect(spec.Memory.Guest).To(Equal(resource.MustParse("256Mi")))
			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachineInstancetype", true),
			Entry("VirtualMachineClusterInstancetype", false),
		)

		DescribeTable("with defined gpus", func(namespaced bool) {
			args := []string{create.CREATE, Instancetype,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(GPUFlag, "name:gpu1,devicename:nvidia"),
			}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(args...)()
			Expect(err).ToNot(HaveOccurred())

			spec, err := getInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.GPUs).To(HaveLen(1))
			Expect(spec.GPUs[0].Name).To(Equal("gpu1"))
			Expect(spec.GPUs[0].DeviceName).To(Equal("nvidia"))
			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachineInstancetype", true),
			Entry("VirtualMachineClusterInstancetype", false),
		)

		DescribeTable("with defined hostDevices", func(namespaced bool) {
			args := []string{create.CREATE, Instancetype,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(HostDeviceFlag, "name:device1,devicename:intel"),
			}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(args...)()
			Expect(err).ToNot(HaveOccurred())

			spec, err := getInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.HostDevices).To(HaveLen(1))
			Expect(spec.HostDevices[0].Name).To(Equal("device1"))
			Expect(spec.HostDevices[0].DeviceName).To(Equal("intel"))
			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachineInstancetype", true),
			Entry("VirtualMachineClusterInstancetype", false),
		)

		DescribeTable("with valid IOThreadsPolicy", func(policy v1.IOThreadsPolicy, namespaced bool) {
			args := []string{create.CREATE, Instancetype,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(IOThreadsPolicyFlag, string(policy)),
			}
			if namespaced {
				args = append(args, setFlag(NamespacedFlag, "true"))
			}
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(args...)()
			Expect(err).ToNot(HaveOccurred())

			spec, err := getInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(*spec.IOThreadsPolicy).To(Equal(policy))
			Expect(validateInstancetypeSpec(spec)).To(BeEmpty())
		},
			Entry("VirtualMachineInstacetype set to auto", v1.IOThreadsPolicyAuto, true),
			Entry("VirtualMachineInstacetype set to shared", v1.IOThreadsPolicyShared, true),
			Entry("VirtualMachineClusterInstacetype set to auto", v1.IOThreadsPolicyAuto, false),
			Entry("VirtualMachineClusterInstacetype set to shared", v1.IOThreadsPolicyShared, false),
		)

	})

	It("should create namespaced object and apply namespace when namespace is specified", func() {
		const namespace = "my-namespace"
		bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create.CREATE, Instancetype,
			setFlag(CPUFlag, "1"),
			setFlag(MemoryFlag, "128Mi"),
			setFlag("namespace", namespace),
		)()
		Expect(err).ToNot(HaveOccurred())

		decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
		Expect(err).ToNot(HaveOccurred())

		instancetype, ok := decodedObj.(*instancetypev1beta1.VirtualMachineInstancetype)
		Expect(ok).To(BeTrue())
		Expect(instancetype.Namespace).To(Equal(namespace))
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func getInstancetypeSpec(bytes []byte, namespaced bool) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
	Expect(err).ToNot(HaveOccurred())

	switch obj := decodedObj.(type) {
	case *instancetypev1beta1.VirtualMachineInstancetype:
		Expect(namespaced).To(BeTrue())
		Expect(strings.ToLower(obj.Kind)).To(Equal(instancetype.SingularResourceName))
		return &obj.Spec, nil
	case *instancetypev1beta1.VirtualMachineClusterInstancetype:
		Expect(namespaced).To(BeFalse())
		Expect(strings.ToLower(obj.Kind)).To(Equal(instancetype.ClusterSingularResourceName))
		return &obj.Spec, nil
	}

	return nil, errors.New("object must be VirtualMachineInstance or VirtualMachineClusterInstancetype")
}

func validateInstancetypeSpec(spec *instancetypev1beta1.VirtualMachineInstancetypeSpec) []k8sv1.StatusCause {
	return admitters.ValidateInstanceTypeSpec(field.NewPath("spec"), spec)
}
