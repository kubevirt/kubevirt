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

package compute

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	draVfioClaimTemplateName = "vfio-gpu-claim-template"
	draVfioRequestName       = "vfio-gpu"
	draVfioResourceClaimName = "vfio-gpu-claim"
	timeout                  = 5 * time.Minute
	pollingInterval          = 5 * time.Second
)

var _ = Describe("[sig-compute]DRA", Serial, decorators.SigCompute, decorators.DRAGPU, func() {
	const (
		draVfioMatchAttributeRequestCount = 2
		draVfioVMICount                   = 4
	)

	It("should create four VMIs backed by the ResourceClaimTemplate", func() {
		By("creating the shared ResourceClaimTemplate")
		createVFIOGPUResourceClaimTemplate()

		By("creating four VMIs backed by the ResourceClaimTemplate")
		vmiNames := make([]string, 0, draVfioVMICount)
		for range draVfioVMICount {
			createdVMI := createVFIOGPUVMI(
				WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
				WithVfioGPUDevice(),
			)
			vmiNames = append(vmiNames, createdVMI.Name)
		}

		By("Waiting for all ResourceClaims to be created")
		waitForResourceClaimsToBeCreated(draVfioVMICount)

		By("Fetch all the ResourceClaims and check that they are all bound")
		waitForBoundResourceClaimsForVMIs(draVfioVMICount, vmiNames...)

		By("Waiting for all VMIs to reach Running")
		waitForVMIsToBeRunning(vmiNames...)
	})

	It("should create four VMIs and pre-created ResourceClaims individually", func() {
		By("creating four ResourceClaims and VMIs")
		vmiNames := make([]string, 0, draVfioVMICount)
		for range draVfioVMICount {
			createdClaim := createVFIOGPUResourceClaim()
			createdVMI := createVFIOGPUVMI(
				WithVfioGPUResourceClaim(createdClaim.Name),
				WithVfioGPUDevice(WithVfioGPUClaimRequest(createdClaim.Name, draVfioRequestName)),
			)
			vmiNames = append(vmiNames, createdVMI.Name)
		}

		By("Fetch all the ResourceClaims and check that they are all bound")
		waitForBoundResourceClaimsForVMIs(draVfioVMICount, vmiNames...)

		By("Waiting for all VMIs to reach Running")
		waitForVMIsToBeRunning(vmiNames...)
	})

	It("should allocate a vfio-gpu device matching the vendorID", func() {
		By("creating the ResourceClaimTemplate with a strict CEL based selector")
		createVFIOGPUResourceClaimTemplate(
			WithVfioGPUSelectors(vfioGPUVendorIDSelector("e1a5")),
		)

		By("creating the VMI")
		createdVMI := createVFIOGPUVMI(
			WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
			WithVfioGPUDevice(),
		)
		By("Waiting for the ResourceClaim to be created")
		waitForResourceClaimsToBeCreated(1)

		By("Fetch the ResourceClaim and check that it is bound")
		waitForBoundResourceClaimsForVMIs(1, createdVMI.Name)

		By("Waiting for the VMI to reach Running")
		waitForVMIToBeRunning(createdVMI)
	})

	It("should create a VMI with a pre-created ResourceClaim and allocate a vfio-gpu device matching the vendorID", func() {
		By("creating the ResourceClaim")
		createdClaim := createVFIOGPUResourceClaim()

		By("creating the VMI")
		createdVMI := createVFIOGPUVMI(
			WithVfioGPUResourceClaim(createdClaim.Name),
			WithVfioGPUDevice(WithVfioGPUClaimRequest(createdClaim.Name, draVfioRequestName)),
		)

		By("Fetch the ResourceClaim and check that it is bound")
		waitForBoundResourceClaimsForVMIs(1, createdVMI.Name)

		By("Waiting for the VMI to reach Running")
		waitForVMIToBeRunning(createdVMI)
	})

	It("should allocate a vfio-gpu device matching the standardized pciBusId attribute", func() {
		By("creating the ResourceClaimTemplate with a CEL based selector")
		createVFIOGPUResourceClaimTemplate(
			WithVfioGPUSelectors(vfioGPUPCIBusIDSelector("faca:00:05.0")),
		)

		By("creating the VMI")
		createdVMI := createVFIOGPUVMI(
			WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
			WithVfioGPUDevice(),
		)

		By("Waiting for the ResourceClaim to be created")
		waitForResourceClaimsToBeCreated(1)

		By("Fetch the ResourceClaim and check that it is bound")
		waitForBoundResourceClaimsForVMIs(1, createdVMI.Name)

		By("Waiting for the VMI to reach Running")
		waitForVMIToBeRunning(createdVMI)
	})

	// TODO: Add a test that is matching two devices from different drivers. However it requires a fake network device driver. Until then, this test simulates the match attribute scenario by matching two devices with the same vendorID.
	It("should allocate vfio-gpu devices sharing vendorID via matchAttribute", func() {
		By("creating the ResourceClaimTemplate with matchAttribute on vendorID across two requests")
		createVFIOGPUResourceClaimTemplate(
			WithVfioGPUMultipleRequests(draVfioMatchAttributeRequestCount),
			WithVfioGPURequestMatchAttribute(
				"vfio-gpu.example.com/vendorID",
				vfioGPUIndexedRequestName(0),
				vfioGPUIndexedRequestName(1),
			),
		)

		By("creating the VMI with two GPUs backed by the matched requests")
		createdVMI := createVFIOGPUVMI(
			WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
			WithVfioGPUMultipleGPUs(draVfioMatchAttributeRequestCount),
		)

		By("Waiting for the ResourceClaim to be created")
		waitForResourceClaimsToBeCreated(1)

		By("Fetch the ResourceClaim and check that two devices were allocated")
		waitForBoundResourceClaimsForVMIs(1, createdVMI.Name)
		expectAllocatedDeviceCountForVMI(createdVMI.Name, draVfioMatchAttributeRequestCount)

		By("Waiting for the VMI to reach Running")
		waitForVMIToBeRunning(createdVMI)
	})

	It("should allocate three vfio-gpu devices via separate requests and reach Running", func() {
		const multiDeviceRequestCount = 3
		const draVfioMultiDevicePCIOffset = 3
		By("creating the ResourceClaimTemplate with multiple device requests")
		createVFIOGPUResourceClaimTemplate(
			WithVfioGPUMultipleRequestsFromIndex(multiDeviceRequestCount, draVfioMultiDevicePCIOffset),
		)

		By("creating the VMI with multiple device requests")
		createdVMI := createVFIOGPUVMI(
			WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
			WithVfioGPUMultipleGPUs(multiDeviceRequestCount),
			withVfioGPUMultiDeviceMemory(),
		)

		By("Waiting for the ResourceClaim to be created")
		waitForResourceClaimsToBeCreated(1)

		By("Fetch the ResourceClaim and check that three devices were allocated")
		waitForBoundResourceClaimsForVMIs(1, createdVMI.Name)
		expectAllocatedDeviceCountForVMI(createdVMI.Name, multiDeviceRequestCount)

		By("Waiting for the VMI to reach Running")
		waitForVMIToBeRunning(createdVMI)
	})

	It("should fail when the matchattribute has different values", func() {
		By("creating the ResourceClaimTemplate with matchAttribute on pciBusID across two requests")
		createVFIOGPUResourceClaimTemplate(
			WithVfioGPUMultipleRequests(draVfioMatchAttributeRequestCount),
			WithVfioGPURequestMatchAttribute(
				"resource.kubernetes.io/pciBusID",
				vfioGPUIndexedRequestName(0),
				vfioGPUIndexedRequestName(1),
			),
		)

		By("creating the VMI with two GPUs backed by the matched requests")
		vmi := createVFIOGPUVMI(
			WithVfioGPUResourceClaimTemplate(draVfioClaimTemplateName),
			WithVfioGPUMultipleGPUs(draVfioMatchAttributeRequestCount),
		)

		Eventually(matcher.ThisVMI(vmi), 30*time.Second, pollingInterval).Should(matcher.BeInPhase(v1.Scheduling))
		Consistently(matcher.ThisVMI(vmi), 30*time.Second, pollingInterval).Should(matcher.BeInPhase(v1.Scheduling))
		verifyVMIPodUnschedulable(vmi.Name, ContainSubstring("cannot allocate all claims"))
	})
})

