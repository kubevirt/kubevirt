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

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[rfe_id:127][posneg:negative][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Console", func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error

		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	RunVMIAndWaitForStart := func(vmi *v1.VirtualMachineInstance) {
		By("Creating a new VirtualMachineInstance")
		Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(util.NamespaceTestDefault).Body(vmi).Do(context.Background()).Error()).To(Succeed())

		By("Waiting until it starts")
		tests.WaitForSuccessfulVMIStart(vmi)
	}

	ExpectConsoleOutput := func(vmi *v1.VirtualMachineInstance, expected string) {
		By("Checking that the console output equals to expected one")
		Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: expected},
		}, 120)).To(Succeed())
	}

	OpenConsole := func(vmi *v1.VirtualMachineInstance) (expect.Expecter, <-chan error) {
		By("Expecting the VirtualMachineInstance console")
		expecter, errChan, err := console.NewExpecter(virtClient, vmi, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
		return expecter, errChan
	}

	deleteDataVolume := func(dv *cdiv1.DataVolume) {
		if dv != nil {
			By("Deleting the DataVolume")
			ExpectWithOffset(1, virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
		}
	}

	Describe("[rfe_id:127][posneg:negative][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with a serial console", func() {
			Context("with a cirros image", func() {

				It("[test_id:1588]should return that we are running cirros", func() {
					vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
					RunVMIAndWaitForStart(vmi)
					ExpectConsoleOutput(
						vmi,
						"login as 'cirros' user",
					)
				})
			})

			Context("with a fedora image", func() {
				It("[sig-compute][test_id:1589]should return that we are running fedora", func() {
					vmi := tests.NewRandomVMIWithEphemeralDiskHighMemory(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling))
					RunVMIAndWaitForStart(vmi)
					ExpectConsoleOutput(
						vmi,
						"Welcome to",
					)
				})
			})

			Context("with an alpine image", func() {
				type vmiBuilder func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume)

				newVirtualMachineInstanceWithAlpineContainerDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
					return tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine)), nil
				}

				newVirtualMachineInstanceWithAlpineOCSFileDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
					return tests.NewRandomVirtualMachineInstanceWithOCSDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeFilesystem)
				}

				newVirtualMachineInstanceWithAlpineOCSBlockDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
					return tests.NewRandomVirtualMachineInstanceWithOCSDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeBlock)
				}

				table.DescribeTable("should return that we are running alpine", func(createVMI vmiBuilder) {
					vmi, dv := createVMI()
					defer deleteDataVolume(dv)
					RunVMIAndWaitForStart(vmi)
					ExpectConsoleOutput(vmi, "login")
				},
					table.Entry("[test_id:4636]with ContainerDisk", newVirtualMachineInstanceWithAlpineContainerDisk),
					table.Entry("[test_id:4637][rook-ceph]with OCS Filesystem Disk", newVirtualMachineInstanceWithAlpineOCSFileDisk),
					table.Entry("[test_id:4638][rook-ceph]with OCS Block Disk", newVirtualMachineInstanceWithAlpineOCSBlockDisk),
				)
			})

			It("[test_id:1590]should be able to reconnect to console multiple times", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))

				RunVMIAndWaitForStart(vmi)

				for i := 0; i < 5; i++ {
					ExpectConsoleOutput(vmi, "login")
				}
			})

			It("[test_id:1591]should close console connection when new console connection is opened", func(done Done) {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))

				RunVMIAndWaitForStart(vmi)

				By("opening 1st console connection")
				expecter, errChan := OpenConsole(vmi)
				defer expecter.Close()

				By("expecting error on 1st console connection")
				go func() {
					defer GinkgoRecover()
					select {
					case receivedErr := <-errChan:
						Expect(receivedErr.Error()).To(ContainSubstring("close"))
						close(done)
					case <-time.After(60 * time.Second):
						Fail("timed out waiting for closed 1st connection")
					}
				}()

				By("opening 2nd console connection")
				ExpectConsoleOutput(vmi, "login")
			}, 220)

			It("[test_id:1592]should wait until the virtual machine is in running state and return a stream interface", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				By("Creating a new VirtualMachineInstance")
				_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: 30 * time.Second})
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1593]should fail waiting for the virtual machine instance to be running", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi.Spec.Affinity = &k8sv1.Affinity{
					NodeAffinity: &k8sv1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
							NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
								{
									MatchExpressions: []k8sv1.NodeSelectorRequirement{
										{Key: "kubernetes.io/hostname", Operator: k8sv1.NodeSelectorOpIn, Values: []string{"notexist"}},
									},
								},
							},
						},
					},
				}

				By("Creating a new VirtualMachineInstance")
				Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(util.NamespaceTestDefault).Body(vmi).Do(context.Background()).Error()).To(Succeed())

				_, err := virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: 30 * time.Second})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Timeout trying to connect to the virtual machine instance"))
			})

			It("[test_id:1594]should fail waiting for the expecter", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi.Spec.Affinity = &k8sv1.Affinity{
					NodeAffinity: &k8sv1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
							NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
								{
									MatchExpressions: []k8sv1.NodeSelectorRequirement{
										{Key: "kubernetes.io/hostname", Operator: k8sv1.NodeSelectorOpIn, Values: []string{"notexist"}},
									},
								},
							},
						},
					},
				}

				By("Creating a new VirtualMachineInstance")
				Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(util.NamespaceTestDefault).Body(vmi).Do(context.Background()).Error()).To(Succeed())

				By("Expecting the VirtualMachineInstance console")
				_, _, err := console.NewExpecter(virtClient, vmi, 30*time.Second)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Timeout trying to connect to the virtual machine instance"))
			})
		})

		Context("without a serial console", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				f := false
				vmi.Spec.Domain.Devices.AutoattachSerialConsole = &f
			})

			It("[test_id:4116]should create the vmi without any issue", func() {
				tests.RunVMIAndExpectLaunch(vmi, 30)
			})

			It("[test_id:4117]should not have the  serial console in xml", func() {
				tests.RunVMIAndExpectLaunch(vmi, 30)

				runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred(), "should get vmi spec without problem")

				Expect(len(runningVMISpec.Devices.Serials)).To(Equal(0), "should not have any serial consoles present")
				Expect(len(runningVMISpec.Devices.Consoles)).To(Equal(0), "should not have any virtio console for serial consoles")
			})

			It("[test_id:4118]should not connect to the serial console", func() {
				vmi = tests.RunVMIAndExpectLaunch(vmi, 30)

				_, err := virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).SerialConsole(vmi.ObjectMeta.Name, &kubecli.SerialConsoleOptions{})

				Expect(err.Error()).To(Equal("No serial consoles are present."), "serial console should not connect if there are no serial consoles present")
			})
		})
	})
})
