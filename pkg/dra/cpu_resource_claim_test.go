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
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("CPU ResourceClaim generation", func() {
	withSupplementalPool := func(opts ...libvmi.Option) []libvmi.Option {
		return append([]libvmi.Option{
			libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicySupplementalPool),
			libvmi.WithSupplementalPoolThreadCount(2),
			libvmi.WithIsolateEmulatorThread(),
		}, opts...)
	}

	DescribeTable("should ask for every host CPU the VM needs in a single request",
		func(expectedHostCPUs int64, opts []libvmi.Option) {
			vmi := libvmi.New(append([]libvmi.Option{libvmi.WithDedicatedCPUPlacement()}, opts...)...)

			claim := generateCPUResourceClaim(vmi, "claim")

			Expect(claim.Spec.Devices.Requests).To(HaveLen(1))
			Expect(claim.Spec.Devices.Requests[0].Name).To(Equal(CPURequestName))
			Expect(claim.Spec.Devices.Requests[0].Exactly.DeviceClassName).To(Equal(CPUDeviceClassName))
			Expect(requestedHostCPUs(claim, CPURequestName)).To(Equal(expectedHostCPUs))
		},
		Entry("gathering every guest vCPU", int64(8),
			[]libvmi.Option{libvmi.WithCPUCount(2, 2, 2)}),
		Entry("adding the supplemental CPUs to the guest vCPUs", int64(7),
			withSupplementalPool(libvmi.WithCPUCount(4, 1, 1))),
		Entry("counting every socket of a multi socket guest", int64(11),
			withSupplementalPool(libvmi.WithCPUCount(4, 1, 2))),
		Entry("using two emulator thread CPUs under even parity", int64(10),
			withSupplementalPool(
				libvmi.WithCPUCount(6, 1, 1),
				libvmi.WithAnnotation(v1.EmulatorThreadCompleteToEvenParity, ""))),
		Entry("counting an odd IO thread count once under even parity", int64(10),
			[]libvmi.Option{
				libvmi.WithCPUCount(6, 1, 1),
				libvmi.WithIsolateEmulatorThread(),
				libvmi.WithIOThreadsPolicy(v1.IOThreadsPolicySupplementalPool),
				libvmi.WithSupplementalPoolThreadCount(3),
				libvmi.WithAnnotation(v1.EmulatorThreadCompleteToEvenParity, ""),
			}),
	)
})

