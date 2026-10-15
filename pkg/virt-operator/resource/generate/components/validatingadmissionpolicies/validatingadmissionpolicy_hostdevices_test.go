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

package validatingadmissionpolicies_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
)

const (
	controllerUser = "system:serviceaccount:kubevirt:kubevirt-controller"
	otherUser      = "system:serviceaccount:tenant:default"
	usbResource    = "devices.kubevirt.io/usb"
	gpuResource    = "nvidia.com/TU104GL_Tesla_T4"
	mdevResource   = "nvidia.com/GRID_T4-1Q"
	kvmResource    = "devices.kubevirt.io/kvm"
)

func kubeVirtParams(cfg v1.KubeVirtConfiguration) map[string]interface{} {
	data, err := json.Marshal(&v1.KubeVirt{
		Spec: v1.KubeVirtSpec{Configuration: cfg},
	})
	Expect(err).NotTo(HaveOccurred())
	var obj map[string]interface{}
	Expect(json.Unmarshal(data, &obj)).To(Succeed())
	return obj
}

func podWithResources(requests, limits corev1.ResourceList) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "workload"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "compute",
				Image: "registry.k8s.io/pause:3.10",
				Resources: corev1.ResourceRequirements{
					Requests: requests,
					Limits:   limits,
				},
			}},
		},
	}
}

func resourceList(name, qty string) corev1.ResourceList {
	return corev1.ResourceList{corev1.ResourceName(name): resource.MustParse(qty)}
}

func admitHostDevice(pod *corev1.Pod, cfg v1.KubeVirtConfiguration, username string) error {
	policy := vap.NewHostDevicesValidatingAdmissionPolicy(controllerUser)
	in := admissionInput{
		object: podToUnstructured(pod),
		params: kubeVirtParams(cfg),
		request: map[string]interface{}{
			"operation": "CREATE",
			"userInfo": map[string]interface{}{
				"username": username,
			},
		},
	}
	if !policyApplies(policy, in) {
		return nil
	}
	return evaluatePolicyInput(policy, in)
}

var permittedUSB = v1.KubeVirtConfiguration{
	PermittedHostDevices: &v1.PermittedHostDevices{
		USB: []v1.USBHostDevice{{
			ResourceName: usbResource,
		}},
		PciHostDevices: []v1.PciHostDevice{{
			PCIVendorSelector: "10de:1eb8",
			ResourceName:      gpuResource,
		}},
		MediatedDevices: []v1.MediatedHostDevice{{
			MDEVNameSelector: "GRID T4-1Q",
			ResourceName:     mdevResource,
		}},
	},
}

