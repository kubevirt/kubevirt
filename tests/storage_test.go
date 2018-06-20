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
	"flag"
	"fmt"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

const (
	diskSerial = "FB-fb_18030C10002032"
)

type VMICreationFunc func(string) *v1.VirtualMachineInstance

var _ = Describe("Storage", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	RunVMIAndExpectLaunch := func(vmi *v1.VirtualMachineInstance, withAuth bool, timeout int) *v1.VirtualMachineInstance {
		By("Starting a VirtualMachineInstance")

		var obj *v1.VirtualMachineInstance
		var err error
		Eventually(func() error {
			obj, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
		By("Waiting until the VirtualMachineInstance will start")
		tests.WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
		return obj
	}

	LaunchVMI := func(vmi *v1.VirtualMachineInstance) {
		By("Starting a VirtualMachineInstance")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
		Expect(err).To(BeNil())

		By("Waiting the VirtualMachineInstance start")
		_, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
		Expect(tests.WaitForSuccessfulVMIStart(obj)).ToNot(BeEmpty())
	}

	VerifyUserDataVMI := func(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
		By("Expecting the VirtualMachineInstance console")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 10*time.Second)
		defer expecter.Close()
		Expect(err).ToNot(HaveOccurred())

		By("Checking that the VirtualMachineInstance serial console output equals to expected one")
		resp, err := expecter.ExpectBatch(commands, timeout)
		log.DefaultLogger().Object(vmi).Infof("%v", resp)
		Expect(err).ToNot(HaveOccurred())
	}

	Describe("Starting a VirtualMachineInstance", func() {
		Context("with Alpine PVC", func() {
			table.DescribeTable("should be successfully started", func(newVMI VMICreationFunc) {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := newVMI(tests.DiskAlpineHostPath)
				RunVMIAndExpectLaunch(vmi, false, 90)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()
			},
				table.Entry("with Disk PVC", tests.NewRandomVMIWithPVC),
				table.Entry("with CDRom PVC", tests.NewRandomVMIWithCDRom),
			)

			table.DescribeTable("should be successfully started and stopped multiple times", func(newVMI VMICreationFunc) {
				vmi := newVMI(tests.DiskAlpineHostPath)

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					vmi := RunVMIAndExpectLaunch(vmi, false, 90)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VirtualMachineInstance console has expected output")
						expecter, err := tests.LoggedInAlpineExpecter(vmi)
						Expect(err).To(BeNil())
						expecter.Close()
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())
					tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
			},
				table.Entry("with Disk PVC", tests.NewRandomVMIWithPVC),
				table.Entry("with CDRom PVC", tests.NewRandomVMIWithCDRom),
			)
		})

		Context("With an emptyDisk defined", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			It("should create a writeable emptyDisk with the right capacity", func() {

				// Start the VirtualMachineInstance with the empty disk attached
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "echo hi!")
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       "emptydisk1",
					VolumeName: "emptydiskvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "emptydiskvolume1",
					VolumeSource: v1.VolumeSource{
						EmptyDisk: &v1.EmptyDiskSource{
							Capacity: resource.MustParse("2Gi"),
						},
					},
				})
				RunVMIAndExpectLaunch(vmi, false, 90)

				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).To(BeNil())
				defer expecter.Close()

				By("Checking that /dev/vdc has a capacity of 2Gi")
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo blockdev --getsize64 /dev/vdc\n"},
					&expect.BExp{R: "2147483648"}, // 2Gi in bytes
				}, 10*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).To(BeNil())

				By("Checking if we can write to /dev/vdc")
				res, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo mkfs.ext4 /dev/vdc\n"},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "0"},
				}, 20*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).To(BeNil())
			})

			It("should create a writeable emptyDisk with specified serial number", func() {
				// Start a Fedora VirtualMachineInstance with the empty disk with SN attached
				userData := fmt.Sprintf(
					"#cloud-config\npassword: %s\nchpasswd: { expire: False }\nssh_authorized_keys:\n  - %s",
					fedoraPassword,
					sshAuthorizedKey,
				)
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(tests.RegistryDiskFor(tests.RegistryDiskFedora), userData)

				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       "emptydisk1",
					VolumeName: "emptydiskvolume1",
					Serial:     diskSerial,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "emptydiskvolume1",
					VolumeSource: v1.VolumeSource{
						EmptyDisk: &v1.EmptyDiskSource{
							Capacity: resource.MustParse("1Gi"),
						},
					},
				})
				LaunchVMI(vmi)

				// NOTE(jdg): In this case we're only assigning an SN to one of the drives,
				// so if we even have a by-id dir that's a win.  Also since there's only one
				// attached drive with an SN we don't need to parse out the ls response.
				By("Checking that ls /dev/disk/by-id returns our serial number")
				VerifyUserDataVMI(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "login:"},
					&expect.BSnd{S: "fedora\n"},
					&expect.BExp{R: "Password:"},
					&expect.BSnd{S: fedoraPassword + "\n"},
					&expect.BExp{R: "$"},
					&expect.BSnd{S: "ls /dev/disk/by-id \n"},
					&expect.BExp{R: "virtio-" + diskSerial},
				}, time.Second*300)

			})

		})

		Context("With ephemeral alpine PVC", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			It("should be successfully started", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithEphemeralPVC(tests.DiskAlpineHostPath)
				RunVMIAndExpectLaunch(vmi, false, 90)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).To(BeNil())
				expecter.Close()
			})

			It("should not persist data", func() {
				vmi := tests.NewRandomVMIWithEphemeralPVC(tests.DiskAlpineHostPath)

				By("Starting the VirtualMachineInstance")
				createdVMI := RunVMIAndExpectLaunch(vmi, false, 90)

				By("Writing an arbitrary file to it's EFI partition")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				_, err = expecter.ExpectBatch([]expect.Batcher{
					// Because "/" is mounted on tmpfs, we need something that normally persists writes - /dev/sda2 is the EFI partition formatted as vFAT.
					&expect.BSnd{S: "mount /dev/sda2 /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "0"},
					&expect.BSnd{S: "echo content > /mnt/checkpoint\n"},
					// The QEMU process will be killed, therefore the write must be flushed to the disk.
					&expect.BSnd{S: "sync\n"},
				}, 200*time.Second)
				Expect(err).ToNot(HaveOccurred())

				By("Killing a VirtualMachineInstance")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForVirtualMachineToDisappearWithTimeout(createdVMI, 120)

				By("Starting the VirtualMachineInstance again")
				RunVMIAndExpectLaunch(vmi, false, 90)

				By("Making sure that the previously written file is not present")
				expecter, err = tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				_, err = expecter.ExpectBatch([]expect.Batcher{
					// Same story as when first starting the VirtualMachineInstance - the checkpoint, if persisted, is located at /dev/sda2.
					&expect.BSnd{S: "mount /dev/sda2 /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "0"},
					&expect.BSnd{S: "cat /mnt/checkpoint &> /dev/null\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "1"},
				}, 200*time.Second)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("With VirtualMachineInstance with two PVCs", func() {
			BeforeEach(func() {
				// Setup second PVC to use in this context
				tests.CreateHostPathPv(tests.CustomHostPath, tests.HostPathCustom)
				tests.CreatePVC(tests.CustomHostPath, "1Gi")
			}, 120)

			AfterEach(func() {
				tests.DeletePVC(tests.CustomHostPath)
				tests.DeletePV(tests.CustomHostPath)
			}, 120)

			It("should start vmi multiple times", func() {
				vmi := tests.NewRandomVMIWithPVC(tests.DiskAlpineHostPath)
				tests.AddPVCDisk(vmi, "disk1", "virtio", tests.DiskCustomHostPath)

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					obj := RunVMIAndExpectLaunch(vmi, false, 120)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the second disk is present")
						expecter, err := tests.LoggedInAlpineExpecter(vmi)
						Expect(err).ToNot(HaveOccurred())
						defer expecter.Close()

						_, err = expecter.ExpectBatch([]expect.Batcher{
							&expect.BSnd{S: "blockdev --getsize64 /dev/vdb\n"},
							&expect.BExp{R: "67108864"},
						}, 200*time.Second)
						Expect(err).ToNot(HaveOccurred())
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())

					tests.WaitForVirtualMachineToDisappearWithTimeout(obj, 120)
				}
			})
		})
	})
})