func createVFIOGPUResourceClaimTemplate(opts ...vfioGPUResourceClaimTemplateOption) {
	resourceClaimTemplate := vfioGPUResourceClaimTemplate(opts...)
	_, err := kubevirt.Client().ResourceV1().ResourceClaimTemplates(testsuite.GetTestNamespace(nil)).Create(
		context.Background(), resourceClaimTemplate, metav1.CreateOptions{},
	)
	Expect(err).ToNot(HaveOccurred())
}

type vfioGPUResourceClaimTemplateOption func(*resourcev1.ResourceClaimTemplate)

func vfioGPUResourceClaimTemplate(opts ...vfioGPUResourceClaimTemplateOption) *resourcev1.ResourceClaimTemplate {
	rct := &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      draVfioClaimTemplateName,
			Namespace: testsuite.GetTestNamespace(nil),
		},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: resourcev1.ResourceClaimSpec{
				Devices: resourcev1.DeviceClaim{
					Requests: []resourcev1.DeviceRequest{vfioGPUDeviceRequest(draVfioRequestName)},
				},
			},
		},
	}
	for _, opt := range opts {
		opt(rct)
	}
	return rct
}

func WithVfioGPUSelectors(expressions ...string) vfioGPUResourceClaimTemplateOption {
	return func(rct *resourcev1.ResourceClaimTemplate) {
		selectors := make([]resourcev1.DeviceSelector, len(expressions))
		for i, expression := range expressions {
			selectors[i] = resourcev1.DeviceSelector{
				CEL: &resourcev1.CELDeviceSelector{Expression: expression},
			}
		}
		rct.Spec.Spec.Devices.Requests[0].Exactly.Selectors = selectors
	}
}

