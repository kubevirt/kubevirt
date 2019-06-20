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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:609]VMIheadless", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
	})

	Describe("[rfe_id:609]Creating a VirtualMachineInstance", func() {

		Context("with headless", func() {

			BeforeEach(func() {
				f := false
				vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = &f
			})

			It("should create headless vmi without any issue", func() {
				tests.RunVMIAndExpectLaunch(vmi, 30)
			})

			It("[test_id:714][posneg:positive]should not have vnc graphic device in xml", func() {
				tests.RunVMIAndExpectLaunch(vmi, 30)

				runningVMISpec, err := tests.GetRunningVMISpec(vmi)
				Expect(err).ToNot(HaveOccurred(), "should get vmi spec without problem")

				Expect(len(runningVMISpec.Devices.Graphics)).To(Equal(0), "should not have any graphics devices present")
			})

			It("[test_id:737][posneg:positive]should match memory with overcommit enabled", func() {
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("100M"),
					},
					OvercommitGuestOverhead: true,
				}
				vmi = tests.RunVMIAndExpectLaunch(vmi, 30)

				readyPod := tests.GetPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				computeContainer := tests.GetComputeContainerOfPod(readyPod)

				Expect(computeContainer.Resources.Requests.Memory().String()).To(Equal("100M"))
			})

			It("[test_id:2444][posneg:negative]should not match memory with overcommit disabled", func() {
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("100M"),
					},
				}
				vmi = tests.RunVMIAndExpectLaunch(vmi, 30)

				readyPod := tests.GetPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				computeContainer := tests.GetComputeContainerOfPod(readyPod)

				Expect(computeContainer.Resources.Requests.Memory().String()).ToNot(Equal("100M"))
			})

			It("[test_id:713]should have more memory on pod when headless", func() {
				normalVmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))

				vmi = tests.RunVMIAndExpectLaunch(vmi, 30)
				normalVmi = tests.RunVMIAndExpectLaunch(normalVmi, 30)

				readyPod := tests.GetPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				computeContainer := tests.GetComputeContainerOfPod(readyPod)

				normalReadyPod := tests.GetPodByVirtualMachineInstance(normalVmi, tests.NamespaceTestDefault)
				normalComputeContainer := tests.GetComputeContainerOfPod(normalReadyPod)

				memDiff := normalComputeContainer.Resources.Requests.Memory()
				memDiff.Sub(*computeContainer.Resources.Requests.Memory())

				Expect(memDiff.ScaledValue(resource.Mega) > 15).To(BeTrue(), "memory difference between headless and normal should be roughly 16M")
			})

			It("[test_id:738][posneg:negative]should not connect to VNC", func() {
				vmi = tests.RunVMIAndExpectLaunch(vmi, 30)

				_, err := virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).VNC(vmi.ObjectMeta.Name)

				Expect(err.Error()).To(Equal("Can't connect to websocket (400): No graphics devices are present.\n"), "vnc should not connect on headless VM")
			})

			It("[test_id:709][posneg:positive]should connect to console", func() {
				vmi = tests.RunVMIAndExpectLaunch(vmi, 30)

				By("checking that console works")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()
			})

		})

		Context("without headless", func() {

			It("[test_id:714][posneg:negative]should have one vnc graphic device in xml", func() {
				tests.RunVMIAndExpectLaunch(vmi, 30)

				runningVMISpec, err := tests.GetRunningVMISpec(vmi)
				Expect(err).ToNot(HaveOccurred(), "should get vmi spec without problem")

				vncCount := 0
				for _, gr := range runningVMISpec.Devices.Graphics {
					if strings.ToLower(gr.Type) == "vnc" {
						vncCount += 1
					}
				}
				Expect(vncCount).To(Equal(1), "should have exactly one VNC device")
			})

		})

	})

})
