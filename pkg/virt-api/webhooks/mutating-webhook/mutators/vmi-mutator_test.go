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
	"reflect"
	rt "runtime"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/api/admission/v1beta1"
	v12 "k8s.io/api/authentication/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
)

var privilegedUser = fmt.Sprintf("%s:%s:%s:%s", "system", "serviceaccount", "kubevirt", rbac.ControllerServiceAccountName)

var _ = Describe("VirtualMachineInstance Mutator", func() {
	var vmi *v1.VirtualMachineInstance
	var preset *v1.VirtualMachineInstancePreset
	var presetInformer cache.SharedIndexInformer
	var namespaceLimit *k8sv1.LimitRange
	var namespaceLimitInformer cache.SharedIndexInformer
	var configMapInformer cache.SharedIndexInformer
	var mutator *VMIsMutator
	var _true bool = true
	var _false bool = false

	memoryLimit := "128M"
	cpuModelFromConfig := "Haswell"
	machineTypeFromConfig := "pc-q35-3.0"
	cpuRequestFromConfig := "800m"

	getVMISpecMetaFromResponse := func() (*v1.VirtualMachineInstanceSpec, *k8smetav1.ObjectMeta) {
		vmiBytes, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VMI")
		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Operation: v1beta1.Create,
				Resource:  k8smetav1.GroupVersionResource{Group: v1.VirtualMachineInstanceGroupVersionKind.Group, Version: v1.VirtualMachineInstanceGroupVersionKind.Version, Resource: "virtualmachineinstances"},
				Object: runtime.RawExtension{
					Raw: vmiBytes,
				},
			},
		}
		By("Mutating the VMI")
		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(BeTrue())

		By("Getting the VMI spec from the response")
		vmiSpec := &v1.VirtualMachineInstanceSpec{}
		vmiMeta := &k8smetav1.ObjectMeta{}
		patch := []patchOperation{
			{Value: vmiSpec},
			{Value: vmiMeta},
		}
		err = json.Unmarshal(resp.Patch, &patch)
		Expect(err).ToNot(HaveOccurred())
		Expect(patch).NotTo(BeEmpty())

		return vmiSpec, vmiMeta
	}

	getVMIStatusFromResponse := func(oldVMI *v1.VirtualMachineInstance, newVMI *v1.VirtualMachineInstance, user string) *v1.VirtualMachineInstanceStatus {
		oldVMIBytes, err := json.Marshal(oldVMI)
		Expect(err).ToNot(HaveOccurred())
		newVMIBytes, err := json.Marshal(newVMI)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VMI")
		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				UserInfo: v12.UserInfo{
					Username: user,
				},
				Operation: v1beta1.Update,
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
		patch := []patchOperation{
			{Value: vmiStatus},
		}
		err = json.Unmarshal(resp.Patch, &patch)
		Expect(err).ToNot(HaveOccurred())
		if len(patch) == 0 {
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

		namespaceLimit = &k8sv1.LimitRange{
			Spec: k8sv1.LimitRangeSpec{
				Limits: []k8sv1.LimitRangeItem{
					{
						Type: k8sv1.LimitTypeContainer,
						Default: k8sv1.ResourceList{
							k8sv1.ResourceMemory: resource.MustParse(memoryLimit),
						},
					},
				},
			},
		}
		namespaceLimitInformer, _ = testutils.NewFakeInformerFor(&k8sv1.LimitRange{})
		namespaceLimitInformer.GetIndexer().Add(namespaceLimit)
		webhooks.SetInformers(
			&webhooks.Informers{
				VMIPresetInformer:       presetInformer,
				NamespaceLimitsInformer: namespaceLimitInformer,
			},
		)

		mutator = &VMIsMutator{}
		mutator.ClusterConfig, configMapInformer, _, _ = testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
	})

	It("should apply presets on VMI create", func() {
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.CPU).ToNot(BeNil())
		Expect(vmiSpec.Domain.CPU.Cores).To(Equal(uint32(4)))
	})

	It("should apply namespace limit ranges on VMI create", func() {
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.Resources.Limits.Memory().String()).To(Equal(memoryLimit))
	})

	It("should apply defaults on VMI create", func() {
		// no limits wanted on this test, to not copy the limit to requests
		namespaceLimitInformer, _ = testutils.NewFakeInformerFor(&k8sv1.LimitRange{})
		webhooks.SetInformers(
			&webhooks.Informers{
				VMIPresetInformer:       presetInformer,
				NamespaceLimitsInformer: namespaceLimitInformer,
			},
		)
		vmiSpec, _ := getVMISpecMetaFromResponse()
		if rt.GOARCH == "ppc64le" {
			Expect(vmiSpec.Domain.Machine.Type).To(Equal("pseries"))
		} else {
			Expect(vmiSpec.Domain.Machine.Type).To(Equal("q35"))
		}
		Expect(vmiSpec.Domain.CPU.Model).To(Equal(""))
		Expect(vmiSpec.Domain.Resources.Requests.Cpu().String()).To(Equal("100m"))
		// no default for requested memory when no memory is specified
		Expect(vmiSpec.Domain.Resources.Requests.Memory().Value()).To(Equal(int64(0)))
	})

	It("should apply configurable defaults on VMI create", func() {
		// no limits wanted on this test, to not copy the limit to requests
		namespaceLimitInformer, _ = testutils.NewFakeInformerFor(&k8sv1.LimitRange{})
		webhooks.SetInformers(
			&webhooks.Informers{
				VMIPresetInformer:       presetInformer,
				NamespaceLimitsInformer: namespaceLimitInformer,
			},
		)
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{
				virtconfig.CPUModelKey:    cpuModelFromConfig,
				virtconfig.MachineTypeKey: machineTypeFromConfig,
				virtconfig.CPURequestKey:  cpuRequestFromConfig,
			},
		})

		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.CPU.Model).To(Equal(cpuModelFromConfig))
		Expect(vmiSpec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))
		Expect(vmiSpec.Domain.Resources.Requests.Cpu().String()).To(Equal(cpuRequestFromConfig))
	})

	table.DescribeTable("it should", func(given []v1.Volume, expected []v1.Volume) {
		vmi.Spec.Volumes = given
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Volumes).To(Equal(expected))
	},
		table.Entry("set the ImagePullPolicy to Always if :latest is specified",
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
		table.Entry("set the ImagePullPolicy to Always if no tag or shasum is specified",
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
		table.Entry("set the ImagePullPolicy to IfNotPresent if arbitrary tags are specified",
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
		table.Entry("set the right ImagePullPolicy on a mixture of sources",
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

	table.DescribeTable("should add the default network interface",
		func(iface string) {
			expectedIface := "bridge"
			switch iface {
			case "masquerade":
				expectedIface = "masquerade"
			case "slirp":
				expectedIface = "slirp"
			}

			testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
				Data: map[string]string{
					virtconfig.NetworkInterfaceKey:  expectedIface,
					virtconfig.PermitSlirpInterface: "true",
				},
			})

			vmiSpec, _ := getVMISpecMetaFromResponse()
			switch expectedIface {
			case "bridge":
				Expect(vmiSpec.Domain.Devices.Interfaces[0].Bridge).NotTo(BeNil())
			case "masquerade":
				Expect(vmiSpec.Domain.Devices.Interfaces[0].Masquerade).NotTo(BeNil())
			case "slirp":
				Expect(vmiSpec.Domain.Devices.Interfaces[0].Slirp).NotTo(BeNil())
			}
		},
		table.Entry("as bridge", "bridge"),
		table.Entry("as masquerade", "masquerade"),
		table.Entry("as slirp", "slirp"),
	)

	table.DescribeTable("should not add the default interfaces if", func(interfaces []v1.Interface, networks []v1.Network) {
		vmi.Spec.Domain.Devices.Interfaces = append([]v1.Interface{}, interfaces...)
		vmi.Spec.Networks = append([]v1.Network{}, networks...)
		vmiSpec, _ := getVMISpecMetaFromResponse()
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
		table.Entry("interfaces and networks are non-empty", []v1.Interface{{Name: "a"}}, []v1.Network{{Name: "b"}}),
		table.Entry("interfaces is non-empty", []v1.Interface{{Name: "a"}}, []v1.Network{}),
		table.Entry("networks is non-empty", []v1.Interface{}, []v1.Network{{Name: "b"}}),
	)

	It("should not override specified properties with defaults on VMI create", func() {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{
				virtconfig.CPUModelKey:    cpuModelFromConfig,
				virtconfig.MachineTypeKey: machineTypeFromConfig,
				virtconfig.CPURequestKey:  cpuRequestFromConfig,
			},
		})

		vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			k8sv1.ResourceCPU:    resource.MustParse("600m"),
			k8sv1.ResourceMemory: resource.MustParse("512Mi"),
		}
		vmi.Spec.Domain.CPU = &v1.CPU{Model: "EPYC"}
		vmi.Spec.Domain.Machine.Type = "q35"

		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.CPU.Model).To(Equal(vmi.Spec.Domain.CPU.Model))
		Expect(vmiSpec.Domain.Machine.Type).To(Equal(vmi.Spec.Domain.Machine.Type))
		Expect(vmiSpec.Domain.Resources.Requests.Cpu()).To(Equal(vmi.Spec.Domain.Resources.Requests.Cpu()))
		Expect(vmiSpec.Domain.Resources.Requests.Memory()).To(Equal(vmi.Spec.Domain.Resources.Requests.Memory()))
	})

	It("should apply memory-overcommit when guest-memory is set and memory-request is not set", func() {
		// no limits wanted on this test, to not copy the limit to requests
		namespaceLimitInformer, _ = testutils.NewFakeInformerFor(&k8sv1.LimitRange{})
		webhooks.SetInformers(
			&webhooks.Informers{
				VMIPresetInformer:       presetInformer,
				NamespaceLimitsInformer: namespaceLimitInformer,
			},
		)
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{
				virtconfig.MemoryOvercommitKey: "150",
			},
		})
		guestMemory := resource.MustParse("3072M")
		vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.Memory.Guest.String()).To(Equal("3072M"))
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("2048M"))
	})

	It("should apply memory-overcommit when hugepages are set and memory-request is not set", func() {
		// no limits wanted on this test, to not copy the limit to requests
		namespaceLimitInformer, _ = testutils.NewFakeInformerFor(&k8sv1.LimitRange{})
		webhooks.SetInformers(
			&webhooks.Informers{
				VMIPresetInformer:       presetInformer,
				NamespaceLimitsInformer: namespaceLimitInformer,
			},
		)
		vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "3072M"}}
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.Memory.Hugepages.PageSize).To(Equal("3072M"))
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("3072M"))
	})

	It("should not apply memory overcommit when memory-request and guest-memory are set", func() {
		vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
			k8sv1.ResourceMemory: resource.MustParse("512M"),
		}
		guestMemory := resource.MustParse("4096M")
		vmi.Spec.Domain.Memory = &v1.Memory{Guest: &guestMemory}
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("512M"))
		Expect(vmiSpec.Domain.Memory.Guest.String()).To(Equal("4096M"))
	})

	It("should apply foreground finalizer on VMI create", func() {
		_, vmiMeta := getVMISpecMetaFromResponse()
		Expect(vmiMeta.Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))
	})

	It("should copy cpu limits to requests if only limits are set", func() {
		vmi.Spec.Domain.Resources = v1.ResourceRequirements{
			Requests: k8sv1.ResourceList{},
			Limits: k8sv1.ResourceList{
				k8sv1.ResourceCPU: resource.MustParse("1"),
			},
		}
		vmiSpec, _ := getVMISpecMetaFromResponse()
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
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("64M"))
		Expect(vmiSpec.Domain.Resources.Limits.Memory().String()).To(Equal("64M"))
	})

	It("should set the hyperv dependencies", func() {
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				SyNICTimer: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(*(vmiSpec.Domain.Features.Hyperv.VPIndex.Enabled)).To(BeTrue())
		Expect(*(vmiSpec.Domain.Features.Hyperv.SyNIC.Enabled)).To(BeTrue())
		Expect(*(vmiSpec.Domain.Features.Hyperv.SyNICTimer.Enabled)).To(BeTrue())
	})

	It("Should not mutate VMIs without HyperV configuration", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		Expect(vmi.Spec.Domain.Features).To(BeNil())
		err := webhooks.SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		Expect(err).To(BeNil())
		Expect(vmi.Spec.Domain.Features).To(BeNil())
	})

	It("Should not mutate VMIs with empty HyperV configuration", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{},
		}
		err := webhooks.SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		Expect(err).To(BeNil())
		hyperv := v1.FeatureHyperv{}
		ok := reflect.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should not mutate VMIs with hyperv configuration without deps", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				Runtime: &v1.FeatureState{
					Enabled: &_true,
				},
				Reset: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		err := webhooks.SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		Expect(err).To(BeNil())

		hyperv := v1.FeatureHyperv{
			Relaxed: &v1.FeatureState{
				Enabled: &_true,
			},
			Runtime: &v1.FeatureState{
				Enabled: &_true,
			},
			Reset: &v1.FeatureState{
				Enabled: &_true,
			},
		}

		ok := reflect.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should mutate VMIs with hyperv configuration to fix deps", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{
					Enabled: &_true,
				},
				SyNICTimer: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		err := webhooks.SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		Expect(err).To(BeNil())

		hyperv := v1.FeatureHyperv{
			Relaxed: &v1.FeatureState{
				Enabled: &_true,
			},
			VPIndex: &v1.FeatureState{
				Enabled: &_true,
			},
			SyNIC: &v1.FeatureState{
				Enabled: &_true,
			},
			SyNICTimer: &v1.FeatureState{
				Enabled: &_true,
			},
		}

		ok := reflect.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})

	It("Should partially mutate VMIs with explicit hyperv configuration", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		vmi.Spec.Domain.Features = &v1.Features{
			Hyperv: &v1.FeatureHyperv{
				VPIndex: &v1.FeatureState{
					Enabled: &_false,
				},
				// should enable SyNIC
				SyNICTimer: &v1.FeatureState{
					Enabled: &_true,
				},
			},
		}
		webhooks.SetVirtualMachineInstanceHypervFeatureDependencies(vmi)
		// we MUST report the error in mutation, but production code is
		// supposed to ignore it to fullfill the design semantics, see
		// the discussion in https://github.com/kubevirt/kubevirt/pull/2408

		hyperv := v1.FeatureHyperv{
			VPIndex: &v1.FeatureState{
				Enabled: &_false,
			},
			SyNIC: &v1.FeatureState{
				Enabled: &_true,
			},
			SyNICTimer: &v1.FeatureState{
				Enabled: &_true,
			},
		}

		ok := reflect.DeepEqual(*vmi.Spec.Domain.Features.Hyperv, hyperv)
		if !ok {
			// debug aid
			fmt.Fprintf(GinkgoWriter, "got: %#v\n", *vmi.Spec.Domain.Features.Hyperv)
			fmt.Fprintf(GinkgoWriter, "exp: %#v\n", hyperv)
		}
		Expect(ok).To(BeTrue())
	})
	table.DescribeTable("modify the VMI status", func(user string, shouldChange bool) {
		oldVMI := &v1.VirtualMachineInstance{}
		oldVMI.Status = v1.VirtualMachineInstanceStatus{
			Phase: v1.Running,
		}
		newVMI := oldVMI.DeepCopy()
		newVMI.Status = v1.VirtualMachineInstanceStatus{
			Phase: v1.Failed,
		}
		status := getVMIStatusFromResponse(oldVMI, newVMI, user)
		if shouldChange {
			Expect(&newVMI.Status).To(Equal(status))
		} else {
			Expect(&oldVMI.Status).To(Equal(status))
		}
	},
		table.Entry("if our service accounts modfies it", privilegedUser, true),
		table.Entry("not if the user is not one of ours", "unknown", false),
	)
})
