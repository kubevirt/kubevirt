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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("VirtualMachineInstance Mutator", func() {
	var vmi *v1.VirtualMachineInstance
	var preset *v1.VirtualMachineInstancePreset
	var presetInformer cache.SharedIndexInformer
	var namespaceLimit *k8sv1.LimitRange
	var namespaceLimitInformer cache.SharedIndexInformer
	var configMapInformer cache.SharedIndexInformer
	var mutator *VMIsMutator

	memoryLimit := "128M"
	cpuModelFromConfig := "Haswell"
	machineTypeFromConfig := "pc-q35-3.0"
	cpuRequestFromConfig := "800m"
	memoryRequestFromConfig := "256Mi"

	getVMISpecMetaFromResponse := func() (*v1.VirtualMachineInstanceSpec, *k8smetav1.ObjectMeta) {
		vmiBytes, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())
		By("Creating the test admissions review from the VMI")
		ar := &v1beta1.AdmissionReview{
			Request: &v1beta1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{Group: v1.VirtualMachineInstanceGroupVersionKind.Group, Version: v1.VirtualMachineInstanceGroupVersionKind.Version, Resource: "virtualmachineinstances"},
				Object: runtime.RawExtension{
					Raw: vmiBytes,
				},
			},
		}
		By("Mutating the VMI")
		resp := mutator.Mutate(ar)
		Expect(resp.Allowed).To(Equal(true))

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
		mutator.ClusterConfig, configMapInformer = testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})
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
		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.Machine.Type).To(Equal("q35"))
		Expect(vmiSpec.Domain.CPU.Model).To(Equal(""))
		Expect(vmiSpec.Domain.Resources.Requests.Cpu().String()).To(Equal("100m"))
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal("8Mi"))
	})

	It("should apply configurable defaults on VMI create", func() {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{
				virtconfig.CpuModelKey:      cpuModelFromConfig,
				virtconfig.MachineTypeKey:   machineTypeFromConfig,
				virtconfig.MemoryRequestKey: memoryRequestFromConfig,
				virtconfig.CpuRequestKey:    cpuRequestFromConfig,
			},
		})

		vmiSpec, _ := getVMISpecMetaFromResponse()
		Expect(vmiSpec.Domain.CPU.Model).To(Equal(cpuModelFromConfig))
		Expect(vmiSpec.Domain.Machine.Type).To(Equal(machineTypeFromConfig))
		Expect(vmiSpec.Domain.Resources.Requests.Cpu().String()).To(Equal(cpuRequestFromConfig))
		Expect(vmiSpec.Domain.Resources.Requests.Memory().String()).To(Equal(memoryRequestFromConfig))
	})

	It("should not override specified properties with defaults on VMI create", func() {
		testutils.UpdateFakeClusterConfig(configMapInformer, &k8sv1.ConfigMap{
			Data: map[string]string{
				virtconfig.CpuModelKey:      cpuModelFromConfig,
				virtconfig.MachineTypeKey:   machineTypeFromConfig,
				virtconfig.MemoryRequestKey: memoryRequestFromConfig,
				virtconfig.CpuRequestKey:    cpuRequestFromConfig,
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

	It("should apply foreground finalizer on VMI create", func() {
		_, vmiMeta := getVMISpecMetaFromResponse()
		Expect(vmiMeta.Finalizers).To(ContainElement(v1.VirtualMachineInstanceFinalizer))
	})
})
