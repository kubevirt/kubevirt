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

package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	draVfioClaimTemplateName          = "vfio-gpu-claim-template"
	draVfioRequestName                = "vfio-gpu"
	draVfioResourceClaimName          = "vfio-gpu-claim"
	draVfioVMIName                    = "dra-vfio-test"
	draVfioHostDeviceName             = "gpu0"
	draVfioDeviceClass                = "vfio-gpu.example.com"
	draVfioPCIBusID                   = "faca:00:05.0"
	draVfioUnmatchablePCIBusID        = "cccc:cc:cc.c"
	draVfioInvalidCELExpression       = `device.attributes["resource.kubernetes.io"].pciBusID.string == "faca:00:00.0"`
	draVfioVMICount                   = 8
	draVfioMultiDeviceRequestCount    = 3
	draVfioMatchAttributeRequestCount = 2
	draVfioVendorIDMatchAttribute     = "vfio-gpu.example.com/vendorID"
	draVfioPCIBusIDMatchAttribute     = "resource.kubernetes.io/pciBusID"
	timeout                           = 2 * time.Minute
	draCleanupTimeout                 = 90 * time.Second
	pollingInterval                   = 5 * time.Second
)

var _ = Describe("[sig-compute]DRA", Serial, decorators.SigCompute, decorators.DRA, func() {
	var (
		virtClient kubecli.KubevirtClient
		namespace  string
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		namespace = testsuite.GetTestNamespace(nil)
	})

	AfterEach(func() {
		cleanupDRATestResources(virtClient, namespace)
	})

	Context("create eight VMIs backed by the ResourceClaimTemplate", func() {
		It("should create eight VMIs backed by the ResourceClaimTemplate", func() {
			By("creating the shared ResourceClaimTemplate")
			rct := vfioGPUResourceClaimTemplate(namespace)
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(
				context.Background(), rct, metav1.CreateOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			By("creating eight VMIs backed by the ResourceClaimTemplate")
			for i := range draVfioVMICount {
				By(fmt.Sprintf("Creating VMI dra-vfio-test-%d", i))
				vmi := vfioGPUVMI(fmt.Sprintf("dra-vfio-test-%d", i),
					WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
					WithVfioGPUHostDevice(),
				)
				_, err := virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			By("Waiting for all VMIs to reach Running")
			waitForVMIsToBeRunning(virtClient, namespace, draVfioVMICount)

			By("Fetch all the ResourceClaims and check that they are all bound")
			expectBoundResourceClaimsForVMIs(virtClient, namespace, draVfioVMICount, draVfioTestVMINames(draVfioVMICount)...)
		})
	})

	Context("create 8 VMIs and pre-created ResourceClaims individually", func() {
		It("creating 8 VMIs and pre-created ResourceClaims individually", func() {
			for i := range draVfioVMICount {
				resourceClaim := vfioGPUResourceClaim(namespace, fmt.Sprintf("dra-vfio-test-%d", i))
				_, err := virtClient.ResourceV1().ResourceClaims(namespace).Create(context.Background(), resourceClaim, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				claimName := fmt.Sprintf("dra-vfio-test-%d", i)
				vmi := vfioGPUVMI(claimName,
					WithVfioGPUResourceClaim(claimName),
					WithVfioGPUHostDevice(WithVfioHostDeviceClaimRequest(claimName, draVfioRequestName)),
				)
				_, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			By("Waiting for all VMIs to reach Running")
			waitForVMIsToBeRunning(virtClient, namespace, draVfioVMICount)

			By("Fetch all the ResourceClaims and check that they are all bound")
			expectBoundResourceClaimsForVMIs(virtClient, namespace, draVfioVMICount, draVfioTestVMINames(draVfioVMICount)...)
		})
	})

	Context("create a VMI with a strict CEL based ResourceClaimTemplate", func() {
		It("should allocate a vfio-gpu device matching the index", func() {
			By("creating the ResourceClaimTemplate with a strict CEL based selector")
			resourceClaimTemplate := vfioGPUResourceClaimTemplate(namespace,
				WithVfioGPUSelectors(fmt.Sprintf(`device.attributes["vfio-gpu.example.com"].index == 0`)),
			)
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(context.Background(), resourceClaimTemplate, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating the VMI")
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUHostDevice(),
			)
			_, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VMI to reach Running")
			waitForVMIsToBeRunning(virtClient, namespace, 1)

			By("Fetch the ResourceClaim and check that it is bound")
			expectBoundResourceClaimsForVMIs(virtClient, namespace, 1, draVfioVMIName)

			time.Sleep(30 * time.Second)
		})

		It("should allocate a vfio-gpu device matching the CEL selector", func() {
			By("creating the ResourceClaimTemplate with a CEL based selector")
			celExpression := fmt.Sprintf(
				`device.attributes["resource.kubernetes.io"].pciBusID == %q`,
				draVfioPCIBusID,
			)
			resourceClaimTemplate := vfioGPUResourceClaimTemplate(namespace,
				WithVfioGPUSelectors(celExpression),
			)
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(context.Background(), resourceClaimTemplate, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating the VMI")
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUHostDevice(),
			)
			_, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VMI to reach Running")
			waitForVMIsToBeRunning(virtClient, namespace, 1)

			By("Fetch the ResourceClaim and check that it is bound")
			expectBoundResourceClaimsForVMIs(virtClient, namespace, 1, draVfioVMIName)

			time.Sleep(30 * time.Second)
		})
	})

	Context("create a VMI with a matchattribute based ResourceClaimTemplate", func() {
		It("should allocate vfio-gpu devices sharing vendorID via matchAttribute", func() {
			By("creating the ResourceClaimTemplate with matchAttribute on vendorID across two requests")
			resourceClaimTemplate := vfioGPUResourceClaimTemplate(namespace,
				WithVfioGPUMultipleRequests(draVfioMatchAttributeRequestCount),
				WithVfioGPURequestMatchAttribute(
					draVfioVendorIDMatchAttribute,
					vfioGPUIndexedRequestName(0),
					vfioGPUIndexedRequestName(1),
				),
			)
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(context.Background(), resourceClaimTemplate, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating the VMI with two host devices backed by the matched requests")
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUMultipleHostDevices(draVfioMatchAttributeRequestCount),
			)
			_, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VMI to reach Running")
			waitForVMIsToBeRunning(virtClient, namespace, 1)

			By("Fetch the ResourceClaim and check that two devices were allocated")
			expectBoundResourceClaimsForVMIs(virtClient, namespace, 1, draVfioVMIName)
			expectAllocatedDeviceCountForVMI(virtClient, namespace, draVfioVMIName, draVfioMatchAttributeRequestCount)

			time.Sleep(30 * time.Second)
		})
	})

	Context("create a VMI with multiple device requests in one ResourceClaim", func() {
		It("should allocate three vfio-gpu devices via separate requests and reach Running", func() {
			By("creating the ResourceClaimTemplate with multiple device requests")
			resourceClaimTemplate := vfioGPUResourceClaimTemplate(namespace,
				WithVfioGPUMultipleRequests(draVfioMultiDeviceRequestCount),
			)
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(context.Background(), resourceClaimTemplate, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating the VMI with multiple device requests")
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUMultipleHostDevices(draVfioMultiDeviceRequestCount),
			)
			_, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VMI to reach Running")
			waitForVMIsToBeRunning(virtClient, namespace, 1)

			By("Fetch the ResourceClaim and check that three devices were allocated")
			expectBoundResourceClaimsForVMIs(virtClient, namespace, 1, draVfioVMIName)
			expectAllocatedDeviceCountForVMI(virtClient, namespace, draVfioVMIName, draVfioMultiDeviceRequestCount)

			time.Sleep(30 * time.Second)
		})
	})
})

var _ = Describe("[sig-compute]DRA failing scenarios", Serial, decorators.SigCompute, func() {
	var (
		virtClient kubecli.KubevirtClient
		namespace  string
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		namespace = testsuite.GetTestNamespace(nil)
	})

	AfterEach(func() {
		cleanupDRATestResources(virtClient, namespace)
	})

	Context("failing scenarios", func() {
		It("should reject a VMI when the host device references an unknown resourceClaim", func() {
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUHostDevice(WithVfioHostDeviceClaimRequest("unknown-claim", draVfioRequestName)),
			)

			_, err := virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resourceClaims must specify all claims"))
		})

		It("should fail when a ResourceClaimTemplate requests more than one device per request", func() {
			By("creating the ResourceClaimTemplate")
			resourceClaimTemplate := vfioGPUResourceClaimTemplate(namespace, WithVfioGPUDeviceCount(3))
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(context.Background(), resourceClaimTemplate, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating the VMI")
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUHostDevice(),
			)
			vmi, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the VMI to remain in Scheduled phase as the KubeVirt only supports exactly one device per request (count > 1 is not supported)")
			time.Sleep(20 * time.Second)
			Consistently(matcher.ThisVMI(vmi), 30*time.Second, pollingInterval).Should(matcher.BeInPhase(v1.Scheduled))
		})

		It("should remain unschedulable when the CEL selector uses invalid attribute access", func() {
			By("creating the ResourceClaimTemplate")
			resourceClaimTemplate := vfioGPUResourceClaimTemplate(namespace,
				WithVfioGPUSelectors(draVfioInvalidCELExpression),
			)
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(context.Background(), resourceClaimTemplate, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating the VMI")
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUHostDevice(),
			)
			vmi, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the VMI to be unschedulable")
			time.Sleep(10 * time.Second)
			Consistently(matcher.ThisVMI(vmi), 30*time.Second, pollingInterval).Should(matcher.BeInPhase(v1.Scheduling))
			verifyVMIPodUnschedulable(virtClient, namespace, vmi.Name, ContainSubstring("no such key: string"))

		})

		It("should remain unschedulable when the CEL selector matches no device", func() {
			By("creating the ResourceClaimTemplate")
			celExpression := fmt.Sprintf(
				`device.attributes["resource.kubernetes.io"].pciBusID == %q`,
				draVfioUnmatchablePCIBusID,
			)
			resourceClaimTemplate := vfioGPUResourceClaimTemplate(namespace,
				WithVfioGPUSelectors(celExpression),
			)
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(context.Background(), resourceClaimTemplate, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating the VMI")
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUHostDevice(),
			)
			vmi, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(10 * time.Second)

			By("waiting for the VMI to be unschedulable")
			Consistently(matcher.ThisVMI(vmi), 30*time.Second, pollingInterval).Should(matcher.BeInPhase(v1.Scheduling))
			verifyVMIPodUnschedulable(virtClient, namespace, vmi.Name, ContainSubstring("cannot allocate all claims"))
		})
	})

	Context("create a VMI with a matchattribute based ResourceClaimTemplate", func() {
		It("should fail when the matchattribute has different values", func() {
			By("creating the ResourceClaimTemplate with matchAttribute on pciBusID across two requests")
			resourceClaimTemplate := vfioGPUResourceClaimTemplate(namespace,
				WithVfioGPUMultipleRequests(draVfioMatchAttributeRequestCount),
				WithVfioGPURequestMatchAttribute(
					draVfioPCIBusIDMatchAttribute,
					vfioGPUIndexedRequestName(0),
					vfioGPUIndexedRequestName(1),
				),
			)
			_, err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Create(context.Background(), resourceClaimTemplate, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("creating the VMI with two host devices backed by the matched requests")
			vmi := vfioGPUVMI(draVfioVMIName,
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUMultipleHostDevices(draVfioMatchAttributeRequestCount),
			)
			_, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			time.Sleep(10 * time.Second)

			By("VMI should remain in Scheduling phase as the matchattribute has different values")
			Consistently(matcher.ThisVMI(vmi), 30*time.Second, pollingInterval).Should(matcher.BeInPhase(v1.Scheduling))
			verifyVMIPodUnschedulable(virtClient, namespace, vmi.Name, ContainSubstring("cannot allocate all claims"))
		})
	})
})

type vfioGPUResourceClaimTemplateOption func(*resourcev1.ResourceClaimTemplate)

func vfioGPUResourceClaimTemplate(namespace string, opts ...vfioGPUResourceClaimTemplateOption) *resourcev1.ResourceClaimTemplate {
	rct := &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      draVfioClaimTemplateName,
			Namespace: namespace,
		},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: resourcev1.ResourceClaimSpec{
				Devices: resourcev1.DeviceClaim{
					Requests: []resourcev1.DeviceRequest{{
						Name: draVfioRequestName,
						Exactly: &resourcev1.ExactDeviceRequest{
							DeviceClassName: draVfioDeviceClass,
						},
					}},
				},
			},
		},
	}
	for _, opt := range opts {
		opt(rct)
	}
	return rct
}

