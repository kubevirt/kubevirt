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

package dra

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("CPU ResourceClaim synthesis", func() {
	It("should gather every guest vCPU into a single unconstrained request", func() {
		vmi := libvmi.New(
			libvmi.WithName("testvmi"),
			libvmi.WithNamespace("default"),
			libvmi.WithDedicatedCPUPlacement(),
			libvmi.WithCPUCount(2, 2, 2),
		)
		vmi.UID = "abc-123"

		claim := generateCPUResourceClaim(vmi, CPUResourceClaimName(vmi.Name))
		Expect(claim.Spec.Devices.Requests).To(HaveLen(1))
		Expect(claim.Spec.Devices.Requests[0].Name).To(Equal(CPURequestName))
		Expect(claim.Spec.Devices.Requests[0].Exactly.DeviceClassName).To(Equal(CPUDeviceClassName))
		Expect(requestedHostCPUs(claim, CPURequestName)).To(Equal(int64(8)))
	})

	It("should add the supplemental CPUs to the same request as the guest vCPUs", func() {
		vmi := libvmi.New(
			libvmi.WithDedicatedCPUPlacement(),
			libvmi.WithCPUCount(4, 1, 1),
			libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicySupplementalPool),
			libvmi.WithSupplementalPoolThreadCount(2),
			libvmi.WithIsolateEmulatorThread(),
		)

		claim := generateCPUResourceClaim(vmi, "claim")
		Expect(claim.Spec.Devices.Requests).To(HaveLen(1))
		Expect(requestedHostCPUs(claim, CPURequestName)).To(Equal(int64(7)))
	})

	It("should count every socket of a multi socket guest towards the request", func() {
		vmi := libvmi.New(
			libvmi.WithDedicatedCPUPlacement(),
			libvmi.WithCPUCount(4, 1, 2),
			libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicySupplementalPool),
			libvmi.WithSupplementalPoolThreadCount(2),
			libvmi.WithIsolateEmulatorThread(),
		)

		claim := generateCPUResourceClaim(vmi, "claim")
		Expect(claim.Spec.Devices.Requests).To(HaveLen(1))
		Expect(requestedHostCPUs(claim, CPURequestName)).To(Equal(int64(11)))
	})

	It("should use two emulator thread CPUs with even parity annotation", func() {
		vmi := libvmi.New(
			libvmi.WithDedicatedCPUPlacement(),
			libvmi.WithCPUCount(6, 1, 1),
			libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicySupplementalPool),
			libvmi.WithSupplementalPoolThreadCount(2),
			libvmi.WithIsolateEmulatorThread(),
		)
		vmi.Annotations = map[string]string{v1.EmulatorThreadCompleteToEvenParity: ""}

		claim := generateCPUResourceClaim(vmi, "claim")
		Expect(requestedHostCPUs(claim, CPURequestName)).To(Equal(int64(10)))
	})

	It("should default to a single vCPU when the topology is unset", func() {
		vmi := libvmi.New(libvmi.WithDedicatedCPUPlacement())
		claim := generateCPUResourceClaim(vmi, "claim")
		Expect(claim.Spec.Devices.Requests).To(HaveLen(1))
		Expect(claim.Spec.Devices.Requests[0].Exactly.DeviceClassName).To(Equal(CPUDeviceClassName))
		Expect(requestedHostCPUs(claim, CPURequestName)).To(Equal(int64(1)))
	})
})

var _ = Describe("CPU ResourceClaim creation", func() {
	newVMI := func(uid string) *v1.VirtualMachineInstance {
		vmi := libvmi.New(
			libvmi.WithName("testvmi"),
			libvmi.WithNamespace("default"),
			libvmi.WithDedicatedCPUPlacement(),
		)
		vmi.UID = types.UID(uid)
		return vmi
	}

	newClient := func(k8sClient *k8sfake.Clientset) *kubecli.MockKubevirtClient {
		client := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		client.EXPECT().ResourceV1().Return(k8sClient.ResourceV1()).AnyTimes()
		return client
	}

	It("should reject an existing claim controlled by an old VMI", func() {
		vmi := newVMI("current-uid")
		staleClaim := generateCPUResourceClaim(newVMI("old-uid"), CPUResourceClaimName(vmi.Name))
		k8sClient := k8sfake.NewSimpleClientset(staleClaim)

		err := CreateCPUResourceClaim(vmi, newClient(k8sClient))
		Expect(err).To(MatchError(ContainSubstring("is not controlled by VMI UID current-uid")))
	})
})

func requestedHostCPUs(claim *resourcev1.ResourceClaim, requestName string) int64 {
	for _, request := range claim.Spec.Devices.Requests {
		if request.Name == requestName {
			capacity := request.Exactly.Capacity.Requests[CPUCapacityAttribute]
			return capacity.Value()
		}
	}
	Fail("no request named " + requestName + " in the claim")
	return 0
}