func vfioGPUDeviceRequest(name string) resourcev1.DeviceRequest {
	return vfioGPUDeviceRequestWithSelector(name, `device.attributes["resource.kubernetes.io"].pcieRoot == "pcifaca:00"`)
}

func vfioGPUDeviceRequestWithSelector(name, celExpression string) resourcev1.DeviceRequest {
	return resourcev1.DeviceRequest{
		Name: name,
		Exactly: &resourcev1.ExactDeviceRequest{
			DeviceClassName: "vfio-gpu.example.com",
			Selectors: []resourcev1.DeviceSelector{{
				CEL: &resourcev1.CELDeviceSelector{Expression: celExpression},
			}},
		},
	}
}

func vfioGPUIndexedPCIBusID(index int) string {
	return fmt.Sprintf("faca:00:%02x.0", index)
}

func vfioGPUPCIBusIDSelector(pciBusID string) string {
	return fmt.Sprintf(`device.attributes["resource.kubernetes.io"].pciBusID == %q`, pciBusID)
}

func vfioGPUVendorIDSelector(vendorID string) string {
	return fmt.Sprintf(`device.attributes["vfio-gpu.example.com"].vendorID == %q`, vendorID)
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
	return WithVfioGPUMultipleRequestsFromIndex(count, 0)
}

func WithVfioGPUMultipleRequestsFromIndex(count, pciOffset int) vfioGPUResourceClaimTemplateOption {
	return func(rct *resourcev1.ResourceClaimTemplate) {
		requests := make([]resourcev1.DeviceRequest, count)
		for i := range count {
			requests[i] = vfioGPUDeviceRequestWithSelector(
				vfioGPUIndexedRequestName(i),
				vfioGPUPCIBusIDSelector(vfioGPUIndexedPCIBusID(pciOffset+i)),
			)
		}
		rct.Spec.Spec.Devices.Requests = requests
	}
}