func WithVfioGPUSelectors(expression string) vfioGPUResourceClaimTemplateOption {
	return func(rct *resourcev1.ResourceClaimTemplate) {
		rct.Spec.Spec.Devices.Requests[0].Exactly.Selectors = append(
			rct.Spec.Spec.Devices.Requests[0].Exactly.Selectors,
			resourcev1.DeviceSelector{
				CEL: &resourcev1.CELDeviceSelector{Expression: expression},
			},
		)
	}
}

func WithVfioGPURequestMatchAttribute(attribute string, requestNames ...string) vfioGPUResourceClaimTemplateOption {
	return func(rct *resourcev1.ResourceClaimTemplate) {
		fqName := resourcev1.FullyQualifiedName(attribute)
		constraint := resourcev1.DeviceConstraint{
			MatchAttribute: &fqName,
		}
		if len(requestNames) > 0 {
			constraint.Requests = requestNames
		}
		rct.Spec.Spec.Devices.Constraints = append(
			rct.Spec.Spec.Devices.Constraints,
			constraint,
		)
	}
}

func WithVfioGPUDeviceCount(count int64) vfioGPUResourceClaimTemplateOption {
	return func(rct *resourcev1.ResourceClaimTemplate) {
		rct.Spec.Spec.Devices.Requests[0].Exactly.Count = count
	}
}