var _ = Describe("Host device ValidatingAdmissionPolicy", func() {
	It("should target virt-controller and the KubeVirt CR in the install namespace", func() {
		policy := vap.NewHostDevicesValidatingAdmissionPolicy(controllerUser)
		binding := vap.NewHostDevicesValidatingAdmissionPolicyBinding("kubevirt")

		Expect(policy.Name).To(Equal(vap.HostDevicesRestrictionPolicyName))
		Expect(policy.Spec.ParamKind.Kind).To(Equal("KubeVirt"))
		Expect(policy.Spec.ParamKind.APIVersion).To(Equal(v1.SchemeGroupVersion.String()))
		Expect(policy.Spec.FailurePolicy).NotTo(BeNil())
		Expect(*policy.Spec.FailurePolicy).To(Equal(admissionregistrationv1.Fail))
		Expect(policy.Spec.Validations).To(HaveLen(1))
		Expect(policy.Spec.Validations[0].Message).To(Equal(vap.HostDevicesRestrictionErr))
		Expect(policy.Spec.Validations[0].Message).NotTo(ContainSubstring("\n"))
		Expect(policy.Spec.MatchConstraints.ResourceRules[0].Resources).To(Equal([]string{"pods"}))

		var joined string
		for _, variable := range policy.Spec.Variables {
			joined += variable.Expression
		}
		Expect(joined).To(ContainSubstring(controllerUser))

		Expect(binding.Name).To(Equal(vap.HostDevicesRestrictionBindingName))
		Expect(binding.Spec.PolicyName).To(Equal(policy.Name))
		Expect(binding.Spec.ParamRef.Namespace).To(Equal("kubevirt"))
		Expect(binding.Spec.ParamRef.Name).To(BeEmpty())
		Expect(binding.Spec.ParamRef.Selector).NotTo(BeNil())
		Expect(binding.Spec.ParamRef.ParameterNotFoundAction).NotTo(BeNil())
		Expect(*binding.Spec.ParamRef.ParameterNotFoundAction).To(Equal(admissionregistrationv1.AllowAction))
		Expect(binding.Spec.ValidationActions).To(ConsistOf(admissionregistrationv1.Deny))
	})

	DescribeTable("should allow pods that do not request a KubeVirt-managed host device",
		func(pod *corev1.Pod, cfg v1.KubeVirtConfiguration) {
			Expect(admitHostDevice(pod, cfg, otherUser)).To(Succeed())
		},
		Entry("cpu and memory only, even when host devices are permitted",
			podWithResources(corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			}, nil),
			permittedUSB,
		),
		Entry("hypervisor device plugin resources used by virt-launcher and virtctl guestfs",
			podWithResources(resourceList(kvmResource, "1"), nil),
			permittedUSB,
		),
		Entry("a host-device resource that is not declared on the KubeVirt CR",
			podWithResources(resourceList(usbResource, "1"), nil),
			v1.KubeVirtConfiguration{},
		),
		Entry("a device left to an external device plugin",
			podWithResources(resourceList(gpuResource, "1"), nil),
			v1.KubeVirtConfiguration{
				PermittedHostDevices: &v1.PermittedHostDevices{
					PciHostDevices: []v1.PciHostDevice{{
						PCIVendorSelector:        "10de:1eb8",
						ResourceName:             gpuResource,
						ExternalResourceProvider: true,
					}},
				},
			},
		),
	)

	DescribeTable("should deny unrelated pods that request a KubeVirt-managed host device",
		func(pod *corev1.Pod) {
			err := admitHostDevice(pod, permittedUSB, otherUser)
			Expect(err).To(MatchError(ContainSubstring(vap.HostDevicesRestrictionErr)))
		},
		Entry("usb in requests", podWithResources(resourceList(usbResource, "1"), nil)),
		Entry("pci device in requests", podWithResources(resourceList(gpuResource, "1"), nil)),
		Entry("mediated device in requests", podWithResources(resourceList(mdevResource, "1"), nil)),
		Entry("usb in limits only", podWithResources(nil, resourceList(usbResource, "1"))),
		Entry("usb in an init container", func() *corev1.Pod {
			pod := podWithResources(nil, nil)
			pod.Spec.InitContainers = []corev1.Container{{
				Name:      "init",
				Image:     "registry.k8s.io/pause:3.10",
				Resources: corev1.ResourceRequirements{Requests: resourceList(usbResource, "1")},
			}}
			return pod
		}()),
		Entry("usb in an ephemeral container", func() *corev1.Pod {
			pod := podWithResources(nil, nil)
			pod.Spec.EphemeralContainers = []corev1.EphemeralContainer{{
				EphemeralContainerCommon: corev1.EphemeralContainerCommon{
					Name:      "debug",
					Image:     "registry.k8s.io/pause:3.10",
					Resources: corev1.ResourceRequirements{Requests: resourceList(usbResource, "1")},
				},
			}}
			return pod
		}()),
	)

	It("should allow virt-controller to create a pod that requests the host device", func() {
		pod := podWithResources(corev1.ResourceList{
			corev1.ResourceName(kvmResource): resource.MustParse("1"),
			corev1.ResourceName(usbResource): resource.MustParse("1"),
		}, nil)
		Expect(admitHostDevice(pod, permittedUSB, controllerUser)).To(Succeed())
	})

	It("should match pod creation only, because extended resource requests are immutable", func() {
		policy := vap.NewHostDevicesValidatingAdmissionPolicy(controllerUser)
		binding := vap.NewHostDevicesValidatingAdmissionPolicyBinding("kubevirt")
		Expect(policy.Spec.MatchConstraints.ResourceRules[0].Operations).To(Equal([]admissionregistrationv1.OperationType{admissionregistrationv1.Create}))
		Expect(binding.Spec.MatchResources.ResourceRules[0].Operations).To(Equal([]admissionregistrationv1.OperationType{admissionregistrationv1.Create}))
	})

	It("should ignore an empty resource name and a device owned by an external plugin when another device is managed", func() {
		cfg := v1.KubeVirtConfiguration{
			PermittedHostDevices: &v1.PermittedHostDevices{
				USB: []v1.USBHostDevice{
					{ResourceName: ""},
					{ResourceName: usbResource, ExternalResourceProvider: true},
				},
				PciHostDevices: []v1.PciHostDevice{{
					PCIVendorSelector: "8086:6f54",
					ResourceName:      "intel.com/qat",
				}},
			},
		}
		Expect(admitHostDevice(podWithResources(resourceList(usbResource, "1"), nil), cfg, otherUser)).To(Succeed())
		Expect(admitHostDevice(podWithResources(resourceList("intel.com/qat", "1"), nil), cfg, otherUser)).NotTo(Succeed())
	})
})
