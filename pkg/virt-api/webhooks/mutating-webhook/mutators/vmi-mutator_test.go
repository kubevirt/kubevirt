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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package mutators

import (
	"encoding/json"
	"fmt"
	rt "runtime"

	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	v12 "k8s.io/api/authentication/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var privilegedUser = fmt.Sprintf("%s:%s:%s:%s", "system", "serviceaccount", "kubevirt", components.ControllerServiceAccountName)

var _ = Describe("VirtualMachineInstance Mutator", func() {
	var vmi *v1.VirtualMachineInstance
	var preset *v1.VirtualMachineInstancePreset
	var presetInformer cache.SharedIndexInformer
	var kvInformer cache.SharedIndexInformer
	var mutator *VMIsMutator

	cpuModelFromConfig := "Haswell"
	machineTypeFromConfig := "pc-q35-3.0"
	cpuReq := resource.MustParse("800m")

	admitVMI := func(arch string) *admissionv1.AdmissionResponse {
		vmi.Spec.Architecture = arch
		vmiBytes, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VMI")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Namespace: vmi.Namespace,
				Operation: admissionv1.Create,
				Resource:  k8smetav1.GroupVersionResource{Group: v1.VirtualMachineInstanceGroupVersionKind.Group, Version: v1.VirtualMachineInstanceGroupVersionKind.Version, Resource: "virtualmachineinstances"},
				Object: runtime.RawExtension{
					Raw: vmiBytes,
				},
			},
		}
		By("Mutating the VMI")
		return mutator.Mutate(ar)
	}

	getMetaSpecStatusFromAdmit := func(arch string) (*k8smetav1.ObjectMeta, *v1.VirtualMachineInstanceSpec, *v1.VirtualMachineInstanceStatus) {
		resp := admitVMI(arch)
		Expect(resp.Allowed).To(BeTrue())

		By("Getting the VMI spec from the response")
		vmiSpec := &v1.VirtualMachineInstanceSpec{}
		vmiMeta := &k8smetav1.ObjectMeta{}
		vmiStatus := &v1.VirtualMachineInstanceStatus{}
		patchOps := []patch.PatchOperation{
			{Value: vmiSpec},
			{Value: vmiMeta},
			{Value: vmiStatus},
		}
		err := json.Unmarshal(resp.Patch, &patchOps)
		Expect(err).ToNot(HaveOccurred())
		Expect(patchOps).NotTo(BeEmpty())

		return vmiMeta, vmiSpec, vmiStatus
	}

	getVMIStatusFromResponseWithUpdate := func(oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance, user string) *v1.VirtualMachineInstanceStatus {
		oldVMIBytes, err := json.Marshal(oldVMI)
		Expect(err).ToNot(HaveOccurred())
		newVMIBytes, err := json.Marshal(newVMI)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VMI")
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				UserInfo: v12.UserInfo{
					Username: user,
				},
				Operation: admissionv1.Update,
				Resource:  k8smetav1.GroupVersionResource{Group: v1.VirtualMachineInstanceGroupVersionKind.Group, Version: v1.VirtualMachineInstanceGroupVersionKind.Version, Resource: "virtualmachineinstances"},
				Object: runtime.RawExtension{
					Raw: newVMIBytes,
				},
				OldObject: runtime.RawExtension{
					Raw: oldVMIBytes,
				},
			},
		}
		By("Mutating the VMI")
		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())

		By("Getting the VMI spec from the response")
		vmiStatus := &v1.VirtualMachineInstanceStatus{}
		patchOps := []patch.PatchOperation{
			{Value: vmiStatus},
		}
		err = json.Unmarshal(resp.Patch, &patchOps)
		Expect(err).ToNot(HaveOccurred())
		if len(patchOps) == 0 {
			return &newVMI.Status
		}

		return vmiStatus
	}

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: k8smetav1.ObjectMeta{
				Labels: map[string]string{"test": "test"},
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Resources: v1.ResourceRequirements{},
				},
			},
		}

		selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{"test": "test"}}
		preset = &v1.VirtualMachineInstancePreset{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "test-preset",
			},
			Spec: v1.VirtualMachineInstancePresetSpec{
				Domain: &v1.DomainSpec{
					CPU: &v1.CPU{Cores: 4},
				},
				Selector: selector,
			},
		}
		presetInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstancePreset{})
		presetInformer.GetIndexer().Add(preset)

		mutator = &VMIsMutator{}
		mutator.ClusterConfig, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		mutator.VMIPresetInformer = presetInformer
	})

	It("should apply presets on VMI create", func() {
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.CPU).ToNot(BeNil())
		Expect(vmiSpec.Domain.CPU.Cores).To(Equal(uint32(4)))
	})

	It("should include deprecation warning in response when presets are applied to VMI", func() {
		resp := admitVMI(vmi.Spec.Architecture)
		Expect(resp.Allowed).To(BeTrue())
		Expect(resp.Warnings).ToNot(BeEmpty())
		Expect(resp.Warnings[0]).To(ContainSubstring("VirtualMachineInstancePresets is now deprecated"))
	})

	DescribeTable("should apply defaults on VMI create", func(arch string, cpuModel string) {
		// no limits wanted on this test, to not copy the limit to requests

		if arch == "" {
			if rt.GOARCH == "amd64" {
				cpuModel = v1.DefaultCPUModel
			} else {
				cpuModel = v1.CPUModeHostPassthrough
			}
		}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(arch)

		if webhooks.IsPPC64(vmiSpec) {
			Expect(vmiSpec.Domain.Machine.Type).To(Equal("pseries"))
		} else if webhooks.IsARM64(vmiSpec) {
			Expect(vmiSpec.Domain.Machine.Type).To(Equal("virt"))
		} else {
			Expect(vmiSpec.Domain.Machine.Type).To(Equal("q35"))
		}

		Expect(vmiSpec.Domain.CPU.Model).To(Equal(cpuModel))
		Expect(vmiSpec.Domain.Resources.Requests.Cpu().IsZero()).To(BeTrue())
		Expect(vmiSpec.Domain.Resources.Requests.Memory().Value()).To(Equal(int64(0)))
	},
		Entry("when architecture is amd64", "amd64", v1.DefaultCPUModel),
		Entry("when architecture is arm64", "arm64", v1.CPUModeHostPassthrough),
		Entry("when architecture is ppcle64", "ppcle64", v1.DefaultCPUModel),
		Entry("when architecture is not specified", "", v1.DefaultCPUModel))

	DescribeTable("should apply configurable defaults on VMI create", func(arch string, cpuModel string) {
		defer func() {
			webhooks.Arch = rt.GOARCH
		}()
		webhooks.Arch = arch

		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					CPUModel:   cpuModelFromConfig,
					CPURequest: &cpuReq,
					ArchitectureConfiguration: &v1.ArchConfiguration{
						Amd64:   &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
						Arm64:   &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
						Ppc64le: &v1.ArchSpecificConfiguration{MachineType: machineTypeFromConfig},
					},
				},
			},
		})

		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(arch)
		Expect(vmiSpec.Domain.CPU.Model).To(Equal(cpuModel))
		Expect(vmiSpec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))
		Expect(*vmiSpec.Domain.Resources.Requests.Cpu()).To(Equal(cpuReq))
	},
		Entry("on amd64", "amd64", cpuModelFromConfig),
		// Currently only Host-Passthrough is supported on Arm64, so you can only
		// modify the CPU Model in a VMI yaml file, rather than in cluster config
		Entry("on arm64", "arm64", v1.CPUModeHostPassthrough),
	)

	DescribeTable("it should", func(given []v1.Volume, expected []v1.Volume) {
		vmi.Spec.Volumes = given
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Volumes).To(Equal(expected))
	},
		Entry("set the ImagePullPolicy to Always if :latest is specified",
			[]v1.Volume{
				{
					Name: "a",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "test:latest",
						},
					},
				},
			},
			[]v1.Volume{
				{
					Name: "a",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test:latest",
							ImagePullPolicy: k8sv1.PullAlways,
						},
					},
				},
			},
		),
		Entry("set the ImagePullPolicy to Always if no tag or shasum is specified",
			[]v1.Volume{
				{
					Name: "a",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "test",
						},
					},
				},
			},
			[]v1.Volume{
				{
					Name: "a",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test",
							ImagePullPolicy: k8sv1.PullAlways,
						},
					},
				},
			},
		),
		Entry("set the ImagePullPolicy to IfNotPresent if arbitrary tags are specified",
			[]v1.Volume{
				{
					Name: "a",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "test:notlatest",
						},
					},
				},
			},
			[]v1.Volume{
				{
					Name: "a",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test:notlatest",
							ImagePullPolicy: k8sv1.PullIfNotPresent,
						},
					},
				},
			},
		),
		Entry("set the right ImagePullPolicy on a mixture of sources",
			[]v1.Volume{
				{
					Name: "a",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "test:notlatest",
						},
					},
				},
				{
					Name: "b",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "test:latest",
						},
					},
				},
				{
					Name: "c",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test:latest",
							ImagePullPolicy: k8sv1.PullNever,
						},
					},
				},
				{
					Name: "d",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test",
							ImagePullPolicy: k8sv1.PullIfNotPresent,
						},
					},
				},
				{
					Name: "e",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "test:notlatest",
						},
					},
				},
			},
			[]v1.Volume{
				{
					Name: "a",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test:notlatest",
							ImagePullPolicy: k8sv1.PullIfNotPresent,
						},
					},
				},
				{
					Name: "b",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test:latest",
							ImagePullPolicy: k8sv1.PullAlways,
						},
					},
				},
				{
					Name: "c",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test:latest",
							ImagePullPolicy: k8sv1.PullNever,
						},
					},
				},
				{
					Name: "d",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test",
							ImagePullPolicy: k8sv1.PullIfNotPresent,
						},
					},
				},
				{
					Name: "e",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "test:notlatest",
							ImagePullPolicy: k8sv1.PullIfNotPresent,
						},
					},
				},
			},
		),
	)

	DescribeTable("should add the default network interface",
		func(iface string) {
			expectedIface := "bridge"
			switch iface {
			case "masquerade":
				expectedIface = "masquerade"
			case "slirp":
				expectedIface = "slirp"
			}

			permit := true
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						NetworkConfiguration: &v1.NetworkConfiguration{
							NetworkInterface:     expectedIface,
							PermitSlirpInterface: &permit,
						},
					},
				},
			})

			_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
			switch expectedIface {
			case "bridge":
				Expect(vmiSpec.Domain.Devices.Interfaces[0].Bridge).NotTo(BeNil())
			case "masquerade":
				Expect(vmiSpec.Domain.Devices.Interfaces[0].Masquerade).NotTo(BeNil())
			case "slirp":
				Expect(vmiSpec.Domain.Devices.Interfaces[0].Slirp).NotTo(BeNil())
			}
		},
		Entry("as bridge", "bridge"),
		Entry("as masquerade", "masquerade"),
		Entry("as slirp", "slirp"),
	)

	DescribeTable("should not add the default interfaces if", func(interfaces []v1.Interface, networks []v1.Network) {
		vmi.Spec.Domain.Devices.Interfaces = append([]v1.Interface{}, interfaces...)
		vmi.Spec.Networks = append([]v1.Network{}, networks...)
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		if len(interfaces) == 0 {
			Expect(vmiSpec.Domain.Devices.Interfaces).To(BeNil())
		} else {
			Expect(vmiSpec.Domain.Devices.Interfaces).To(Equal(interfaces))
		}
		if len(networks) == 0 {
			Expect(vmiSpec.Networks).To(BeNil())
		} else {
			Expect(vmiSpec.Networks).To(Equal(networks))
		}
	},
		Entry("interfaces and networks are non-empty", []v1.Interface{{Name: "a"}}, []v1.Network{{Name: "b"}}),
		Entry("interfaces is non-empty", []v1.Interface{{Name: "a"}}, []v1.Network{}),
		Entry("networks is non-empty", []v1.Interface{}, []v1.Network{{Name: "b"}}),
	)

	It("should add a missing volume disk", func() {
		presentVolumeName := "present-vol"
		missingVolumeName := "missing-vol"
		vmi.Spec.Domain.Devices.Disks = []v1.Disk{
			v1.Disk{
				Name: presentVolumeName,
			},
		}
		vmi.Spec.Volumes = []v1.Volume{
			v1.Volume{
				Name: presentVolumeName,
			},
			v1.Volume{
				Name: missingVolumeName,
			},
		}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.Devices.Disks).To(HaveLen(2))
		Expect(vmiSpec.Domain.Devices.Disks[0].Name).To(Equal(presentVolumeName))
		Expect(vmiSpec.Domain.Devices.Disks[1].Name).To(Equal(missingVolumeName))
	})

	It("should set defaults for input devices", func() {
		vmi.Spec.Domain.Devices.Inputs = []v1.Input{{
			Name: "default-0",
		}}

		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.Devices.Inputs).To(HaveLen(1))
		Expect(vmiSpec.Domain.Devices.Inputs[0].Name).To(Equal("default-0"))
		Expect(vmiSpec.Domain.Devices.Inputs[0].Bus).To(Equal(v1.InputBusUSB))
		Expect(vmiSpec.Domain.Devices.Inputs[0].Type).To(Equal(v1.InputTypeTablet))
	})

	It("should not override specified properties with defaults on VMI create", func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					CPUModel:    cpuModelFromConfig,
					MachineType: machineTypeFromConfig,
					CPURequest:  &cpuReq,
				},
			},
		})

		vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			k8sv1.ResourceCPU:    resource.MustParse("600m"),
			k8sv1.ResourceMemory: resource.MustParse("512Mi"),
		}
		vmi.Spec.Domain.CPU = &v1.CPU{Model: "EPYC"}
		vmi.Spec.Domain.Machine = &v1.Machine{Type: "q35"}

		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.CPU.Model).To(Equal(vmi.Spec.Domain.CPU.Model))
		Expect(vmiSpec.Domain.Machine.Type).To(Equal(vmi.Spec.Domain.Machine.Type))
		Expect(vmiSpec.Domain.Resources.Requests.Cpu()).To(Equal(vmi.Spec.Domain.Resources.Requests.Cpu()))
		Expect(vmiSpec.Domain.Resources.Requests.Memory()).To(Equal(vmi.Spec.Domain.Resources.Requests.Memory()))
	})

	It("should convert CPU requests to sockets", func() {
		vmi.Spec.Domain.CPU = &v1.CPU{Model: "EPYC"}
		vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			k8sv1.ResourceCPU: resource.MustParse("2200m"),
		}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)

		Expect(vmiSpec.Domain.CPU.Cores).To(Equal(uint32(1)), "Expect cores")
		Expect(vmiSpec.Domain.CPU.Sockets).To(Equal(uint32(3)), "Expect sockets")
		Expect(vmiSpec.Domain.CPU.Threads).To(Equal(uint32(1)), "Expect threads")
	})

	It("should convert CPU limits to sockets", func() {
		vmi.Spec.Domain.CPU = &v1.CPU{Model: "EPYC"}
		vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			k8sv1.ResourceCPU: resource.MustParse("2.3"),
		}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)

		Expect(vmiSpec.Domain.CPU.Cores).To(Equal(uint32(1)), "Expect cores")
		Expect(vmiSpec.Domain.CPU.Sockets).To(Equal(uint32(3)), "Expect sockets")
		Expect(vmiSpec.Domain.CPU.Threads).To(Equal(uint32(1)), "Expect threads")
	})

	It("should apply memory-overcommit when guest-memory is set and memory-request is not set", func() {
		// no limits wanted on this test, to not copy the limit to requests
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						MemoryOvercommit: 150,
					},
				},
			},
		})

		guestMemory := resource.MustParse("3072M")
		vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.Memory.Guest.String()).To(Equal("3072M"))
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("2048M"))
	})

	It("should apply memory-overcommit when hugepages are set and memory-request is not set", func() {
		// no limits wanted on this test, to not copy the limit to requests
		vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "3072M"}}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.Memory.Hugepages.PageSize).To(Equal("3072M"))
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("3072M"))
	})

	It("should not apply memory overcommit when memory-request and guest-memory are set", func() {
		vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			k8sv1.ResourceMemory: resource.MustParse("512M"),
		}
		guestMemory := resource.MustParse("4096M")
		vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("512M"))
		Expect(vmiSpec.Domain.Memory.Guest.String()).To(Equal("4096M"))
	})

	It("should apply foreground finalizer on VMI create", func() {
		vmiMeta, _, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiMeta.Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))
	})

	It("should copy cpu limits to requests if only limits are set", func() {
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{
			Requests: k8sv1.ResourceList{},
			Limits: k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("1"),
			},
		}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.Resources.Requests.Cpu().String()).To(Equal("1"))
		Expect(vmiSpec.Domain.Resources.Limits.Cpu().String()).To(Equal("1"))
	})

	It("should copy memory limits to requests if only limits are set", func() {
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{
			Requests: k8sv1.ResourceList{},
			Limits: k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64M"),
			},
		}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("64M"))
		Expect(vmiSpec.Domain.Resources.Limits.Memory().String()).To(Equal("64M"))
	})

	It("should set the hyperv dependencies", func() {
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				SyNICTimer: &v1.SyNICTimer{
					Enabled: pointer.Bool(true),
				},
			},
		}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(*(vmiSpec.Domain.Features.Hyperv.VPIndex.Enabled)).To(BeTrue())
		Expect(*(vmiSpec.Domain.Features.Hyperv.SyNIC.Enabled)).To(BeTrue())
		Expect(*(vmiSpec.Domain.Features.Hyperv.SyNICTimer.Enabled)).To(BeTrue())
	})

	It("Should not mutate VMIs without HyperV configuration", func() {
		vmi := api.NewMinimalVMI("testvmi")
		Expect(vmi.Spec.Domain.Features).To(BeNil())
		err := webhooks.SetHypervFeatureDependencies(&vmi.Spec)
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Spec.Domain.Features).To(BeNil())
	})

	It("Should not mutate VMIs with empty HyperV configuration", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{},
		}
		err := webhooks.SetHypervFeatureDependencies(&vmi.Spec)
		Expect(err).ToNot(HaveOccurred())
		hyperv := v1.FeatureHyperv{}
		ok := equality.Semantic.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should not mutate VMIs with hyperv configuration without deps", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: pointer.Bool(true),
				},
				Runtime: &v1.FeatureState{
					Enabled: pointer.Bool(true),
				},
				Reset: &v1.FeatureState{
					Enabled: pointer.Bool(true),
				},
			},
		}
		err := webhooks.SetHypervFeatureDependencies(&vmi.Spec)
		Expect(err).ToNot(HaveOccurred())

		hyperv := v1.FeatureHyperv{
			Relaxed: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
			Runtime: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
			Reset: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
		}

		ok := equality.Semantic.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should mutate VMIs with hyperv configuration to fix deps", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: pointer.Bool(true),
				},
				SyNICTimer: &v1.SyNICTimer{
					Enabled: pointer.Bool(true),
				},
			},
		}
		err := webhooks.SetHypervFeatureDependencies(&vmi.Spec)
		Expect(err).ToNot(HaveOccurred())

		hyperv := v1.FeatureHyperv{
			Relaxed: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
			VPIndex: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
			SyNIC: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
			SyNICTimer: &v1.SyNICTimer{
				Enabled: pointer.Bool(true),
			},
		}

		ok := equality.Semantic.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should partially mutate VMIs with explicit hyperv configuration", func() {
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				VPIndex: &v1.FeatureState{
					Enabled: pointer.Bool(false),
				},
				// should enable SyNIC
				SyNICTimer: &v1.SyNICTimer{
					Enabled: pointer.Bool(true),
				},
				EVMCS: &v1.FeatureState{
					Enabled: pointer.Bool(true),
				},
			},
		}
		webhooks.SetHypervFeatureDependencies(&vmi.Spec)

		// we MUST report the error in mutation, but production code is
		// supposed to ignore it to fulfill the design semantics, see
		// the discussion in https://github.com/kubevirt/kubevirt/pull/2408

		hyperv := v1.FeatureHyperv{
			VPIndex: &v1.FeatureState{
				Enabled: pointer.Bool(false),
			},
			SyNIC: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
			SyNICTimer: &v1.SyNICTimer{
				Enabled: pointer.Bool(true),
			},
			EVMCS: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
			VAPIC: &v1.FeatureState{
				Enabled: pointer.Bool(true),
			},
		}

		ok := equality.Semantic.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})
	DescribeTable("modify the VMI status", func(user string, shouldChange bool) {
		oldVMI := &v1.VirtualMachineInstance{}
		oldVMI.Status = v1.VirtualMachineInstanceStatus{
			Phase: v1.Running,
		}
		newVMI := oldVMI.DeepCopy()
		newVMI.Status = v1.VirtualMachineInstanceStatus{
			Phase: v1.Failed,
		}
		status := getVMIStatusFromResponseWithUpdate(oldVMI, newVMI, user)
		if shouldChange {
			Expect(&newVMI.Status).To(Equal(status))
		} else {
			Expect(&oldVMI.Status).To(Equal(status))
		}
	},
		Entry("if our service accounts modfies it", privilegedUser, true),
		Entry("not if the user is not one of ours", "unknown", false),
	)

	// Check following convert for ARM64
	// 1. should convert CPU model to host-passthrough
	// 2. should convert default bootloader to UEFI non secureboot
	It("should convert cpu model, AutoattachGraphicsDevice and UEFI boot on ARM64", func() {
		// turn on arm validation/mutation
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit("arm64")
		Expect(*(vmiSpec.Domain.Firmware.Bootloader.EFI.SecureBoot)).To(BeFalse())
		Expect(vmiSpec.Domain.CPU.Model).To(Equal("host-passthrough"))
	})

	DescribeTable("should convert disk bus to virtio or scsi on ARM64", func(given v1.Disk, diskType string, expectedBus v1.DiskBus) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "a",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{
					Image: "test:latest",
				},
			},
		})
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, given)
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit("arm64")

		getDiskDeviceBus := func(device string) v1.DiskBus {
			switch device {
			case "Disk":
				return vmiSpec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus
			case "CDRom":
				return vmiSpec.Domain.Devices.Disks[0].DiskDevice.CDRom.Bus
			case "LUN":
				return vmiSpec.Domain.Devices.Disks[0].DiskDevice.LUN.Bus
			default:
				return ""
			}
		}

		Expect(getDiskDeviceBus(diskType)).Should(Equal(expectedBus))
	},
		Entry("Disk device",
			v1.Disk{
				Name: "a",
			}, "Disk", v1.DiskBusVirtio),

		Entry("Disk device with virtio bus",
			v1.Disk{
				Name: "a",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "scsi",
					},
				},
			}, "Disk", v1.DiskBusSCSI),

		Entry("CDRom device",
			v1.Disk{
				Name: "a",
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{},
				},
			}, "CDRom", v1.DiskBusVirtio),

		Entry("LUN device",
			v1.Disk{
				Name: "a",
				DiskDevice: v1.DiskDevice{
					LUN: &v1.LunTarget{},
				},
			}, "LUN", v1.DiskBusVirtio),
	)

	var (
		vmxFeature = v1.CPUFeature{
			Name:   nodelabellerutil.VmxFeature,
			Policy: nodelabellerutil.RequirePolicy,
		}
		cpuFeatures = []v1.CPUFeature{
			vmxFeature,
		}
	)

	DescribeTable("modify the VMI cpu feature ", func(vmi *v1.VirtualMachineInstance, hyperv *v1.FeatureHyperv, resultCPUTopology *v1.CPU) {
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: hyperv,
		}
		err := webhooks.SetHypervFeatureDependencies(&vmi.Spec)
		Expect(err).ToNot(HaveOccurred(), "it should not fail")
		if resultCPUTopology == nil {
			Expect(vmi.Spec.Domain.CPU).To(BeNil(), "cpu topology should not be updated")
		} else {
			Expect(vmi.Spec.Domain.CPU).To(Equal(resultCPUTopology), "cpu topologies should equal")
		}

	},
		Entry("if hyperV doesn't contain EVMCS", api.NewMinimalVMI("testvmi"),
			&v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: pointer.Bool(true),
				},
			}, nil),

		Entry("if EVMCS is explicitly false ", api.NewMinimalVMI("testvmi"),
			&v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
			},
			nil,
		),

		Entry("if hyperV does contain EVMCS", api.NewMinimalVMI("testvmi"),
			&v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{},
			}, &v1.CPU{
				Features: cpuFeatures,
			}),

		Entry("if EVMCS is explicitly true ", api.NewMinimalVMI("testvmi"),
			&v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
			}, &v1.CPU{
				Features: cpuFeatures,
			}),

		Entry("if hyperV does contain EVMCS and cpu sockets ", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Sockets: 2,
					},
				},
			},
		},
			&v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{},
			}, &v1.CPU{
				Sockets:  2,
				Features: cpuFeatures,
			}),

		Entry("if hyperV does contain EVMCS and 0 cpu features ", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Features: []v1.CPUFeature{},
					},
				},
			},
		},
			&v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{},
			}, &v1.CPU{
				Features: cpuFeatures,
			}),

		Entry("if hyperV does contain EVMCS and 1 different cpu feature ", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Features: []v1.CPUFeature{
							{
								Name:   "monitor",
								Policy: nodelabellerutil.RequirePolicy,
							},
						},
					},
				},
			},
		},
			&v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{},
			}, &v1.CPU{
				Features: []v1.CPUFeature{
					{
						Name:   "monitor",
						Policy: nodelabellerutil.RequirePolicy,
					},
					vmxFeature,
				},
			}),

		Entry("if hyperV does contain EVMCS and disabled vmx cpu feature ", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Features: []v1.CPUFeature{
							{
								Name:   nodelabellerutil.VmxFeature,
								Policy: "disabled",
							},
						},
					},
				},
			},
		},
			&v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{},
			}, &v1.CPU{
				Features: cpuFeatures,
			}),
		Entry("if hyperV does contain EVMCS and enabled vmx cpu feature ", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					CPU: &v1.CPU{
						Features: []v1.CPUFeature{
							{
								Name:   nodelabellerutil.VmxFeature,
								Policy: nodelabellerutil.RequirePolicy,
							},
						},
					},
				},
			},
		},
			&v1.FeatureHyperv{
				EVMCS: &v1.FeatureState{},
			}, &v1.CPU{
				Features: cpuFeatures,
			}),
	)

	When("Root feature gate is enabled", func() {

		BeforeEach(func() {
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{virtconfig.Root},
						},
					},
				},
			})
		})

		It("Should not tag vmi as non-root ", func() {
			_, _, status := getMetaSpecStatusFromAdmit(rt.GOARCH)
			Expect(status.RuntimeUser).To(BeZero())
		})
	})
	It("Should tag vmi as non-root ", func() {
		_, _, status := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(status.RuntimeUser).NotTo(BeZero())
	})

	It("should add realtime node label selector with realtime workload", func() {
		vmi.Spec.Domain.CPU = &v1.CPU{Realtime: &v1.Realtime{}}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.NodeSelector).NotTo(BeNil())
		Expect(vmiSpec.NodeSelector).To(BeEquivalentTo(map[string]string{v1.RealtimeLabel: ""}))
	})
	It("should not add realtime node label selector when no realtime workload", func() {
		vmi.Spec.Domain.CPU = &v1.CPU{Realtime: nil}
		vmi.Spec.NodeSelector = map[string]string{v1.NodeSchedulable: "true"}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.NodeSelector).To(BeEquivalentTo(map[string]string{v1.NodeSchedulable: "true"}))
	})
	It("should not overwrite existing node label selectors with realtime workload", func() {
		vmi.Spec.Domain.CPU = &v1.CPU{Realtime: &v1.Realtime{}}
		vmi.Spec.NodeSelector = map[string]string{v1.NodeSchedulable: "true"}
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.NodeSelector).To(BeEquivalentTo(map[string]string{v1.NodeSchedulable: "true", v1.RealtimeLabel: ""}))
	})

	DescribeTable("When scheduling SEV workloads",
		func(nodeSelectorBefore map[string]string,
			nodeSelectorAfter map[string]string,
			launchSec *v1.LaunchSecurity) {
			vmi.Spec.NodeSelector = nodeSelectorBefore
			vmi.Spec.Domain.LaunchSecurity = launchSec
			_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
			Expect(vmiSpec.NodeSelector).NotTo(BeNil())
			Expect(vmiSpec.NodeSelector).To(BeEquivalentTo(nodeSelectorAfter))
		},
		Entry("It should add SEV node label selector with SEV workload",
			map[string]string{},
			map[string]string{v1.SEVLabel: ""},
			&v1.LaunchSecurity{SEV: &v1.SEV{}}),
		Entry("It should not add SEV node label selector when no SEV workload",
			map[string]string{v1.NodeSchedulable: "true"},
			map[string]string{v1.NodeSchedulable: "true"},
			&v1.LaunchSecurity{}),
		Entry("It should not overwrite existing node label selectors with SEV workload",
			map[string]string{v1.NodeSchedulable: "true"},
			map[string]string{v1.NodeSchedulable: "true", v1.SEVLabel: ""},
			&v1.LaunchSecurity{SEV: &v1.SEV{}}),
		Entry("It should add SEV and SEV-ES node label selector with SEV-ES workload",
			map[string]string{},
			map[string]string{
				v1.SEVLabel:   "",
				v1.SEVESLabel: "",
			},
			&v1.LaunchSecurity{
				SEV: &v1.SEV{
					Policy: &v1.SEVPolicy{
						EncryptedState: pointer.Bool(true),
					},
				},
			}),
		Entry("It should not add SEV-ES node label selector when no SEV policy is set",
			map[string]string{v1.NodeSchedulable: "true"},
			map[string]string{
				v1.NodeSchedulable: "true",
				v1.SEVLabel:        "",
			},
			&v1.LaunchSecurity{
				SEV: &v1.SEV{
					Policy: &v1.SEVPolicy{},
				},
			}),
		Entry("It should not add SEV-ES node label selector when no SEV-ES policy bit is set",
			map[string]string{v1.NodeSchedulable: "true"},
			map[string]string{
				v1.NodeSchedulable: "true",
				v1.SEVLabel:        "",
			},
			&v1.LaunchSecurity{
				SEV: &v1.SEV{
					Policy: &v1.SEVPolicy{
						EncryptedState: nil,
					},
				},
			}),
		Entry("It should not add SEV-ES node label selector when SEV-ES policy bit is set to false",
			map[string]string{v1.NodeSchedulable: "true"},
			map[string]string{
				v1.NodeSchedulable: "true",
				v1.SEVLabel:        "",
			},
			&v1.LaunchSecurity{
				SEV: &v1.SEV{
					Policy: &v1.SEVPolicy{
						EncryptedState: pointer.Bool(false),
					},
				},
			}),
		Entry("It should not overwrite existing node label selectors with SEV-ES workload",
			map[string]string{v1.NodeSchedulable: "true"},
			map[string]string{
				v1.NodeSchedulable: "true",
				v1.SEVLabel:        "",
				v1.SEVESLabel:      "",
			},
			&v1.LaunchSecurity{
				SEV: &v1.SEV{
					Policy: &v1.SEVPolicy{
						EncryptedState: pointer.Bool(true),
					},
				},
			}),
	)

	DescribeTable("evictionStrategy should match the", func(f func(*v1.VirtualMachineInstanceSpec) v1.EvictionStrategy) {
		expected := f(&vmi.Spec)
		_, vmiSpec, _ := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(vmiSpec.EvictionStrategy).ToNot(BeNil())
		Expect(*vmiSpec.EvictionStrategy).To(Equal(expected))
	},
		Entry("one set in the VMI", func(s *v1.VirtualMachineInstanceSpec) v1.EvictionStrategy {
			liveMigrate := v1.EvictionStrategyLiveMigrate
			s.EvictionStrategy = &liveMigrate
			return liveMigrate
		}),
		Entry("one set cluster-wide", func(*v1.VirtualMachineInstanceSpec) v1.EvictionStrategy {
			noneStrategy := v1.EvictionStrategyNone

			kvCR := testutils.GetFakeKubeVirtClusterConfig(kvInformer)
			kvCR.Spec.Configuration.EvictionStrategy = &noneStrategy
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvCR)

			return noneStrategy
		}),
		Entry("one set in the VMI if both cluster-wide and VMI are set", func(s *v1.VirtualMachineInstanceSpec) v1.EvictionStrategy {
			clusterStrategy := v1.EvictionStrategyLiveMigrate
			vmiStrategy := v1.EvictionStrategyNone

			kvCR := testutils.GetFakeKubeVirtClusterConfig(kvInformer)
			kvCR.Spec.Configuration.EvictionStrategy = &clusterStrategy
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvCR)

			s.EvictionStrategy = &vmiStrategy

			return vmiStrategy
		}),
		Entry("default one if nothing is set", func(s *v1.VirtualMachineInstanceSpec) v1.EvictionStrategy {
			s.EvictionStrategy = nil

			kvCR := testutils.GetFakeKubeVirtClusterConfig(kvInformer)
			kvCR.Spec.Configuration.EvictionStrategy = nil
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvCR)

			defaultStrategy := mutator.ClusterConfig.GetDefaultClusterConfig().EvictionStrategy
			Expect(defaultStrategy).ToNot(BeNil())
			return *defaultStrategy
		}),
	)

	It("should set guest memory status on VMI creation", func() {
		memory := resource.MustParse("128Mi")
		vmi.Spec.Domain.Memory = &v1.Memory{
			Guest: &memory,
		}
		_, _, status := getMetaSpecStatusFromAdmit(rt.GOARCH)
		Expect(status.Memory).ToNot(BeNil())
		Expect(status.Memory.GuestAtBoot).ToNot(BeNil())
		Expect(status.Memory.GuestCurrent).ToNot(BeNil())
		Expect(status.Memory.GuestRequested).ToNot(BeNil())
		Expect(*status.Memory.GuestAtBoot).To(Equal(memory))
		Expect(*status.Memory.GuestCurrent).To(Equal(memory))
		Expect(*status.Memory.GuestRequested).To(Equal(memory))
	})
})