func WithVfioGPUMultipleRequests(count int) vfioGPUResourceClaimTemplateOption {
	return func(rct *resourcev1.ResourceClaimTemplate) {
		requests := make([]resourcev1.DeviceRequest, count)
		for i := range count {
			requests[i] = resourcev1.DeviceRequest{
				Name: vfioGPUIndexedRequestName(i),
				Exactly: &resourcev1.ExactDeviceRequest{
					DeviceClassName: draVfioDeviceClass,
				},
			}
		}
		rct.Spec.Spec.Devices.Requests = requests
	}
}

func vfioGPUIndexedRequestName(index int) string {
	return fmt.Sprintf("%s-%d", draVfioRequestName, index)
}

func vfioGPUIndexedHostDeviceName(index int) string {
	return fmt.Sprintf("gpu%d", index)
}

type vfioGPUVMIOption func(*v1.VirtualMachineInstance)

func vfioGPUVMI(name string, opts ...vfioGPUVMIOption) *v1.VirtualMachineInstance {
	vmi := libvmifact.NewCirros(
		libnet.WithMasqueradeNetworking(),
		libvmi.WithAutoattachGraphicsDevice(false),
	)
	vmi.Name = name
	for _, opt := range opts {
		opt(vmi)
	}
	return vmi
}

