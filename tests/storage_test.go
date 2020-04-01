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
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
)

const (
	diskSerial = "FB-fb_18030C10002032"
)

type VMICreationFunc func(string) *v1.VirtualMachineInstance

var _ = Describe("Storage", func() {
	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("Starting a VirtualMachineInstance", func() {
		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]with Alpine PVC", func() {
			table.DescribeTable("should be successfully started", func(newVMI VMICreationFunc) {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := newVMI(tests.DiskAlpineHostPath)
				tests.RunVMIAndExpectLaunch(vmi, 90)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()
			},
				table.Entry("[test_id:3130]with Disk PVC", tests.NewRandomVMIWithPVC),
				table.Entry("[test_id:3131]with CDRom PVC", tests.NewRandomVMIWithCDRom),
			)

			table.DescribeTable("should be successfully started and stopped multiple times", func(newVMI VMICreationFunc) {
				vmi := newVMI(tests.DiskAlpineHostPath)

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					vmi := tests.RunVMIAndExpectLaunch(vmi, 90)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VirtualMachineInstance console has expected output")
						expecter, err := tests.LoggedInAlpineExpecter(vmi)
						Expect(err).ToNot(HaveOccurred())
						expecter.Close()
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
			},
				table.Entry("[test_id:3132]with Disk PVC", tests.NewRandomVMIWithPVC),
				table.Entry("[test_id:3133]with CDRom PVC", tests.NewRandomVMIWithCDRom),
			)
		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With an emptyDisk defined", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			It("[test_id:3134]should create a writeable emptyDisk with the right capacity", func() {

				// Start the VirtualMachineInstance with the empty disk attached
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "echo hi!")
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "emptydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "emptydisk1",
					VolumeSource: v1.VolumeSource{
						EmptyDisk: &v1.EmptyDiskSource{
							Capacity: resource.MustParse("2Gi"),
						},
					},
				})
				tests.RunVMIAndExpectLaunch(vmi, 90)

				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking that /dev/vdc has a capacity of 2Gi")
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo blockdev --getsize64 /dev/vdc\n"},
					&expect.BExp{R: "2147483648"}, // 2Gi in bytes
				}, 10*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())

				By("Checking if we can write to /dev/vdc")
				res, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo mkfs.ext4 /dev/vdc\n"},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: "0"},
				}, 20*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
			})

		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With an emptyDisk defined and a specified serial number", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			It("[test_id:3135]should create a writeable emptyDisk with the specified serial number", func() {

				// Start the VirtualMachineInstance with the empty disk attached
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "echo hi!")
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name:   "emptydisk1",
					Serial: diskSerial,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "emptydisk1",
					VolumeSource: v1.VolumeSource{
						EmptyDisk: &v1.EmptyDiskSource{
							Capacity: resource.MustParse("1Gi"),
						},
					},
				})
				tests.RunVMIAndExpectLaunch(vmi, 90)

				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Checking for the specified serial number")
				res, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo find /sys -type f -regex \".*/block/.*/serial\" | xargs cat\n"},
					&expect.BExp{R: diskSerial},
				}, 10*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
			})

		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With ephemeral alpine PVC", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			var isRunOnKindInfra bool
			tests.BeforeAll(func() {
				isRunOnKindInfra = tests.IsRunningOnKindInfra()
			})

			It("[test_id:3136]should be successfully started", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithEphemeralPVC(tests.DiskAlpineHostPath)
				if isRunOnKindInfra {
					tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 90)
				} else {
					tests.RunVMIAndExpectLaunch(vmi, 90)
				}
				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()
			})

			It("[test_id:3137]should not persist data", func() {
				vmi := tests.NewRandomVMIWithEphemeralPVC(tests.DiskAlpineHostPath)

				By("Starting the VirtualMachineInstance")
				var createdVMI *v1.VirtualMachineInstance
				if isRunOnKindInfra {
					createdVMI = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 90)
				} else {
					createdVMI = tests.RunVMIAndExpectLaunch(vmi, 90)
				}

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
				if isRunOnKindInfra {
					createdVMI = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 90)
				} else {
					createdVMI = tests.RunVMIAndExpectLaunch(vmi, 90)
				}

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

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With VirtualMachineInstance with two PVCs", func() {
			BeforeEach(func() {
				// Setup second PVC to use in this context
				tests.CreateHostPathPv(tests.CustomHostPath, tests.HostPathCustom)
				tests.CreateHostPathPVC(tests.CustomHostPath, "1Gi")
			}, 120)

			It("[test_id:3138]should start vmi multiple times", func() {
				vmi := tests.NewRandomVMIWithPVC(tests.DiskAlpineHostPath)
				tests.AddPVCDisk(vmi, "disk1", "virtio", tests.DiskCustomHostPath)

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					obj := tests.RunVMIAndExpectLaunch(vmi, 120)

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
					Expect(err).ToNot(HaveOccurred())

					tests.WaitForVirtualMachineToDisappearWithTimeout(obj, 120)
				}
			})
		})

		Context("[rfe_id:2298][crit:medium][vendor:cnv-qe@redhat.com][level:component] With HostDisk and PVC initialization", func() {

			Context("With a HostDisk defined", func() {

				const hostDiskDir = "/tmp/kubevirt-hostdisks"
				var nodeName string

				BeforeEach(func() {
					nodeName = ""
				})

				AfterEach(func() {
					// Delete all VMIs and wait until they disappear to ensure that no disk is in use and that we can delete the whole folder
					Expect(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("virtualmachineinstances").Do().Error()).ToNot(HaveOccurred())
					Eventually(func() int {
						vmis, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).List(&metav1.ListOptions{})
						Expect(err).ToNot(HaveOccurred())
						return len(vmis.Items)
					}, 120, 1).Should(BeZero())
					if nodeName != "" {
						tests.RemoveHostDiskImage(filepath.Join(hostDiskDir, "/*"), nodeName)
					}
				})

				Context("With 'DiskExistsOrCreate' type", func() {
					diskName := "disk-" + uuid.NewRandom().String() + ".img"
					diskPath := filepath.Join(hostDiskDir, diskName)

					table.DescribeTable("Should create a disk image and start", func(driver string) {
						By("Starting VirtualMachineInstance")
						// do not choose a specific node to run the test
						vmi := tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, "")
						vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = driver

						tests.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if disk.img has been created")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
						nodeName = vmiPod.Spec.NodeName
						output, err := tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("host-disk"), "-name", diskName, "-size", "1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath("host-disk", diskPath)))
					},
						table.Entry("[test_id:851]with virtio driver", "virtio"),
						table.Entry("[test_id:3057]with sata driver", "sata"),
					)

					It("[test_id:3107]should start with multiple hostdisks in the same directory", func() {
						By("Starting VirtualMachineInstance")
						// do not choose a specific node to run the test
						vmi := tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, "")
						tests.AddHostDisk(vmi, filepath.Join(hostDiskDir, "another.img"), v1.HostDiskExistsOrCreate, "anotherdisk")
						tests.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if another.img has been created")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
						nodeName = vmiPod.Spec.NodeName
						output, err := tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("anotherdisk"), "-size", "1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath("anotherdisk", filepath.Join(hostDiskDir, "another.img"))))

						By("Checking if disk.img has been created")
						output, err = tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("host-disk"), "-size", "1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath("host-disk", diskPath)))
					})

				})

				Context("With 'DiskExists' type", func() {
					diskName := "disk-" + uuid.NewRandom().String() + ".img"
					diskPath := filepath.Join(hostDiskDir, diskName)
					// it is mandatory to run a pod which is creating a disk image
					// on the same node with a HostDisk VMI

					BeforeEach(func() {
						// create a disk image before test
						job := tests.CreateHostDiskImage(diskPath)
						job, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
						Expect(err).ToNot(HaveOccurred())
						getStatus := func() k8sv1.PodPhase {
							pod, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Get(job.Name, metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							if pod.Spec.NodeName != "" && nodeName == "" {
								nodeName = pod.Spec.NodeName
							}
							return pod.Status.Phase
						}
						Eventually(getStatus, 30, 1).Should(Equal(k8sv1.PodSucceeded))
					})

					It("[test_id:2306]Should use existing disk image and start", func() {
						By("Starting VirtualMachineInstance")
						vmi := tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExists, nodeName)
						tests.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if disk.img exists")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
						output, err := tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("host-disk"), "-name", diskName},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(diskName))
					})

					It("[test_id:847]Should fail with a capacity option", func() {
						By("Starting VirtualMachineInstance")
						vmi := tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExists, nodeName)
						for i, volume := range vmi.Spec.Volumes {
							if volume.HostDisk != nil {
								vmi.Spec.Volumes[i].HostDisk.Capacity = resource.MustParse("1Gi")
								break
							}
						}
						_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
						Expect(err).To(HaveOccurred())
					})
				})

				Context("With unknown hostDisk type", func() {
					It("[test_id:852]Should fail to start VMI", func() {
						By("Starting VirtualMachineInstance")
						vmi := tests.NewRandomVMIWithHostDisk("/data/unknown.img", "unknown", "")
						_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("With multiple empty PVCs", func() {

				var pvcs = [...]string{"empty-pvc1", "empty-pvc2", "empty-pvc3"}

				BeforeEach(func() {
					for _, pvc := range pvcs {
						tests.CreateHostPathPv(pvc, filepath.Join(tests.HostPathBase, pvc))
						tests.CreateHostPathPVC(pvc, "1G")
					}
				}, 120)

				AfterEach(func() {
					for _, pvc := range pvcs {
						tests.DeletePVC(pvc)
						tests.DeletePV(pvc)
					}
				}, 120)

				It("[test_id:868]Should initialize an empty PVC by creating a disk.img", func() {
					for _, pvc := range pvcs {
						By("starting VirtualMachineInstance")
						vmi := tests.NewRandomVMIWithPVC("disk-" + pvc)
						tests.RunVMIAndExpectLaunch(vmi, 90)

						By("Checking if disk.img exists")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
						output, _ := tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", "/var/run/kubevirt-private/vmi-disks/disk0/", "-name", "disk.img", "-size", "1G"},
						)
						By("Checking if a disk image for PVC has been created")
						Expect(strings.Contains(output, "disk.img")).To(BeTrue())
					}
				})
			})

			Context("With smaller than requested PVCs", func() {

				var mountDir string
				var diskPath string
				var pod *k8sv1.Pod
				var diskSize int

				BeforeEach(func() {

					By("Creating a hostPath pod which prepares a mounted directory which goes away when the pod dies")
					tmpDir := "/tmp/kubevirt/" + rand.String(10)
					mountDir = filepath.Join(tmpDir, "mount")
					diskPath = filepath.Join(mountDir, "disk.img")
					pod = tests.RenderHostPathJob("host-path-preparator", tmpDir, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationBidirectional, []string{"/usr/bin/bash", "-c"}, []string{fmt.Sprintf("mkdir -p %s && mkdir -p /tmp/yyy  && mount --bind /tmp/yyy %s && while true; do sleep 1; done", mountDir, mountDir)})
					pod.Spec.Containers[0].Lifecycle = &k8sv1.Lifecycle{
						PreStop: &k8sv1.Handler{
							Exec: &k8sv1.ExecAction{
								Command: []string{"umount", mountDir},
							},
						},
					}
					pod, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(pod)
					Expect(err).NotTo(HaveOccurred())

					By("Waiting for hostPath pod to prepare the mounted directory")
					Eventually(func() k8sv1.ConditionStatus {
						p, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Get(pod.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						for _, c := range p.Status.Conditions {
							if c.Type == k8sv1.PodReady {
								return c.Status
							}
						}
						return k8sv1.ConditionFalse
					}, 30, 1).Should(Equal(k8sv1.ConditionTrue))

					By("Determining the size of the mounted directory")
					diskSizeStr, _, err := tests.ExecuteCommandOnPodV2(virtClient, pod, pod.Spec.Containers[0].Name, []string{"/usr/bin/bash", "-c", fmt.Sprintf("df %s | tail -n 1 | awk '{print $4}'", mountDir)})
					Expect(err).ToNot(HaveOccurred())
					diskSize, err = strconv.Atoi(strings.TrimSpace(diskSizeStr))
					diskSize = diskSize * 1000 // byte to kilobyte
					Expect(err).ToNot(HaveOccurred())

				})

				configureToleration := func(toleration int) {
					By("By configuring toleration")
					tests.UpdateClusterConfigValueAndWait(virtconfig.LessPVCSpaceTolerationKey, strconv.Itoa(toleration))
				}

				It("[test_id:3108]Should not initialize an empty PVC with a disk.img when disk is too small even with toleration", func() {

					configureToleration(10)

					By("starting VirtualMachineInstance")
					vmi := tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, pod.Spec.NodeName)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					tests.RunVMI(vmi, 30)

					By("Checking events")
					objectEventWatcher := tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(120) * time.Second)
					stopChan := make(chan struct{})
					defer close(stopChan)
					objectEventWatcher.WaitFor(stopChan, tests.WarningEvent, v1.SyncFailed.String())

				})

				It("[test_id:3109]Should initialize an empty PVC with a disk.img when disk is too small but within toleration", func() {

					configureToleration(30)

					By("starting VirtualMachineInstance")
					vmi := tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, pod.Spec.NodeName)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					tests.RunVMIAndExpectLaunch(vmi, 30)

					By("Checking events")
					objectEventWatcher := tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(30) * time.Second)
					objectEventWatcher.FailOnWarnings()
					stopChan := make(chan struct{})
					defer close(stopChan)
					objectEventWatcher.WaitFor(stopChan, tests.EventType(hostdisk.EventTypeToleratedSmallPV), hostdisk.EventReasonToleratedSmallPV)
				})
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component] With Cirros BlockMode PVC", func() {
			BeforeEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.CreateBlockVolumePvAndPvc("1Gi")
			})

			It("[test_id:1015] should be successfully started", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(tests.BlockDiskForTest)
				// Without userdata the hostname isn't set correctly and the login expecter fails...
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				tests.RunVMIAndExpectLaunch(vmi, 90)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "Cirros login successfully")
				expecter.Close()
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component]With Alpine ISCSI PVC", func() {

			pvName := "test-iscsi-lun" + rand.String(48)

			BeforeEach(func() {
				tests.SkipIfVersionAboveOrEqual("re-enable this once https://github.com/kubevirt/kubevirt/issues/2272 is fixed", "1.13.3")
				// Start a ISCSI POD and service
				By("Creating a ISCSI POD")
				iscsiTargetIP := tests.CreateISCSITargetPOD(tests.ContainerDiskAlpine)
				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiTargetIP, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
			})

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			})

			It("[test_id:3139]should be successfully started", func() {
				By("Create a VMIWithPVC")
				// Start the VirtualMachineInstance with the PVC attached
				vmi := tests.NewRandomVMIWithPVC(pvName)
				By("Launching a VMI with PVC ")
				tests.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "Alpine login successfully")
				expecter.Close()
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component] With not existing PVC", func() {
			It("[test_id:1040] should get unschedulable condition", func() {
				// Start the VirtualMachineInstance
				pvcName := "nonExistingPVC"
				vmi := tests.NewRandomVMIWithPVC(pvcName)

				tests.RunVMI(vmi, 10)

				virtClient, err := kubecli.GetKubevirtClient()
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					if vmi.Status.Phase != v1.Pending {
						return false
					}
					if len(vmi.Status.Conditions) == 0 {
						return false
					}

					expectPodScheduledCondition := func(vmi *v1.VirtualMachineInstance) {
						getType := func(c v1.VirtualMachineInstanceCondition) string { return string(c.Type) }
						getReason := func(c v1.VirtualMachineInstanceCondition) string { return c.Reason }
						getStatus := func(c v1.VirtualMachineInstanceCondition) k8sv1.ConditionStatus { return c.Status }
						getMessage := func(c v1.VirtualMachineInstanceCondition) string { return c.Message }
						Expect(vmi.Status.Conditions).To(
							ContainElement(
								And(
									WithTransform(getType, Equal(string(k8sv1.PodScheduled))),
									WithTransform(getReason, Equal(k8sv1.PodReasonUnschedulable)),
									WithTransform(getStatus, Equal(k8sv1.ConditionFalse)),
									WithTransform(getMessage, Equal(fmt.Sprintf("failed to render launch manifest: didn't find PVC %v", pvcName))),
								),
							),
						)
					}
					expectPodScheduledCondition(vmi)
					return true
				}, time.Duration(10)*time.Second).Should(BeTrue(), "Timed out waiting for VMI to get Unschedulable condition")

			})
		})
	})
})
