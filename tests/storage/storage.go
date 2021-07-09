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

package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	storageframework "kubevirt.io/kubevirt/tests/framework/storage"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	virtv1 "kubevirt.io/client-go/api/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	. "kubevirt.io/kubevirt/tests/framework/matcher"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
)

const (
	diskSerial = "FB-fb_18030C10002032"
)

type VMICreationFunc func(string) *v1.VirtualMachineInstance

var _ = SIGDescribe("Storage", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		tests.BeforeTestCleanup()
	})

	Describe("Starting a VirtualMachineInstance", func() {
		var vmi *v1.VirtualMachineInstance
		var targetImagePath string

		BeforeEach(func() {
			vmi = nil
			targetImagePath = tests.HostPathAlpine
		})

		initNFS := func(targetImage string) *k8sv1.Pod {
			// Prepare a NFS backed PV
			By("Starting an NFS POD")
			nfsPod := storageframework.RenderNFSServer("nfsserver", targetImage)
			nfsPod, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(context.Background(), nfsPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisPod(nfsPod), 180).Should(BeInPhase(k8sv1.PodRunning))
			nfsPod, err = ThisPod(nfsPod)()
			Expect(err).ToNot(HaveOccurred())
			return nfsPod
		}

		createNFSPvAndPvc := func(ipFamily k8sv1.IPFamily, nfsPod *k8sv1.Pod) string {
			pvName := fmt.Sprintf("test-nfs%s", rand.String(48))

			// create a new PV and PVC (PVs can't be reused)
			By("create a new NFS PV and PVC")
			nfsIP := libnet.GetPodIpByFamily(nfsPod, ipFamily)
			ExpectWithOffset(1, nfsIP).NotTo(BeEmpty())
			os := string(cd.ContainerDiskAlpine)
			tests.CreateNFSPvAndPvc(pvName, tests.NamespaceTestDefault, "5Gi", nfsIP, os)
			return pvName
		}

		runHostPathJobAndExpectCompletion := func(pod *k8sv1.Pod) {
			pod, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(context.Background(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisPod(pod), 120).Should(BeInPhase(k8sv1.PodSucceeded))
			_, err = ThisPod(pod)()
			Expect(err).ToNot(HaveOccurred())
		}

		copyAlpineWithNonQEMUPermissions := func() string {

			dstPath := tests.HostPathAlpine + "-nopriv"

			hostPathType := k8sv1.HostPathDirectoryOrCreate

			args := []string{fmt.Sprintf(`mkdir -p %[1]s-nopriv && cp %[1]s/disk.img %[1]s-nopriv/ && chmod 644 %[1]s-nopriv/disk.img`, tests.HostPathAlpine)}

			By("creating an image with without qemu permissions")
			pod := tests.RenderHostPathPod("tmp-image-create-job", tests.HostPathBase, hostPathType, k8sv1.MountPropagationNone, []string{"/bin/bash", "-c"}, args)

			runHostPathJobAndExpectCompletion(pod)
			return dstPath
		}

		deleteAlpineWithNonQEMUPermissions := func() {

			dst := tests.HostPathAlpine + "-nopriv"

			hostPathType := k8sv1.HostPathDirectoryOrCreate

			args := []string{fmt.Sprintf(`rm -rf %s`, dst)}

			pod := tests.RenderHostPathPod("remove-tmp-image-job", tests.HostPathBase, hostPathType, k8sv1.MountPropagationNone, []string{"/bin/bash", "-c"}, args)

			runHostPathJobAndExpectCompletion(pod)
		}
		Context("with faulty disk", func() {

			var (
				nodeName   string
				deviceName string = "error"
				pv         *k8sv1.PersistentVolume
				pvc        *k8sv1.PersistentVolumeClaim
			)

			BeforeEach(func() {
				nodeName = tests.NodeNameWithHandler()
				tests.CreateFaultyDisk(nodeName, deviceName)
				var err error
				pv, pvc, err = tests.CreatePVandPVCwithFaultyDisk(nodeName, deviceName, tests.NamespaceTestDefault)
				Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for faulty disk")
			})

			AfterEach(func() {
				tests.RemoveFaultyDisk(nodeName, deviceName)

				err := virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should pause VMI on IO error", func() {
				By("Creating VMI with faulty disk")
				vmi := tests.NewRandomVMIWithPVC(pvc.Name)
				_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil(), "Failed to create vmi")

				tests.WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(vmi, 120)

				refresh := ThisVMI(vmi)
				By("Expecting VMI to be paused")
				Eventually(
					func() bool {
						vmi, err = refresh()
						Expect(err).NotTo(HaveOccurred())

						for _, condition := range vmi.Status.Conditions {
							if condition.Type == v1.VirtualMachineInstancePaused {
								return true
							}
						}
						return false
					}, 60*time.Second, time.Second).Should(BeTrue())

				err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil(), "Failed to delete VMI")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})
		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]with Alpine PVC", func() {

			Context("should be successfully", func() {
				var pvName string
				var nfsPod *k8sv1.Pod
				AfterEach(func() {
					if targetImagePath != tests.HostPathAlpine {
						deleteAlpineWithNonQEMUPermissions()
					}
				})
				table.DescribeTable("started", func(newVMI VMICreationFunc, storageEngine string, family k8sv1.IPFamily, imageOwnedByQEMU bool) {
					if family == k8sv1.IPv6Protocol {
						libnet.SkipWhenNotDualStackCluster(virtClient)
					}

					var ignoreWarnings bool
					// Start the VirtualMachineInstance with the PVC attached
					if storageEngine == "nfs" {
						targetImage := targetImagePath
						if !imageOwnedByQEMU {
							targetImage = copyAlpineWithNonQEMUPermissions()
						}
						nfsPod = initNFS(targetImage)
						pvName = createNFSPvAndPvc(family, nfsPod)
						ignoreWarnings = true
					} else {
						pvName = tests.DiskAlpineHostPath
					}
					vmi = newVMI(pvName)

					tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi, 180, ignoreWarnings)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToAlpine(vmi)).To(Succeed())
				},
					table.Entry("[test_id:3130]with Disk PVC", tests.NewRandomVMIWithPVC, "", nil, true),
					table.Entry("[test_id:3131]with CDRom PVC", tests.NewRandomVMIWithCDRom, "", nil, true),
					table.Entry("[test_id:4618]with NFS Disk PVC using ipv4 address of the NFS pod", tests.NewRandomVMIWithPVC, "nfs", k8sv1.IPv4Protocol, true),
					// Skipping the following 2 tests until https://github.com/kubevirt/kubevirt/issues/4829 is fixed
					table.PEntry("with NFS Disk PVC using ipv6 address of the NFS pod", tests.NewRandomVMIWithPVC, "nfs", k8sv1.IPv6Protocol, true),
					table.PEntry("with NFS Disk PVC using ipv4 address of the NFS pod not owned by qemu", tests.NewRandomVMIWithPVC, "nfs", k8sv1.IPv4Protocol, false),
				)
			})

			table.DescribeTable("should be successfully started and stopped multiple times", func(newVMI VMICreationFunc) {
				vmi = newVMI(tests.DiskAlpineHostPath)

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					vmi := tests.RunVMIAndExpectLaunch(vmi, 90)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VirtualMachineInstance console has expected output")
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
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
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(cd.ContainerDiskFor(cd.ContainerDiskCirros), "echo hi!")
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
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				By("Checking that /dev/vdc has a capacity of 2Gi")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo blockdev --getsize64 /dev/vdc\n"},
					&expect.BExp{R: "2147483648"}, // 2Gi in bytes
				}, 10)).To(Succeed())

				By("Checking if we can write to /dev/vdc")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo mkfs.ext4 /dev/vdc\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 20)).To(Succeed())
			})

		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With an emptyDisk defined and a specified serial number", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			It("[test_id:3135]should create a writeable emptyDisk with the specified serial number", func() {

				// Start the VirtualMachineInstance with the empty disk attached
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "echo hi!")
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
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				By("Checking for the specified serial number")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo find /sys -type f -regex \".*/block/.*/serial\" | xargs cat\n"},
					&expect.BExp{R: diskSerial},
				}, 10)).To(Succeed())
			})

		})
		Context("VirtIO-FS with an empty PVC", func() {

			var pvc = "empty-pvc1"

			BeforeEach(func() {
				tests.CreateHostPathPv(pvc, filepath.Join(tests.HostPathBase, pvc))
				tests.CreateHostPathPVC(pvc, "1G")
			}, 120)

			AfterEach(func() {
				tests.DeletePVC(pvc)
				tests.DeletePV(pvc)
			}, 120)

			It("should be successfully started and virtiofs could be accessed", func() {
				pvcName := fmt.Sprintf("disk-%s", pvc)
				vmi := tests.NewRandomVMIWithPVCFS(pvcName)
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512Mi")
				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				// add userdata for guest agent and mount virtio-fs
				fs := vmi.Spec.Domain.Devices.Filesystems[0]
				virtiofsMountPath := fmt.Sprintf("/mnt/virtiof_%s", fs.Name)
				virtiofsTestFile := fmt.Sprintf("%s/virtiofs_test", virtiofsMountPath)
				mountVirtiofsCommands := fmt.Sprintf(`
                                   mkdir %s
                                   mount -t virtiofs %s %s
                                   touch %s
                           `, virtiofsMountPath, fs.Name, virtiofsMountPath, virtiofsTestFile)
				userData := fmt.Sprintf("%s\n%s", tests.GetFedoraToolsGuestAgentUserData(), mountVirtiofsCommands)
				tests.AddUserData(vmi, "cloud-init", userData)

				vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

				// Wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

				virtioFsFileTestCmd := fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", fs.Name)
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				podVirtioFsFileExist, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", virtioFsFileTestCmd},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))
			})
		})
		Context("Run a VMI with VirtIO-FS and a datavolume", func() {
			var dataVolume *cdiv1.DataVolume
			BeforeEach(func() {
				if !tests.HasCDI() {
					Skip("Skip DataVolume tests when CDI is not present")
				}
				dataVolume = tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
			})
			AfterEach(func() {
				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should be successfully started and virtiofs could be accessed", func() {
				vmi := tests.NewRandomVMIWithFSFromDataVolume(dataVolume.Name)
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				By("Waiting until the DataVolume is ready")
				if tests.HasBindingModeWaitForFirstConsumer() {
					tests.WaitForDataVolumePhaseWFFC(dataVolume.Namespace, dataVolume.Name, 30)
				}
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512Mi")

				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				// add userdata for guest agent and mount virtio-fs
				fs := vmi.Spec.Domain.Devices.Filesystems[0]
				virtiofsMountPath := fmt.Sprintf("/mnt/virtiof_%s", fs.Name)
				virtiofsTestFile := fmt.Sprintf("%s/virtiofs_test", virtiofsMountPath)
				mountVirtiofsCommands := fmt.Sprintf(`
                                       mkdir %s
                                       mount -t virtiofs %s %s
                                       touch %s
                               `, virtiofsMountPath, fs.Name, virtiofsMountPath, virtiofsTestFile)
				userData := fmt.Sprintf("%s\n%s", tests.GetFedoraToolsGuestAgentUserData(), mountVirtiofsCommands)
				tests.AddUserData(vmi, "cloud-init", userData)

				// with WFFC the run actually starts the import and then runs VM, so the timeout has to include both
				// import and start
				vmi = tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)

				// Wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

				By("Checking that virtio-fs is mounted")
				listVirtioFSDisk := fmt.Sprintf("ls -l %s/*disk* | wc -l\n", virtiofsMountPath)
				Expect(console.ExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: listVirtioFSDisk},
					&expect.BExp{R: console.RetValue("1")},
				}, 30*time.Second)).To(Succeed(), "Should be able to access the mounted virtiofs file")

				virtioFsFileTestCmd := fmt.Sprintf("test -f /run/kubevirt-private/vmi-disks/%s/virtiofs_test && echo exist", fs.Name)
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				podVirtioFsFileExist, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"/usr/bin/bash", "-c", virtioFsFileTestCmd},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(strings.Trim(podVirtioFsFileExist, "\n")).To(Equal("exist"))
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

			})
		})
		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With ephemeral alpine PVC", func() {
			var isRunOnKindInfra bool
			tests.BeforeAll(func() {
				isRunOnKindInfra = tests.IsRunningOnKindInfra()
			})

			Context("should be successfully", func() {
				var pvName string
				var nfsPod *k8sv1.Pod

				BeforeEach(func() {
					nfsPod = nil
					pvName = ""
				})

				AfterEach(func() {
					if vmi != nil {
						By("Deleting the VMI")
						Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

						By("Waiting for VMI to disappear")
						tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
					}
				})

				AfterEach(func() {
					if pvName != "" && pvName != tests.DiskAlpineHostPath {
						// PVs can't be reused
						By("Deleting PV and PVC")
						tests.DeletePvAndPvc(pvName)
					}
				})

				// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
				table.DescribeTable("started", func(newVMI VMICreationFunc, storageEngine string, family k8sv1.IPFamily) {
					if family == k8sv1.IPv6Protocol {
						libnet.SkipWhenNotDualStackCluster(virtClient)
					}
					var ignoreWarnings bool
					// Start the VirtualMachineInstance with the PVC attached
					if storageEngine == "nfs" {
						nfsPod = initNFS(tests.HostPathAlpine)
						pvName = createNFSPvAndPvc(family, nfsPod)
						ignoreWarnings = true
					} else {
						pvName = tests.DiskAlpineHostPath
					}
					vmi = newVMI(pvName)
					vmi = tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi, 120, ignoreWarnings)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToAlpine(vmi)).To(Succeed())
				},
					table.Entry("[test_id:3136]with Ephemeral PVC", tests.NewRandomVMIWithEphemeralPVC, "", nil),
					table.Entry("[test_id:4619]with Ephemeral PVC from NFS using ipv4 address of the NFS pod", tests.NewRandomVMIWithEphemeralPVC, "nfs", k8sv1.IPv4Protocol),
					table.Entry("[QUARANTINE]with Ephemeral PVC from NFS using ipv6 address of the NFS pod", tests.NewRandomVMIWithEphemeralPVC, "nfs", k8sv1.IPv6Protocol),
				)
			})

			// Not a candidate for testing on NFS because the VMI is restarted and NFS PVC can't be re-used
			It("[test_id:3137]should not persist data", func() {
				vmi = tests.NewRandomVMIWithEphemeralPVC(tests.DiskAlpineHostPath)

				By("Starting the VirtualMachineInstance")
				var createdVMI *v1.VirtualMachineInstance
				if isRunOnKindInfra {
					createdVMI = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 90)
				} else {
					createdVMI = tests.RunVMIAndExpectLaunch(vmi, 90)
				}

				By("Writing an arbitrary file to it's EFI partition")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// Because "/" is mounted on tmpfs, we need something that normally persists writes - /dev/sda2 is the EFI partition formatted as vFAT.
					&expect.BSnd{S: "mount /dev/sda2 /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "echo content > /mnt/checkpoint\n"},
					&expect.BExp{R: console.PromptExpression},
					// The QEMU process will be killed, therefore the write must be flushed to the disk.
					&expect.BSnd{S: "sync\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 200)).To(Succeed())

				By("Killing a VirtualMachineInstance")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForVirtualMachineToDisappearWithTimeout(createdVMI, 120)

				By("Starting the VirtualMachineInstance again")
				if isRunOnKindInfra {
					tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 90)
				} else {
					tests.RunVMIAndExpectLaunch(vmi, 90)
				}

				By("Making sure that the previously written file is not present")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// Same story as when first starting the VirtualMachineInstance - the checkpoint, if persisted, is located at /dev/sda2.
					&expect.BSnd{S: "mount /dev/sda2 /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/checkpoint &> /dev/null\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("1")},
				}, 200)).To(Succeed())
			})
		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With VirtualMachineInstance with two PVCs", func() {
			BeforeEach(func() {
				// Setup second PVC to use in this context
				tests.CreateHostPathPv(tests.CustomHostPath, tests.HostPathCustom)
				tests.CreateHostPathPVC(tests.CustomHostPath, "1Gi")
			}, 120)

			// Not a candidate for testing on NFS because the VMI is restarted and NFS PVC can't be re-used
			It("[test_id:3138]should start vmi multiple times", func() {
				vmi = tests.NewRandomVMIWithPVC(tests.DiskAlpineHostPath)
				tests.AddPVCDisk(vmi, "disk1", "virtio", tests.DiskCustomHostPath)

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					obj := tests.RunVMIAndExpectLaunch(vmi, 120)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the second disk is present")
						Expect(console.LoginToAlpine(vmi)).To(Succeed())

						Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
							&expect.BSnd{S: "blockdev --getsize64 /dev/vdb\n"},
							&expect.BExp{R: "67108864"},
						}, 200)).To(Succeed())
					}

					err = virtClient.VirtualMachineInstance(obj.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					Eventually(ThisVMI(obj), 120).Should(BeGone())
				}
			})
		})

		Context("[Serial]With feature gates disabled for", func() {
			It("[test_id:4620]HostDisk, it should fail to start a VMI", func() {
				tests.DisableFeatureGate(virtconfig.HostDiskGate)
				vmi = tests.NewRandomVMIWithHostDisk("somepath", v1.HostDiskExistsOrCreate, "")
				virtClient, err := kubecli.GetKubevirtClient()
				Expect(err).ToNot(HaveOccurred())
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("HostDisk feature gate is not enabled"))
			})
			It("VirtioFS, it should fail to start a VMI", func() {
				tests.DisableFeatureGate(virtconfig.VirtIOFSGate)
				vmi := tests.NewRandomVMIWithFSFromDataVolume("something")
				virtClient, err := kubecli.GetKubevirtClient()
				Expect(err).ToNot(HaveOccurred())
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("virtiofs feature gate is not enabled"))
			})
		})

		Context("[rfe_id:2298][crit:medium][vendor:cnv-qe@redhat.com][level:component] With HostDisk and PVC initialization", func() {

			Context("With a HostDisk defined", func() {

				var hostDiskDir string
				var nodeName string

				BeforeEach(func() {
					if !tests.HasFeature(virtconfig.HostDiskGate) {
						Skip("Cluster has the HostDisk featuregate disabled, skipping  the tests")
					}
					hostDiskDir = tests.RandTmpDir()
					nodeName = ""
				})

				AfterEach(func() {
					if vmi != nil {
						err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
						if err != nil && !errors.IsNotFound(err) {
							Expect(err).ToNot(HaveOccurred())
						}
						Eventually(ThisVMI(vmi), 30).Should(Or(BeGone(), BeInPhase(virtv1.Failed), BeInPhase(virtv1.Succeeded)))
					}
					if nodeName != "" {
						tests.RemoveHostDiskImage(hostDiskDir, nodeName)
					}
				})

				Context("With 'DiskExistsOrCreate' type", func() {
					var diskName string
					var diskPath string
					BeforeEach(func() {
						diskName = fmt.Sprintf("disk-%s.img", uuid.NewRandom().String())
						diskPath = filepath.Join(hostDiskDir, diskName)
					})

					table.DescribeTable("Should create a disk image and start", func(driver string) {
						By("Starting VirtualMachineInstance")
						// do not choose a specific node to run the test
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, "")
						vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = driver

						tests.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if disk.img has been created")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
						nodeName = vmiPod.Spec.NodeName
						output, err := tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("host-disk"), "-name", diskName, "-size", "1G", "-o", "-size", "+1G"},
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
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, "")
						tests.AddHostDisk(vmi, filepath.Join(hostDiskDir, "another.img"), v1.HostDiskExistsOrCreate, "anotherdisk")
						tests.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if another.img has been created")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
						nodeName = vmiPod.Spec.NodeName
						output, err := tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("anotherdisk"), "-size", "1G", "-o", "-size", "+1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath("anotherdisk", filepath.Join(hostDiskDir, "another.img"))))

						By("Checking if disk.img has been created")
						output, err = tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("host-disk"), "-size", "1G", "-o", "-size", "+1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath("host-disk", diskPath)))
					})

				})

				Context("With 'DiskExists' type", func() {
					var diskPath string
					var diskName string
					BeforeEach(func() {
						diskName = fmt.Sprintf("disk-%s.img", uuid.NewRandom().String())
						diskPath = filepath.Join(hostDiskDir, diskName)
						// create a disk image before test
						job := tests.CreateHostDiskImage(diskPath)
						job, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(context.Background(), job, metav1.CreateOptions{})
						Expect(err).ToNot(HaveOccurred())
						getStatus := func() k8sv1.PodPhase {
							pod, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Get(context.Background(), job.Name, metav1.GetOptions{})
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
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExists, nodeName)
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
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExists, nodeName)
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
						vmi = tests.NewRandomVMIWithHostDisk("/data/unknown.img", "unknown", "")
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

				// Not a candidate for NFS testing because multiple VMIs are started
				It("[test_id:868]Should initialize an empty PVC by creating a disk.img", func() {
					for _, pvc := range pvcs {
						By("starting VirtualMachineInstance")
						vmi = tests.NewRandomVMIWithPVC(fmt.Sprintf("disk-%s", pvc))
						tests.RunVMIAndExpectLaunch(vmi, 90)

						By("Checking if disk.img exists")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
						output, err := tests.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", "/var/run/kubevirt-private/vmi-disks/disk0/", "-name", "disk.img", "-size", "1G", "-o", "-size", "+1G"},
						)
						Expect(err).ToNot(HaveOccurred())

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
					tmpDir := tests.RandTmpDir()
					mountDir = filepath.Join(tmpDir, "mount")
					diskPath = filepath.Join(mountDir, "disk.img")
					srcDir := filepath.Join(tmpDir, "src")
					cmd := "mkdir -p " + mountDir + " && mkdir -p " + srcDir + " && chcon -t container_file_t " + srcDir + " && mount --bind " + srcDir + " " + mountDir + " && while true; do sleep 1; done"
					pod = tests.RenderHostPathPod("host-path-preparator", tmpDir, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationBidirectional, []string{"/usr/bin/bash", "-c"}, []string{cmd})
					pod.Spec.Containers[0].Lifecycle = &k8sv1.Lifecycle{
						PreStop: &k8sv1.Handler{
							Exec: &k8sv1.ExecAction{
								Command: []string{"umount", mountDir},
							},
						},
					}
					pod, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(context.Background(), pod, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("Waiting for hostPath pod to prepare the mounted directory")
					Eventually(func() k8sv1.ConditionStatus {
						p, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Get(context.Background(), pod.Name, metav1.GetOptions{})
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
					cfg := tests.GetCurrentKv(virtClient).Spec.Configuration
					cfg.DeveloperConfiguration.LessPVCSpaceToleration = toleration
					tests.UpdateKubeVirtConfigValueAndWait(cfg)
				}

				// Not a candidate for NFS test due to usage of host disk
				It("[Serial][test_id:3108]Should not initialize an empty PVC with a disk.img when disk is too small even with toleration", func() {

					configureToleration(10)

					By("starting VirtualMachineInstance")
					vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, pod.Spec.NodeName)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					tests.RunVMI(vmi, 30)

					By("Checking events")
					objectEventWatcher := tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(120) * time.Second)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					objectEventWatcher.WaitFor(ctx, tests.WarningEvent, v1.SyncFailed.String())

				})

				// Not a candidate for NFS test due to usage of host disk
				It("[Serial][test_id:3109]Should initialize an empty PVC with a disk.img when disk is too small but within toleration", func() {

					configureToleration(30)

					By("starting VirtualMachineInstance")
					vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, pod.Spec.NodeName)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					tests.RunVMIAndExpectLaunch(vmi, 30)

					By("Checking events")
					objectEventWatcher := tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(30) * time.Second)
					wp := tests.WarningsPolicy{FailOnWarnings: true}
					objectEventWatcher.SetWarningsPolicy(wp)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					objectEventWatcher.WaitFor(ctx, tests.EventType(hostdisk.EventTypeToleratedSmallPV), hostdisk.EventReasonToleratedSmallPV)
				})
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component] With Cirros BlockMode PVC", func() {
			BeforeEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.CreateBlockVolumePvAndPvc("1Gi")
			})

			// Not a candidate for NFS because local volumes are used in test
			It("[test_id:1015]should be successfully started", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi = tests.NewRandomVMIWithPVC(tests.BlockDiskForTest)
				// Without userdata the hostname isn't set correctly and the login expecter fails...
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component]With Alpine ISCSI PVC (using ISCSI IPv4 address)", func() {

			pvName := fmt.Sprintf("test-iscsi-lun%s", rand.String(48))

			BeforeEach(func() {
				// Start a ISCSI POD and service
				By("Creating a ISCSI POD")
				iscsiTargetPod := tests.CreateISCSITargetPOD(cd.ContainerDiskAlpine)
				iscsiTargetIPAddress := libnet.GetPodIpByFamily(iscsiTargetPod, k8sv1.IPv4Protocol)
				Expect(iscsiTargetIPAddress).NotTo(BeEmpty())

				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiTargetIPAddress, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
			})

			AfterEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				tests.DeletePvAndPvc(pvName)
			})

			// Not a candidate for NFS because these tests exercise ISCSI
			It("[test_id:3139]should be successfully started", func() {
				By("Create a VMIWithPVC")
				// Start the VirtualMachineInstance with the PVC attached
				vmi = tests.NewRandomVMIWithPVC(pvName)
				By("Launching a VMI with PVC ")
				tests.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component] With not existing PVC", func() {
			// Not a candidate for NFS because the PVC in question doesn't actually exist
			It("[test_id:1040] should get unschedulable condition", func() {
				// Start the VirtualMachineInstance
				pvcName := "nonExistingPVC"
				vmi = tests.NewRandomVMIWithPVC(pvcName)

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

		Context("With both SCSI and SATA devices", func() {
			It("should successfully start with distinct device names", func() {

				// Start the VirtualMachineInstance with two empty disks attached, one per bus
				vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "emptydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "scsi",
						},
					},
				})
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "emptydisk2",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "sata",
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
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "emptydisk2",
					VolumeSource: v1.VolumeSource{
						EmptyDisk: &v1.EmptyDiskSource{
							Capacity: resource.MustParse("1Gi"),
						},
					},
				})
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking that /dev/sda has a capacity of 1Gi")
				Expect(console.ExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "blockdev --getsize64 /dev/sda\n"},
					&expect.BExp{R: "1073741824"}, // 2Gi in bytes
				}, 10*time.Second)).To(Succeed())

				By("Checking that /dev/sdb has a capacity of 1Gi")
				Expect(console.ExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "blockdev --getsize64 /dev/sdb\n"},
					&expect.BExp{R: "1073741824"}, // 1Gi in bytes
				}, 10*time.Second)).To(Succeed())
			})

		})
	})
})
