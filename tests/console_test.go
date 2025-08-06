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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/testsuite"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libvmi"
)

func withNodeAffinityTo(label string, value string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Affinity = &k8sv1.Affinity{
			NodeAffinity: &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{Key: label, Operator: k8sv1.NodeSelectorOpIn, Values: []string{value}},
							},
						},
					},
				},
			},
		}
	}
}

var _ = Describe("[rfe_id:127][posneg:negative][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Console", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	expectConsoleOutput := func(vmi *v1.VirtualMachineInstance, expected string) {
		By("Checking that the console output equals to expected one")
		Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: expected},
		}, 120)).To(Succeed())
	}

	Describe("[rfe_id:127][posneg:negative][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with a serial console", func() {
			Context("with a cirros image", func() {

				It("[test_id:1588]should return that we are running cirros", func() {
					vmi := libvmi.NewCirros()
					vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
					expectConsoleOutput(
						vmi,
						"login as 'cirros' user",
					)
				})
			})

			Context("with a fedora image", func() {
				It("[sig-compute][test_id:1589]should return that we are running fedora", func() {
					vmi := libvmi.NewFedora()
					vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
					expectConsoleOutput(
						vmi,
						"Welcome to",
					)
				})
			})

			Context("with an alpine image", func() {
				type vmiBuilder func() *v1.VirtualMachineInstance

				newVirtualMachineInstanceWithAlpineFileDisk := func() *v1.VirtualMachineInstance {
					vmi, _ := tests.NewRandomVirtualMachineInstanceWithFileDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteOnce)
					return vmi
				}

				newVirtualMachineInstanceWithAlpineBlockDisk := func() *v1.VirtualMachineInstance {
					vmi, _ := tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteOnce)
					return vmi
				}

				DescribeTable("should return that we are running alpine", func(createVMI vmiBuilder) {
					vmi := createVMI()
					vmi = tests.RunVMIAndExpectLaunch(vmi, 120)
					expectConsoleOutput(vmi, "login")
				},
					Entry("[test_id:4637][storage-req]with Filesystem Disk", decorators.StorageReq, newVirtualMachineInstanceWithAlpineFileDisk),
					Entry("[test_id:4638][storage-req]with Block Disk", decorators.StorageReq, newVirtualMachineInstanceWithAlpineBlockDisk),
				)
			})

			It("[test_id:1590]should be able to reconnect to console multiple times", func() {
				vmi := libvmi.NewAlpine()
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

				for i := 0; i < 5; i++ {
					expectConsoleOutput(vmi, "login")
				}
			})

			It("[test_id:1591]should close console connection when new console connection is opened", func() {
				vmi := libvmi.NewAlpine()
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

				By("opening 1st console connection")
				expecter, errChan, err := console.NewExpecter(virtClient, vmi, 30*time.Second)
				Expect(err).ToNot(HaveOccurred())

				defer expecter.Close()

				By("expecting error on 1st console connection")
				go func() {
					defer GinkgoRecover()
					select {
					case receivedErr := <-errChan:
						Expect(receivedErr.Error()).To(ContainSubstring("close"))
					case <-time.After(60 * time.Second):
						Fail("timed out waiting for closed 1st connection")
					}
				}()

				By("opening 2nd console connection")
				expectConsoleOutput(vmi, "login")
			})

			It("[test_id:1592]should wait until the virtual machine is in running state and return a stream interface", func() {
				vmi := libvmi.NewAlpine()
				By("Creating a new VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				By("and connecting to it very quickly. Hopefully the VM is not yet up")
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: 60 * time.Second})
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1593]should not be connected if scheduled to non-existing host", func() {
				vmi := libvmi.NewAlpine(withNodeAffinityTo("kubernetes.io/hostname", "nonexistent"))

				By("Creating a new VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: 60 * time.Second})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Timeout trying to connect to the virtual machine instance"))
			})
		})

		Context("without a serial console", func() {

			It("[test_id:4118]should run but not be connectable via the serial console", func() {
				vmi := libvmi.NewAlpine()
				f := false
				vmi.Spec.Domain.Devices.AutoattachSerialConsole = &f
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

				runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred(), "should get vmi spec without problem")

				Expect(runningVMISpec.Devices.Serials).To(BeEmpty(), "should not have any serial consoles present")
				Expect(runningVMISpec.Devices.Consoles).To(BeEmpty(), "should not have any virtio console for serial consoles")

				By("failing to connect to serial console")
				_, err = virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).SerialConsole(vmi.ObjectMeta.Name, &kubecli.SerialConsoleOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("No serial consoles are present."), "serial console should not connect if there are no serial consoles present")
			})
		})
	})
})