func WithVfioGPUResourceClaimTemplate(templateName string) vfioGPUVMIOption {
	return func(vmi *v1.VirtualMachineInstance) {
		libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{
			Name:                      draVfioResourceClaimName,
			ResourceClaimTemplateName: &templateName,
		})(vmi)
	}
}

func WithVfioGPUResourceClaim(claimName string) vfioGPUVMIOption {
	return func(vmi *v1.VirtualMachineInstance) {
		libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{
			Name:              claimName,
			ResourceClaimName: &claimName,
		})(vmi)
	}
}

func WithVfioGPUHostDevice(opts ...vfioGPUHostDeviceOption) vfioGPUVMIOption {
	return func(vmi *v1.VirtualMachineInstance) {
		libvmi.WithHostDevice(vfioGPUHostDevice(opts...))(vmi)
	}
}

func WithVfioGPUMultipleHostDevices(count int) vfioGPUVMIOption {
	return func(vmi *v1.VirtualMachineInstance) {
		for i := range count {
			libvmi.WithHostDevice(vfioGPUHostDevice(
				WithVfioHostDeviceName(vfioGPUIndexedHostDeviceName(i)),
				WithVfioHostDeviceClaimRequest(draVfioResourceClaimName, vfioGPUIndexedRequestName(i)),
			))(vmi)
		}
	}
}

