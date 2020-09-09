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

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
)

const (
	diskSerial = "FB-fb_18030C10002032"
)

type VMICreationFunc func(string) *v1.VirtualMachineInstance

var _ = Describe("[Serial]Storage", func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var originalKVConfiguration v1.KubeVirtConfiguration

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		kv := tests.GetCurrentKv(virtClient)
		originalKVConfiguration = kv.Spec.Configuration
	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("Starting a VirtualMachineInstance", func() {
		var _pvName string
		var vmi *v1.VirtualMachineInstance
		var nfsInitialized bool

		initNFS := func(isIpv6 bool) string {
			if !nfsInitialized {
				_pvName = "test-nfs" + rand.String(48)
				// Prepare a NFS backed PV
				By("Starting an NFS POD")
				os := string(cd.ContainerDiskAlpine)
				nfsIPs := tests.CreateNFSTargetPOD(os)
				// create a new PV and PVC (PVs can't be reused)
				By("create a new NFS PV and PVC")
				nfsIP := libnet.GetIp(libnet.GetPodIpsStrings(nfsIPs), isIpv6)
				if nfsIP == "" && isIpv6 {
					Skip("skipping ipv6 nfs test on a single stack cluster")
				}
				Expect(nfsIP).NotTo(BeEmpty())
				tests.CreateNFSPvAndPvc(_pvName, "5Gi", nfsIP, os)

				nfsInitialized = true
			}
			return _pvName
		}

		BeforeEach(func() {
			nfsInitialized = false
		})

		AfterEach(func() {
			if nfsInitialized {
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for VMI to disappear")
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				// PVs can't be reused
				tests.DeletePvAndPvc(_pvName)

				By("Deleting NFS pod")
				Expect(virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Delete(tests.NFSTargetName, &metav1.DeleteOptions{})).To(Succeed())
				By("Waiting for NFS pod to disappear")
				tests.WaitForPodToDisappearWithTimeout(tests.NFSTargetName, 120)
			}
		})
		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]with Alpine PVC", func() {
			table.DescribeTable("should be successfully started", func(newVMI VMICreationFunc, storageEngine string, isIpv6 bool) {
				tests.SkipPVCTestIfRunnigOnKindInfra()

				var ignoreWarnings bool
				var pvName string
				// Start the VirtualMachineInstance with the PVC attached
				if storageEngine == "nfs" {
					pvName = initNFS(isIpv6)
					ignoreWarnings = true
				} else {
					pvName = tests.DiskAlpineHostPath
				}
				vmi = newVMI(pvName)
				tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi, 120, ignoreWarnings)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()
			},
				table.Entry("[test_id:3130]with Disk PVC", tests.NewRandomVMIWithPVC, "", false),
				table.Entry("[test_id:3131]with CDRom PVC", tests.NewRandomVMIWithCDRom, "", false),
				table.Entry("[test_id:4618]with NFS Disk PVC using ipv4 address of the NFS pod", tests.NewRandomVMIWithPVC, "nfs", false),
				table.Entry("with NFS Disk PVC using ipv6 address of the NFS pod", tests.NewRandomVMIWithPVC, "nfs", true),
			)

			table.DescribeTable("should be successfully started and stopped multiple times", func(newVMI VMICreationFunc) {
				tests.SkipPVCTestIfRunnigOnKindInfra()

				vmi = newVMI(tests.DiskAlpineHostPath)

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
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "echo hi!")
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
					&expect.BExp{R: tests.RetValue("0")},
				}, 20*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
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
		Context("Run a VMI wiht VirtIO-FS and a datavolume", func() {
			var dataVolume *cdiv1.DataVolume
			BeforeEach(func() {
				if !tests.HasCDI() {
					Skip("Skip DataVolume tests when CDI is not present")
				}
				dataVolume = tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				tests.EnableFeatureGate(virtconfig.VirtIOFSGate)
			})

			AfterEach(func() {
				err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(dataVolume.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())
				tests.DisableFeatureGate(virtconfig.VirtIOFSGate)
			})
			It("should be successfully started and viriofs could be accessed", func() {
				tests.SkipPVCTestIfRunnigOnKindInfra()
				waitForDataVolume := func(dataVolume *cdiv1.DataVolume) {
					Eventually(func() cdiv1.DataVolumePhase {
						dv, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Get(dataVolume.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())

						return dv.Status.Phase
					}, 160*time.Second, 1*time.Second).
						Should(Equal(cdiv1.Succeeded), "Timed out waiting for DataVolume to complete")
				}

				vmi := tests.NewRandomVMIWithFSFromDataVolume(dataVolume.Name)
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(dataVolume)
				Expect(err).To(BeNil())
				waitForDataVolume(dataVolume)
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
				userData := fmt.Sprintf("%s\n%s", tests.GetGuestAgentUserData(), mountVirtiofsCommands)
				tests.AddUserData(vmi, "cloud-init", userData)

				vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				// Wait for cloud init to finish and start the agent inside the vmi.
				tests.WaitAgentConnected(virtClient, vmi)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInFedoraExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should be able to login to the Fedora VM")
				defer expecter.Close()

				By("Checking that virtio-fs is mounted")
				listVirtioFSDisk := fmt.Sprintf("ls -l %s/*disk* | wc -l\n", virtiofsMountPath)
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: listVirtioFSDisk},
					&expect.BExp{R: tests.RetValue("1")},
				}, 30*time.Second)
				Expect(err).ToNot(HaveOccurred(), "Should be able to access the mounted virtiofs file")

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
		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With ephemeral alpine PVC", func() {
			var isRunOnKindInfra bool
			tests.BeforeAll(func() {
				isRunOnKindInfra = tests.IsRunningOnKindInfra()
			})

			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			table.DescribeTable("should be successfully started", func(newVMI VMICreationFunc, storageEngine string, isIpv6 bool) {
				tests.SkipPVCTestIfRunnigOnKindInfra()
				var ignoreWarnings bool
				var pvName string
				// Start the VirtualMachineInstance with the PVC attached
				if storageEngine == "nfs" {
					pvName = initNFS(isIpv6)
					ignoreWarnings = true
				} else {
					pvName = tests.DiskAlpineHostPath
				}
				vmi = newVMI(pvName)
				vmi = tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi, 120, ignoreWarnings)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()
			},
				table.Entry("[test_id:3136]with Ephemeral PVC", tests.NewRandomVMIWithEphemeralPVC, "", false),
				table.Entry("[test_id:4619]with Ephemeral PVC from NFS using ipv4 address of the NFS pod", tests.NewRandomVMIWithEphemeralPVC, "nfs", false),
				table.Entry("with Ephemeral PVC from NFS using ipv6 address of the NFS pod", tests.NewRandomVMIWithEphemeralPVC, "nfs", true),
			)

			// Not a candidate for testing on NFS because the VMI is restarted and NFS PVC can't be re-used
			It("[test_id:3137]should not persist data", func() {
				tests.SkipPVCTestIfRunnigOnKindInfra()

				vmi = tests.NewRandomVMIWithEphemeralPVC(tests.DiskAlpineHostPath)

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
					&expect.BExp{R: tests.RetValue("0")},
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
					&expect.BExp{R: tests.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/checkpoint &> /dev/null\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("1")},
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

			// Not a candidate for testing on NFS because the VMI is restarted and NFS PVC can't be re-used
			It("[test_id:3138]should start vmi multiple times", func() {
				tests.SkipPVCTestIfRunnigOnKindInfra()

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

				hostDiskDir := tests.RandTmpDir()
				var nodeName string
				var currentKvConfiguration v1.KubeVirtConfiguration

				BeforeEach(func() {
					nodeName = ""
					kv := tests.GetCurrentKv(virtClient)
					currentKvConfiguration = kv.Spec.Configuration

					tests.EnableFeatureGate(virtconfig.HostDiskGate)
				})

				AfterEach(func() {
					currentKvConfiguration.DeveloperConfiguration.FeatureGates = originalKVConfiguration.DeveloperConfiguration.FeatureGates
					kv := tests.UpdateKubeVirtConfigValueAndWait(currentKvConfiguration)
					currentKvConfiguration = kv.Spec.Configuration

					// Delete all VMIs and wait until they disappear to ensure that no disk is in use and that we can delete the whole folder
					Expect(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("virtualmachineinstances").Do().Error()).ToNot(HaveOccurred())
					Eventually(func() int {
						vmis, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).List(&metav1.ListOptions{})
						Expect(err).ToNot(HaveOccurred())
						return len(vmis.Items)
					}, 120, 1).Should(BeZero())
					if nodeName != "" {
						tests.RemoveHostDiskImage(hostDiskDir, nodeName)
					}
				})

				Context("Without the HostDisk feature gate enabled", func() {
					BeforeEach(func() {
						tests.DisableFeatureGate(virtconfig.HostDiskGate)
					})

					It("[test_id:4620]Should fail to start a VMI", func() {
						diskName := "disk-" + uuid.NewRandom().String() + ".img"
						diskPath := filepath.Join(hostDiskDir, diskName)
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, "")
						virtClient, err := kubecli.GetKubevirtClient()
						Expect(err).ToNot(HaveOccurred())
						_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("HostDisk feature gate is not enabled"))
					})
				})

				Context("With 'DiskExistsOrCreate' type", func() {
					diskName := "disk-" + uuid.NewRandom().String() + ".img"
					diskPath := filepath.Join(hostDiskDir, diskName)

					// Not a candidate for NFS testing due to usage of host disk
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

					// Not a candidate for NFS testing due to usage of host disk
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

					// Not a candidate for NFS testing due to usage of host disk
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

					// Not a candidate for NFS testing due to usage of host disk
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
					// Not a candidate for NFS testing due to usage of host disk
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
						vmi = tests.NewRandomVMIWithPVC("disk-" + pvc)
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
				var currentKvConfiguration v1.KubeVirtConfiguration

				BeforeEach(func() {
					By("Enabling the HostDisk feature gate")
					kv := tests.EnableFeatureGate(virtconfig.HostDiskGate)
					currentKvConfiguration = kv.Spec.Configuration

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

				AfterEach(func() {
					config := originalKVConfiguration.DeepCopy()
					currentKvConfiguration.DeveloperConfiguration.FeatureGates = config.DeveloperConfiguration.FeatureGates
					kv := tests.UpdateKubeVirtConfigValueAndWait(currentKvConfiguration)
					currentKvConfiguration = kv.Spec.Configuration
				})

				configureToleration := func(toleration int) {
					By("By configuring toleration")
					currentKvConfiguration.DeveloperConfiguration.LessPVCSpaceToleration = toleration
					kv := tests.UpdateKubeVirtConfigValueAndWait(currentKvConfiguration)
					currentKvConfiguration = kv.Spec.Configuration
				}

				// Not a candidate for NFS test due to usage of host disk
				It("[test_id:3108]Should not initialize an empty PVC with a disk.img when disk is too small even with toleration", func() {

					configureToleration(10)

					By("starting VirtualMachineInstance")
					vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, pod.Spec.NodeName)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					tests.RunVMI(vmi, 30)

					By("Checking events")
					objectEventWatcher := tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(120) * time.Second)
					stopChan := make(chan struct{})
					defer close(stopChan)
					objectEventWatcher.WaitFor(stopChan, tests.WarningEvent, v1.SyncFailed.String())

				})

				// Not a candidate for NFS test due to usage of host disk
				It("[test_id:3109]Should initialize an empty PVC with a disk.img when disk is too small but within toleration", func() {

					configureToleration(30)

					By("starting VirtualMachineInstance")
					vmi = tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, pod.Spec.NodeName)
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

			// Not a candidate for NFS because local volumes are used in test
			It("[test_id:1015] should be successfully started", func() {
				tests.SkipPVCTestIfRunnigOnKindInfra()
				// Start the VirtualMachineInstance with the PVC attached
				vmi = tests.NewRandomVMIWithPVC(tests.BlockDiskForTest)
				// Without userdata the hostname isn't set correctly and the login expecter fails...
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")

				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				By("Checking that the VirtualMachineInstance console has expected output")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "Cirros login successfully")
				expecter.Close()
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component]With Alpine ISCSI PVC", func() {

			pvName := "test-iscsi-lun" + rand.String(48)

			BeforeEach(func() {
				if tests.IsIPv6Cluster(virtClient) {
					Skip("Skip ISCSI on IPv6")
				}
				// Start a ISCSI POD and service
				By("Creating a ISCSI POD")
				iscsiTargetIP := tests.CreateISCSITargetPOD(cd.ContainerDiskAlpine)
				tests.CreateISCSIPvAndPvc(pvName, "1Gi", iscsiTargetIP, k8sv1.ReadWriteMany, k8sv1.PersistentVolumeBlock)
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
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "Alpine login successfully")
				expecter.Close()
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
	})
})