func vfioGPUIndexedRequestName(index int) string {
	return fmt.Sprintf("%s-%d", draVfioRequestName, index)
}

func vfioGPUIndexedName(index int) string {
	return fmt.Sprintf("gpu%d", index)
}

type vfioGPUVMIOption func(*v1.VirtualMachineInstance)

func createVFIOGPUVMI(opts ...vfioGPUVMIOption) *v1.VirtualMachineInstance {
	vmi := libvmifact.NewAlpine(libvmi.WithMemoryRequest("32Mi"))
	for _, opt := range opts {
		opt(vmi)
	}

	createdVMI, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return createdVMI
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

func WithVfioGPUDevice(opts ...vfioGPUOption) vfioGPUVMIOption {
	return func(vmi *v1.VirtualMachineInstance) {
		libvmi.WithGPU(vfioGPU(opts...))(vmi)
	}
}

func WithVfioGPUMultipleGPUs(count int) vfioGPUVMIOption {
	return func(vmi *v1.VirtualMachineInstance) {
		for i := range count {
			libvmi.WithGPU(vfioGPU(
				WithVfioGPUName(vfioGPUIndexedName(i)),
				WithVfioGPUClaimRequest(draVfioResourceClaimName, vfioGPUIndexedRequestName(i)),
			))(vmi)
		}
	}
}

type vfioGPUOption func(*v1.GPU)

func vfioGPU(opts ...vfioGPUOption) v1.GPU {
	gpu := v1.GPU{
		Name: "gpu0",
		ClaimRequest: &v1.ClaimRequest{
			ClaimName:   draVfioResourceClaimName,
			RequestName: draVfioRequestName,
		},
	}
	for _, opt := range opts {
		opt(&gpu)
	}
	return gpu
}

func WithVfioGPUName(name string) vfioGPUOption {
	return func(gpu *v1.GPU) {
		gpu.Name = name
	}
}

func WithVfioGPUClaimRequest(claimName, requestName string) vfioGPUOption {
	return func(gpu *v1.GPU) {
		gpu.ClaimRequest = &v1.ClaimRequest{
			ClaimName:   claimName,
			RequestName: requestName,
		}
	}
}

func verifyVMIPodUnschedulable(vmiName string, messageMatcher OmegaMatcher) {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmiName, metav1.GetOptions{})
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

func expectVMISyncFailed(vmiName string, messageMatcher OmegaMatcher) {
	Eventually(func(g Gomega) {
		vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmiName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		cond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(
			vmi, v1.VirtualMachineInstanceSynchronized,
		)
		g.Expect(cond).NotTo(BeNil())
		g.Expect(cond.Status).To(Equal(k8sv1.ConditionFalse))
		g.Expect(cond.Reason).To(Equal("Synchronizing with the Domain failed."))
		g.Expect(cond.Message).To(messageMatcher)
	}, timeout, pollingInterval).Should(Succeed())
}

func withVfioGPUMultiDeviceMemory() vfioGPUVMIOption {
	return func(vmi *v1.VirtualMachineInstance) {
		libvmi.WithMemoryRequest("128Mi")(vmi)
	}
}

func waitForVMIToBeRunning(vmi *v1.VirtualMachineInstance) {
	Eventually(matcher.ThisVMI(vmi), timeout, pollingInterval).Should(matcher.BeRunning())
}

func waitForVMIsToBeRunning(vmiNames ...string) {
	Eventually(func(g Gomega) {
		for _, vmiName := range vmiNames {
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmiName, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vmi).To(matcher.BeRunning(), "VMI %s", vmiName)
		}
	}, timeout, pollingInterval).Should(Succeed())
}

func virtLauncherPodNamePrefixForVMI(vmiName string) string {
	const libvmiRandNamePrefix = "testvmi-"
	const libvmiRandNameRandomLen = 5
	if strings.HasPrefix(vmiName, libvmiRandNamePrefix) &&
		len(vmiName) >= len(libvmiRandNamePrefix)+libvmiRandNameRandomLen+1 {
		uniqueName := vmiName[:len(libvmiRandNamePrefix)+libvmiRandNameRandomLen+1]
		return fmt.Sprintf("virt-launcher-%s", uniqueName)
	}
	return fmt.Sprintf("virt-launcher-%s-", vmiName)
}