type vfioGPUHostDeviceOption func(*v1.HostDevice)

func vfioGPUHostDevice(opts ...vfioGPUHostDeviceOption) v1.HostDevice {
	hostDevice := v1.HostDevice{
		Name: draVfioHostDeviceName,
		ClaimRequest: &v1.ClaimRequest{
			ClaimName:   draVfioResourceClaimName,
			RequestName: draVfioRequestName,
		},
	}
	for _, opt := range opts {
		opt(&hostDevice)
	}
	return hostDevice
}

func WithVfioHostDeviceName(name string) vfioGPUHostDeviceOption {
	return func(hostDevice *v1.HostDevice) {
		hostDevice.Name = name
	}
}

func WithVfioHostDeviceClaimRequest(claimName, requestName string) vfioGPUHostDeviceOption {
	return func(hostDevice *v1.HostDevice) {
		hostDevice.ClaimRequest = &v1.ClaimRequest{
			ClaimName:   claimName,
			RequestName: requestName,
		}
	}
}

func verifyVMIPodUnschedulable(virtClient kubecli.KubevirtClient, namespace, vmiName string, messageMatcher OmegaMatcher) {
	vmi, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vmiName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	cond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(
		vmi, v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
	)
	Expect(cond).NotTo(BeNil())
	Expect(cond.Status).To(Equal(k8sv1.ConditionFalse))
	Expect(cond.Reason).To(SatisfyAny(
		Equal(k8sv1.PodReasonUnschedulable),
		Equal(k8sv1.PodReasonSchedulerError),
	))
	Expect(cond.Message).To(messageMatcher)
}

func waitForVMIsToBeRunning(virtClient kubecli.KubevirtClient, namespace string, vmiCount int) {
	Eventually(func() int {
		vmiList, err := virtClient.VirtualMachineInstance(namespace).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		running := 0
		for _, vmi := range vmiList.Items {
			if vmi.Status.Phase == v1.Running {
				running++
			}
		}
		return running
	}, timeout, pollingInterval).Should(BeNumerically(">=", vmiCount))
}

func draVfioTestVMINames(count int) []string {
	names := make([]string, count)
	for i := range count {
		names[i] = fmt.Sprintf("dra-vfio-test-%d", i)
	}
	return names
}

func resourceClaimBelongsToVMI(claimName, vmiName string) bool {
	return claimName == vmiName || strings.Contains(claimName, fmt.Sprintf("virt-launcher-%s-", vmiName))
}

