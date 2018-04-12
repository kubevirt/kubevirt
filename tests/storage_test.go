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
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Storage", func() {

	nodeName := ""
	nodeIp := ""
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()

		nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(nodes.Items).ToNot(BeEmpty())
		nodeName = nodes.Items[0].Name
		for _, addr := range nodes.Items[0].Status.Addresses {
			if addr.Type == k8sv1.NodeInternalIP {
				nodeIp = addr.Address
				break
			}
		}
		Expect(nodeIp).ToNot(Equal(""))
	})

	getTargetLogs := func(tailLines int64) string {
		pods, err := virtClient.CoreV1().Pods(metav1.NamespaceSystem).List(metav1.ListOptions{LabelSelector: v1.AppLabel + " in (iscsi-demo-target)"})
		Expect(err).ToNot(HaveOccurred())

		//FIXME Sometimes pods hang in terminating state, select the pod which does not have a deletion timestamp
		podName := ""
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp == nil {
				if pod.Status.HostIP == nodeIp {
					podName = pod.ObjectMeta.Name
					break
				}
			}
		}
		Expect(podName).ToNot(BeEmpty())

		By("Getting the ISCSI pod logs")
		logsRaw, err := virtClient.CoreV1().
			Pods(metav1.NamespaceSystem).
			GetLogs(podName,
				&k8sv1.PodLogOptions{TailLines: &tailLines}).
			DoRaw()
		Expect(err).To(BeNil())

		return string(logsRaw)
	}

	checkReadiness := func() {
		logs := getTargetLogs(75)
		By("Checking that ISCSI is ready")
		Expect(logs).To(ContainSubstring("Target 1: iqn.2017-01.io.kubevirt:sn.42"))
		Expect(logs).To(ContainSubstring("Driver: iscsi"))
		Expect(logs).To(ContainSubstring("State: ready"))
	}

	RunVMAndExpectLaunch := func(vm *v1.VirtualMachine, withAuth bool, timeout int) runtime.Object {
		By("Starting a VM")

		var obj runtime.Object
		var err error
		Eventually(func() error {
			obj, err = virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
		By("Waiting until the VM will start")
		tests.WaitForSuccessfulVMStartWithTimeout(obj, timeout)
		return obj
	}

	Context("with fresh iSCSI target", func() {
		It("should be available and ready", func() {
			checkReadiness()
		})
	})

	Describe("Starting a VM", func() {
		Context("with Alpine PVC", func() {
			It("should be successfully started", func(done Done) {
				checkReadiness()

				// Start the VM with the PVC attached
				vm := tests.NewRandomVMWithPVC(tests.DiskAlpineISCSI)
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				RunVMAndExpectLaunch(vm, false, 45)

				expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, 10*time.Second)
				defer expecter.Close()
				Expect(err).To(BeNil())

				By("Checking that the VM console has expected output")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BExp{R: "Welcome to Alpine"},
				}, 200*time.Second)
				Expect(err).To(BeNil())

				close(done)
			}, 240)

			It("should be successfully started and stopped multiple times", func(done Done) {
				checkReadiness()

				vm := tests.NewRandomVMWithPVC(tests.DiskAlpineISCSI)
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}

				num := 3
				By("Starting and stopping the VM number of times")
				for i := 1; i <= num; i++ {
					obj := RunVMAndExpectLaunch(vm, false, 90)

					// Verify console on last iteration to verify the VM is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VM console has expected output")
						expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, 10*time.Second)
						defer expecter.Close()
						Expect(err).To(BeNil())
						_, err = expecter.ExpectBatch([]expect.Batcher{
							&expect.BExp{R: "Welcome to Alpine"},
						}, 200*time.Second)
						Expect(err).To(BeNil())
					}

					err = virtClient.VM(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())

					tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)
				}
				close(done)
			}, 240)
		})

		Context("With an emptyDisk defined", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VM.
			It("should create a writeable emptyDisk with the right capacity", func(done Done) {

				// Start the VM with the empty disk attached
				vm := tests.NewRandomVMWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "echo hi!")
				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       "emptydisk1",
					VolumeName: "emptydiskvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
					Name: "emptydiskvolume1",
					VolumeSource: v1.VolumeSource{
						EmptyDisk: &v1.EmptyDiskSource{
							Capacity: resource.MustParse("2Gi"),
						},
					},
				})
				RunVMAndExpectLaunch(vm, false, 45)

				expecter, err := tests.LoggedInCirrosExpecter(vm)
				defer expecter.Close()
				Expect(err).To(BeNil())

				By("Checking that /dev/vdc has a capacity of 2Gi")
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo blockdev --getsize64 /dev/vdc\n"},
					&expect.BExp{R: "2147483648"}, // 2Gi in bytes
				}, 10*time.Second)
				log.DefaultLogger().Object(vm).Infof("%v", res)
				Expect(err).To(BeNil())

				By("Checking if we can write to /dev/vdc")
				res, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo mkfs.ext4 /dev/vdc\n"},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "0"},
				}, 20*time.Second)
				log.DefaultLogger().Object(vm).Infof("%v", res)
				Expect(err).To(BeNil())

				close(done)
			}, 240)

		})

		Context("With ephemeral alpine PVC", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VM.
			It("should be successfully started", func(done Done) {
				checkReadiness()

				// Start the VM with the PVC attached
				vm := tests.NewRandomVMWithEphemeralPVC(tests.DiskAlpineISCSI)
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				RunVMAndExpectLaunch(vm, false, 45)

				expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, 10*time.Second)
				defer expecter.Close()
				Expect(err).To(BeNil())

				By("Checking that the VM console has expected output")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BExp{R: "Welcome to Alpine"},
				}, 200*time.Second)
				Expect(err).To(BeNil())

				close(done)
			}, 240)

			It("should not persist data", func(done Done) {
				checkReadiness()
				vm := tests.NewRandomVMWithEphemeralPVC(tests.DiskAlpineISCSI)
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}

				By("Starting the VM")
				obj := RunVMAndExpectLaunch(vm, false, 90)

				By("Writing an arbitrary file to it's EFI partition")
				expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, 10*time.Second)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BExp{R: "Welcome to Alpine"},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "login"},
					&expect.BSnd{S: "root\n"},
					&expect.BExp{R: "#"},
					// Because "/" is mounted on tmpfs, we need something that normally persists writes - /dev/sda2 is the EFI partition formatted as vFAT.
					&expect.BSnd{S: "mount /dev/sda2 /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "0"},
					&expect.BSnd{S: "echo content > /mnt/checkpoint\n"},
					// The QEMU process will be killed, therefore the write must be flushed to the disk.
					&expect.BSnd{S: "sync\n"},
				}, 200*time.Second)
				Expect(err).ToNot(HaveOccurred())

				By("Killing a VM")
				err = virtClient.VM(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)

				By("Starting the VM again")
				RunVMAndExpectLaunch(vm, false, 90)

				By("Making sure that the previously written file is not present")
				expecter, _, err = tests.NewConsoleExpecter(virtClient, vm, 10*time.Second)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BExp{R: "Welcome to Alpine"},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "login"},
					&expect.BSnd{S: "root\n"},
					&expect.BExp{R: "#"},
					// Same story as when first starting the VM - the checkpoint, if persisted, is located at /dev/sda2.
					&expect.BSnd{S: "mount /dev/sda2 /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "0"},
					&expect.BSnd{S: "cat /mnt/checkpoint &> /dev/null\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "1"},
				}, 200*time.Second)
				Expect(err).ToNot(HaveOccurred())

				close(done)
			}, 400)
		})

		Context("With VM with two PVCs", func() {
			BeforeEach(func() {
				// Setup second PVC to use in this context
				tests.CreatePvISCSI(tests.CustomISCSI, 1)
				tests.CreatePVC(tests.CustomISCSI, "1Gi")
			}, 120)

			AfterEach(func() {
				tests.DeletePVC(tests.CustomISCSI)
				tests.DeletePV(tests.CustomISCSI)
			}, 120)

			It("should start vm multiple times", func() {
				checkReadiness()

				vm := tests.NewRandomVMWithPVC(tests.DiskAlpineISCSI)
				tests.AddPVCDisk(vm, "disk1", "virtio", tests.DiskCustomISCSI)
				vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}

				num := 3
				By("Starting and stopping the VM number of times")
				for i := 1; i <= num; i++ {
					obj := RunVMAndExpectLaunch(vm, false, 120)

					// Verify console on last iteration to verify the VM is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the second disk is present")
						expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, 10*time.Second)
						defer expecter.Close()
						Expect(err).To(BeNil())
						_, err = expecter.ExpectBatch([]expect.Batcher{
							&expect.BSnd{S: "\n"},
							&expect.BExp{R: "Welcome to Alpine"},
							&expect.BSnd{S: "root\n"},
							&expect.BExp{R: "#"},
							&expect.BSnd{S: "blockdev --getsize64 /dev/vdb\n"},
							&expect.BExp{R: "1000000000"},
						}, 200*time.Second)
						Expect(err).ToNot(HaveOccurred())
					}

					err = virtClient.VM(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())

					tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)
				}
			})
		})
	})
})