func resourceClaimBelongsToVMI(claim resourcev1.ResourceClaim, vmiName string) bool {
	podPrefix := virtLauncherPodNamePrefixForVMI(vmiName)
	for _, reservedFor := range claim.Status.ReservedFor {
		if reservedFor.Resource == "pods" && strings.HasPrefix(reservedFor.Name, podPrefix) {
			return true
		}
	}
	return false
}

func listResourceClaimsForVMIs(vmiNames ...string) ([]resourcev1.ResourceClaim, error) {
	resourceClaimList, err := kubevirt.Client().ResourceV1().ResourceClaims(testsuite.GetTestNamespace(nil)).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var claims []resourcev1.ResourceClaim
	for _, resourceClaim := range resourceClaimList.Items {
		for _, vmiName := range vmiNames {
			if resourceClaimBelongsToVMI(resourceClaim, vmiName) {
				claims = append(claims, resourceClaim)
				break
			}
		}
	}
	return claims, nil
}

func expectAllocatedDeviceCountForVMI(vmiName string, expectedDeviceCount int) {
	claims, err := listResourceClaimsForVMIs(vmiName)
	Expect(err).ToNot(HaveOccurred())
	Expect(claims).To(HaveLen(1))
	Expect(claims[0].Status.Allocation).NotTo(BeNil())
	Expect(claims[0].Status.Allocation.Devices.Results).To(HaveLen(expectedDeviceCount))
	Expect(claims[0].Status.ReservedFor).NotTo(BeEmpty())
	Expect(resourceClaimBelongsToVMI(claims[0], vmiName)).To(BeTrue())
}

func waitForBoundResourceClaimsForVMIs(expectedCount int, vmiNames ...string) {
	Eventually(func(g Gomega) {
		claims, err := listResourceClaimsForVMIs(vmiNames...)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(claims).To(HaveLen(expectedCount))
		for _, resourceClaim := range claims {
			var matchedVMIName string
			for _, vmiName := range vmiNames {
				if resourceClaimBelongsToVMI(resourceClaim, vmiName) {
					matchedVMIName = vmiName
					break
				}
			}
			g.Expect(matchedVMIName).NotTo(BeEmpty(), "claim %s should belong to one of the VMIs", resourceClaim.Name)

			g.Expect(resourceClaim.Status).NotTo(BeNil())
			g.Expect(resourceClaim.Status.Allocation).NotTo(BeNil())
			g.Expect(resourceClaim.Status.ReservedFor).NotTo(BeEmpty())
			g.Expect(resourceClaim.Status.ReservedFor[0].Resource).To(Equal("pods"))
			g.Expect(resourceClaim.Status.ReservedFor[0].Name).To(HavePrefix(virtLauncherPodNamePrefixForVMI(matchedVMIName)))
		}
	}, timeout, pollingInterval).Should(Succeed())
}

func createVFIOGPUResourceClaim() *resourcev1.ResourceClaim {
	namespace := testsuite.GetTestNamespace(nil)
	resourceClaim := &resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dra-vfio-claim-",
			Namespace:    namespace,
		},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: []resourcev1.DeviceRequest{vfioGPUDeviceRequest(draVfioRequestName)},
			},
		},
	}
	createdClaim, err := kubevirt.Client().ResourceV1().ResourceClaims(namespace).Create(context.Background(), resourceClaim, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return createdClaim
}

func waitForResourceClaimsToBeCreated(count int) {
	Eventually(func(g Gomega) {
		claims, err := kubevirt.Client().ResourceV1().ResourceClaims(testsuite.GetTestNamespace(nil)).List(context.Background(), metav1.ListOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(claims.Items).To(HaveLen(count))
	}, timeout, pollingInterval).Should(Succeed())
}