func listResourceClaimsForVMIs(virtClient kubecli.KubevirtClient, namespace string, vmiNames ...string) ([]resourcev1.ResourceClaim, error) {
	resourceClaimList, err := virtClient.ResourceV1().ResourceClaims(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var claims []resourcev1.ResourceClaim
	for _, resourceClaim := range resourceClaimList.Items {
		for _, vmiName := range vmiNames {
			if resourceClaimBelongsToVMI(resourceClaim.Name, vmiName) {
				claims = append(claims, resourceClaim)
				break
			}
		}
	}
	return claims, nil
}

func expectAllocatedDeviceCountForVMI(virtClient kubecli.KubevirtClient, namespace, vmiName string, expectedDeviceCount int) {
	claims, err := listResourceClaimsForVMIs(virtClient, namespace, vmiName)
	Expect(err).ToNot(HaveOccurred())
	Expect(claims).To(HaveLen(1))
	Expect(claims[0].Status.Allocation).NotTo(BeNil())
	Expect(claims[0].Status.Allocation.Devices.Results).To(HaveLen(expectedDeviceCount))
	Expect(claims[0].Status.ReservedFor).NotTo(BeEmpty())
	Expect(claims[0].Status.ReservedFor[0].Resource).To(Equal("pods"))
	Expect(claims[0].Status.ReservedFor[0].Name).To(ContainSubstring(fmt.Sprintf("virt-launcher-%s-", vmiName)))
}

func expectBoundResourceClaimsForVMIs(virtClient kubecli.KubevirtClient, namespace string, expectedCount int, vmiNames ...string) {
	Eventually(func(g Gomega) {
		claims, err := listResourceClaimsForVMIs(virtClient, namespace, vmiNames...)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(claims).To(HaveLen(expectedCount))
		for _, resourceClaim := range claims {
			var matchedVMIName string
			for _, vmiName := range vmiNames {
				if resourceClaimBelongsToVMI(resourceClaim.Name, vmiName) {
					matchedVMIName = vmiName
					break
				}
			}
			g.Expect(matchedVMIName).NotTo(BeEmpty(), "claim %s should belong to one of the VMIs", resourceClaim.Name)

			g.Expect(resourceClaim.Status).NotTo(BeNil())
			g.Expect(resourceClaim.Status.Allocation).NotTo(BeNil())
			g.Expect(resourceClaim.Status.ReservedFor).NotTo(BeEmpty())
			g.Expect(resourceClaim.Status.ReservedFor[0].Resource).To(Equal("pods"))
			g.Expect(resourceClaim.Status.ReservedFor[0].Name).To(ContainSubstring(fmt.Sprintf("virt-launcher-%s-", matchedVMIName)))
		}
	}, timeout, pollingInterval).Should(Succeed())
}

func cleanupDRATestResources(virtClient kubecli.KubevirtClient, namespace string) {
	_ = virtClient.VirtualMachineInstance(namespace).DeleteCollection(
		context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{},
	)
	_ = virtClient.ResourceV1().ResourceClaims(namespace).DeleteCollection(
		context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{},
	)
	deleteVfioGPUResourceClaimTemplateIfExists(virtClient, namespace)
	Eventually(func(g Gomega) {
		vmiList, err := virtClient.VirtualMachineInstance(namespace).List(context.Background(), metav1.ListOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(vmiList.Items).To(BeEmpty())

		resourceClaimList, err := virtClient.ResourceV1().ResourceClaims(namespace).List(context.Background(), metav1.ListOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(resourceClaimList.Items).To(BeEmpty())
	}, draCleanupTimeout, pollingInterval).Should(Succeed())
}

func deleteVfioGPUResourceClaimTemplateIfExists(virtClient kubecli.KubevirtClient, namespace string) {
	err := virtClient.ResourceV1().ResourceClaimTemplates(namespace).Delete(
		context.Background(), draVfioClaimTemplateName, metav1.DeleteOptions{},
	)
	if err != nil && !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}

func vfioGPUResourceClaim(namespace, name string) *resourcev1.ResourceClaim {
	return &resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{{
					Name: draVfioRequestName,
					Exactly: &resourcev1.ExactDeviceRequest{
						DeviceClassName: draVfioDeviceClass,
					},
				}},
			},
		},
	}
}