var _ = Describe("CPU ResourceClaim creation", func() {
	newVMI := func(uid string, opts ...libvmi.Option) *v1.VirtualMachineInstance {
		vmi := libvmi.New(append([]libvmi.Option{
			libvmi.WithName("testvmi"),
			libvmi.WithNamespace("default"),
			libvmi.WithDedicatedCPUPlacement(),
		}, opts...)...)
		vmi.UID = types.UID(uid)
		return vmi
	}

	newClient := func(k8sClient *k8sfake.Clientset) *kubecli.MockKubevirtClient {
		client := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		client.EXPECT().ResourceV1().Return(k8sClient.ResourceV1()).AnyTimes()
		return client
	}

	storedClaim := func(k8sClient *k8sfake.Clientset, vmi *v1.VirtualMachineInstance) *resourcev1.ResourceClaim {
		claim, err := k8sClient.ResourceV1().ResourceClaims(vmi.Namespace).
			Get(context.Background(), CPUResourceClaimName(vmi.Name), metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return claim
	}

	It("should reject an existing claim controlled by an old VMI", func() {
		vmi := newVMI("current-uid")
		staleClaim := generateCPUResourceClaim(newVMI("old-uid"), CPUResourceClaimName(vmi.Name))
		k8sClient := k8sfake.NewSimpleClientset(staleClaim)

		err := CreateCPUResourceClaim(context.Background(), vmi, newClient(k8sClient))
		Expect(err).To(MatchError(ContainSubstring("is not controlled by VMI UID current-uid")))
	})

	It("should create the claim when none exists yet", func() {
		vmi := newVMI("current-uid", libvmi.WithCPUCount(4, 1, 1))
		k8sClient := k8sfake.NewSimpleClientset()

		err := CreateCPUResourceClaim(context.Background(), vmi, newClient(k8sClient))
		Expect(err).ToNot(HaveOccurred())

		claim := storedClaim(k8sClient, vmi)
		Expect(metav1.GetControllerOf(claim).UID).To(Equal(vmi.UID))
		Expect(requestedHostCPUs(claim, CPURequestName)).To(Equal(int64(4)))
	})

	It("should return the lookup error when an existing claim cannot be read", func() {
		vmi := newVMI("current-uid")
		existingClaim := generateCPUResourceClaim(vmi, CPUResourceClaimName(vmi.Name))
		k8sClient := k8sfake.NewSimpleClientset(existingClaim)
		k8sClient.PrependReactor("get", "resourceclaims",
			func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.NewInternalError(fmt.Errorf("etcd is down"))
			})

		err := CreateCPUResourceClaim(context.Background(), vmi, newClient(k8sClient))
		Expect(err).To(MatchError(ContainSubstring("etcd is down")))

		verbs := make([]string, 0, len(k8sClient.Actions()))
		for _, action := range k8sClient.Actions() {
			verbs = append(verbs, action.GetVerb())
		}
		Expect(verbs).To(Equal([]string{"create", "get"}))
	})

	It("should create a separate claim for each VMI in the namespace", func() {
		first := newVMI("uid-1")
		first.Name = "vm-a"
		second := newVMI("uid-2")
		second.Name = "vm-b"
		k8sClient := k8sfake.NewSimpleClientset()
		client := newClient(k8sClient)

		Expect(CreateCPUResourceClaim(context.Background(), first, client)).To(Succeed())
		Expect(CreateCPUResourceClaim(context.Background(), second, client)).To(Succeed())

		firstClaim := storedClaim(k8sClient, first)
		secondClaim := storedClaim(k8sClient, second)
		Expect(firstClaim.Name).To(Equal(CPUResourceClaimName("vm-a")))
		Expect(secondClaim.Name).To(Equal(CPUResourceClaimName("vm-b")))
		Expect(firstClaim.Name).NotTo(Equal(secondClaim.Name))
		Expect(metav1.GetControllerOf(firstClaim).UID).To(Equal(first.UID))
		Expect(metav1.GetControllerOf(secondClaim).UID).To(Equal(second.UID))
	})
})

var _ = Describe("CPUResourceClaimName", func() {
	It("should include the VMI name", func() {
		Expect(CPUResourceClaimName("testvmi")).To(Equal("testvmi-" + cpuResourceClaimNameSuffix))
	})

	It("should keep a VMI name that fits inside the object name limit", func() {
		room := validation.DNS1123SubdomainMaxLength - len(cpuResourceClaimNameSuffix) - 1
		vmiName := strings.Repeat("b", room)

		Expect(CPUResourceClaimName(vmiName)).To(Equal(vmiName + "-" + cpuResourceClaimNameSuffix))
		Expect(CPUResourceClaimName(vmiName)).To(HaveLen(validation.DNS1123SubdomainMaxLength))
	})

	It("should stay a valid object name when the VMI name is very long", func() {
		longName := strings.Repeat("a", validation.DNS1123SubdomainMaxLength)
		otherLongName := strings.Repeat("a", validation.DNS1123SubdomainMaxLength-1) + "b"

		first := CPUResourceClaimName(longName)
		second := CPUResourceClaimName(otherLongName)

		Expect(len(first)).To(BeNumerically("<=", validation.DNS1123SubdomainMaxLength))
		Expect(len(second)).To(BeNumerically("<=", validation.DNS1123SubdomainMaxLength))
		Expect(validation.IsDNS1123Subdomain(first)).To(BeEmpty())
		Expect(validation.IsDNS1123Subdomain(second)).To(BeEmpty())
		Expect(first).NotTo(Equal(second))
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
