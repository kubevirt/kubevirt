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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/instancetype"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	create     = "create"
	namespaced = "--namespaced"
)

var _ = Describe("create", func() {
	Context("instancetype without arguments", func() {
		DescribeTable("should fail because of required cpu and memory", func(namespacedFlag string) {
			err := clientcmd.NewRepeatableVirtctlCommand(create, Instancetype, namespacedFlag)()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("required flag(s) \"cpu\", \"memory\" not set"))
		},
			Entry("VirtualMachineInstancetype", namespaced),
			Entry("VirtualMachineClusterInstancetype", ""),
		)
	})

	Context("instancetype with arguments", func() {
		var instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec

		DescribeTable("should succeed with defined cpu and memory", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err = getInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.CPU.Guest).To(Equal(uint32(2)))
			Expect(instancetypeSpec.Memory.Guest).To(Equal(resource.MustParse("256Mi")))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("should succeed with defined gpus", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(GPUFlag, "name:gpu1,devicename:nvidia"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err = getInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.GPUs).To(HaveLen(1))
			Expect(instancetypeSpec.GPUs[0].Name).To(Equal("gpu1"))
			Expect(instancetypeSpec.GPUs[0].DeviceName).To(Equal("nvidia"))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("should succeed with defined hostDevices", func(namespacedFlag string, namespaced bool) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(HostDeviceFlag, "name:device1,devicename:intel"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err = getInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec.HostDevices).To(HaveLen(1))
			Expect(instancetypeSpec.HostDevices[0].Name).To(Equal("device1"))
			Expect(instancetypeSpec.HostDevices[0].DeviceName).To(Equal("intel"))
		},
			Entry("VirtualMachineInstancetype", namespaced, true),
			Entry("VirtualMachineClusterInstancetype", "", false),
		)

		DescribeTable("should succeed with valid IOThreadsPolicy", func(namespacedFlag, param string, namespaced bool, policy v1.IOThreadsPolicy) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(IOThreadsPolicyFlag, param),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := getInstancetypeSpec(bytes, namespaced)
			Expect(err).ToNot(HaveOccurred())
			Expect(*instancetypeSpec.IOThreadsPolicy).To(Equal(policy))

		},
			Entry("VirtualMachineInstacetype set to auto", namespaced, "auto", true, v1.IOThreadsPolicyAuto),
			Entry("VirtualMachineInstacetype set to shared", namespaced, "shared", true, v1.IOThreadsPolicyShared),
			Entry("VirtualMachineClusterInstacetype set to auto", "", "auto", false, v1.IOThreadsPolicyAuto),
			Entry("VirtualMachineClusterInstacetype set to shared", "", "shared", false, v1.IOThreadsPolicyShared),
		)

		It("should create namespaced object and apply namespace when namespace is specified", func() {
			const namespace = "my-namespace"
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype,
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

		DescribeTable("invalid cpu and memory", func(cpu, memory, errMsg string) {
			err := clientcmd.NewRepeatableVirtctlCommand(create, Instancetype, namespaced,
				setFlag(CPUFlag, cpu),
				setFlag(MemoryFlag, memory),
			)()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errMsg))
		},
			Entry("Invalid cpu string value", "two", "256Mi", "invalid argument \"two\" for \"--cpu\" flag: strconv.ParseUint: parsing \"two\": invalid syntax"),
			Entry("Invalid cpu negative value", "-2", "256Mi", "invalid argument \"-2\" for \"--cpu\" flag: strconv.ParseUint: parsing \"-2\": invalid syntax"),
			Entry("Invalid memory value", "2", "256My", "quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'"),
		)

		DescribeTable("Invalid arguments", func(namespacedFlag, flag, params, errMsg string) {
			err := clientcmd.NewRepeatableVirtctlCommand(create, Instancetype, namespacedFlag,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(flag, params),
			)()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errMsg))
		},
			Entry("VirtualMachineInstacetype gpu missing name", namespaced, GPUFlag, "devicename:nvidia", fmt.Sprintf("failed to parse \"--gpu\" flag: %+s", NameErr)),
			Entry("VirtualMachineInstacetype gpu missing deviceName", namespaced, GPUFlag, "name:gpu1", fmt.Sprintf("failed to parse \"--gpu\" flag: %+s", DeviceNameErr)),
			Entry("VirtualMachineInstacetype hostdevice missing name", namespaced, HostDeviceFlag, "devicename:intel", fmt.Sprintf("failed to parse \"--hostdevice\" flag: %+s", NameErr)),
			Entry("VirtualMachineInstacetype hostdevice missing deviceName", namespaced, HostDeviceFlag, "name:device1", fmt.Sprintf("failed to parse \"--hostdevice\" flag: %+s", DeviceNameErr)),
			Entry("VirtualMachineInstacetype to IOThreadsPolicy", namespaced, IOThreadsPolicyFlag, "invalid-policy", fmt.Sprintf("failed to parse \"--iothreadspolicy\" flag: %+s", IOThreadErr)),
			Entry("VirtualMachineClusterInstacetype gpu missing name", "", GPUFlag, "devicename:nvidia", fmt.Sprintf("failed to parse \"--gpu\" flag: %+s", NameErr)),
			Entry("VirtualMachineClusterInstacetype gpu missing deviceName", "", GPUFlag, "name:gpu1", fmt.Sprintf("failed to parse \"--gpu\" flag: %+s", DeviceNameErr)),
			Entry("VirtualMachineClusterInstacetype hostdevice missing name", "", HostDeviceFlag, "devicename:intel", fmt.Sprintf("failed to parse \"--hostdevice\" flag: %+s", NameErr)),
			Entry("VirtualMachineClusterInstacetype hostdevice missing deviceName", "", HostDeviceFlag, "name:device1", fmt.Sprintf("failed to parse \"--hostdevice\" flag: %+s", DeviceNameErr)),
			Entry("VirtualMachineClusterInstacetype to IOThreadsPolicy", "", IOThreadsPolicyFlag, "invalid-policy", fmt.Sprintf("failed to parse \"--iothreadspolicy\" flag: %+s", IOThreadErr)),
		)
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func getInstancetypeSpec(bytes []byte, namespaced bool) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), bytes)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	switch obj := decodedObj.(type) {
	case *instancetypev1beta1.VirtualMachineInstancetype:
		ExpectWithOffset(1, namespaced).To(BeTrue(), "expected VirtualMachineInstancetype to be created")
		ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachineInstancetype"))
		return &obj.Spec, nil
	case *instancetypev1beta1.VirtualMachineClusterInstancetype:
		ExpectWithOffset(1, namespaced).To(BeFalse(), "expected VirtualMachineClusterInstancetype to be created")
		ExpectWithOffset(1, obj.Kind).To(Equal("VirtualMachineClusterInstancetype"))
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("object must be VirtualMachineInstance or VirtualMachineClusterInstancetype")
	}
}
