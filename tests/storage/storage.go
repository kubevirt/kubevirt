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

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/libvmi"

	"k8s.io/apimachinery/pkg/api/errors"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	storageframework "kubevirt.io/kubevirt/tests/framework/storage"

	"kubevirt.io/kubevirt/tests/util"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

const (
	failedCreateVMI              = "Failed to create vmi"
	failedDeleteVMI              = "Failed to delete VMI"
	checkingVMInstanceConsoleOut = "Checking that the VirtualMachineInstance console has expected output"
	startingVMInstance           = "Starting VirtualMachineInstance"
	hostDiskName                 = "host-disk"
	diskImgName                  = "disk.img"

	// Without cloud init user data Cirros takes long time to boot,
	// so provide this dummy data to make it boot faster
	cirrosUserData = "#!/bin/bash\necho 'hello'\n"
)

const (
	diskSerial = "FB-fb_18030C10002032"
)

type VMICreationFunc func(string) *virtv1.VirtualMachineInstance

var _ = SIGDescribe("Storage", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		tests.SetupAlpineHostPath()
	})

	Describe("Starting a VirtualMachineInstance", func() {
		var vmi *virtv1.VirtualMachineInstance
		var targetImagePath string

		BeforeEach(func() {
			vmi = nil
			targetImagePath = testsuite.HostPathAlpine
		})

		isPausedOnIOError := func(conditions []v1.VirtualMachineInstanceCondition) bool {
			for _, condition := range conditions {
				if condition.Type == virtv1.VirtualMachineInstancePaused {
					return condition.Status == k8sv1.ConditionTrue && condition.Reason == "PausedIOError"
				}
			}
			return false
		}

		createNFSPvAndPvc := func(ipFamily k8sv1.IPFamily, nfsPod *k8sv1.Pod) string {
			pvName := fmt.Sprintf("test-nfs%s", rand.String(48))

			// create a new PV and PVC (PVs can't be reused)
			By("create a new NFS PV and PVC")
			nfsIP := libnet.GetPodIPByFamily(nfsPod, ipFamily)
			ExpectWithOffset(1, nfsIP).NotTo(BeEmpty())
			os := string(cd.ContainerDiskAlpine)
			libstorage.CreateNFSPvAndPvc(pvName, testsuite.GetTestNamespace(nil), "1Gi", nfsIP, os)
			return pvName
		}

		setShareable := func(vmi *virtv1.VirtualMachineInstance, diskName string) {
			shareable := true
			for i, d := range vmi.Spec.Domain.Devices.Disks {
				if d.Name == diskName {
					vmi.Spec.Domain.Devices.Disks[i].Shareable = &shareable
					return
				}
			}
		}
		Context("[Serial]with error disk", Serial, func() {
			var (
				nodeName, address, device string

				pvc *k8sv1.PersistentVolumeClaim
				pv  *k8sv1.PersistentVolume
			)

			cleanUp := func(vmi *virtv1.VirtualMachineInstance) {
				By("Cleaning up")
				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
			}

			BeforeEach(func() {
				nodeName = tests.NodeNameWithHandler()
				address, device = tests.CreateErrorDisk(nodeName)
				var err error
				pv, pvc, err = tests.CreatePVandPVCwithFaultyDisk(nodeName, device, testsuite.GetTestNamespace(nil))
				Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for faulty disk")
			})

			AfterEach(func() {
				// In order to remove the scsi debug module, the SCSI device cannot be in used by the VM.
				// For this reason, we manually clean-up the VM  before removing the kernel module.
				tests.RemoveSCSIDisk(nodeName, address)
				Expect(virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			})

			It("should pause VMI on IO error", func() {
				By("Creating VMI with faulty disk")
				vmi := libvmi.NewAlpine(libvmi.WithPersistentVolumeClaim("pvc-disk", pvc.Name))
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), failedCreateVMI)

				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				)

				By("Reading from disk")
				Expect(console.LoginToAlpine(vmi)).To(Succeed(), "Should login")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "nohup sh -c \"sleep 10 && while true; do dd if=/dev/vdb of=/dev/null >/dev/null 2>/dev/null; done\" & \n"},
					&expect.BExp{R: console.PromptExpression},
				}, 20)).To(Succeed())

				refresh := ThisVMI(vmi)
				By("Expecting VMI to be paused")
				Eventually(func() []v1.VirtualMachineInstanceCondition {
					vmi, err := refresh()
					Expect(err).ToNot(HaveOccurred())

					return vmi.Status.Conditions
				}, 100*time.Second, time.Second).Should(Satisfy(isPausedOnIOError))

				By("Fixing the device")
				tests.FixErrorDevice(nodeName)

				By("Expecting VMI to NOT be paused")
				Eventually(ThisVMI(vmi), 100*time.Second, time.Second).Should(HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

				cleanUp(vmi)

			})

			It("should report IO errors in the guest with errorPolicy set to report", func() {
				const diskName = "disk1"
				By("Creating VMI with faulty disk")
				vmi := libvmi.NewAlpine(libvmi.WithPersistentVolumeClaim(diskName, pvc.Name))
				for i, d := range vmi.Spec.Domain.Devices.Disks {
					if d.Name == diskName {
						vmi.Spec.Domain.Devices.Disks[i].ErrorPolicy = pointer.P(v1.DiskErrorPolicyReport)
					}
				}

				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), failedCreateVMI)

				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				)

				By("Writing to disk")
				Expect(console.LoginToAlpine(vmi)).To(Succeed(), "Should login")
				tests.CheckResultShellCommandOnVmi(vmi, "dd if=/dev/zero of=/dev/vdb",
					"dd: error writing '/dev/vdb': I/O error", 20)

				cleanUp(vmi)
			})

		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]with Alpine PVC", func() {
			newRandomVMIWithPVC := func(claimName string) *virtv1.VirtualMachineInstance {
				return libvmi.New(
					libvmi.WithPersistentVolumeClaim("disk0", claimName),
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithRng())
			}
			newRandomVMIWithCDRom := func(claimName string) *virtv1.VirtualMachineInstance {
				return libvmi.New(
					libvmi.WithCDRom("disk0", v1.DiskBusSATA, claimName),
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithRng())
			}

			Context("should be successfully", func() {
				var pvName string
				var nfsPod *k8sv1.Pod
				AfterEach(func() {
					// Ensure VMI is deleted before bringing down the NFS server
					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
					libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

					if targetImagePath != testsuite.HostPathAlpine {
						tests.DeleteAlpineWithNonQEMUPermissions()
					}
				})
				DescribeTable("started", func(newVMI VMICreationFunc, storageEngine string, family k8sv1.IPFamily, imageOwnedByQEMU bool) {
					libnet.SkipWhenClusterNotSupportIPFamily(family)

					var nodeName string
					// Start the VirtualMachineInstance with the PVC attached
					if storageEngine == "nfs" {
						if !imageOwnedByQEMU {
							targetImagePath, nodeName = tests.CopyAlpineWithNonQEMUPermissions()
						}
						nfsPod = storageframework.InitNFS(targetImagePath, nodeName)
						pvName = createNFSPvAndPvc(family, nfsPod)
					} else {
						pvName = tests.DiskAlpineHostPath
					}
					vmi = newVMI(pvName)

					if storageEngine == "nfs" {
						vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
					} else {
						vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
					}

					By(checkingVMInstanceConsoleOut)
					Expect(console.LoginToAlpine(vmi)).To(Succeed())
				},
					Entry("[test_id:3130]with Disk PVC", newRandomVMIWithPVC, "", nil, true),
					Entry("[test_id:3131]with CDRom PVC", newRandomVMIWithCDRom, "", nil, true),
					Entry("[test_id:4618]with NFS Disk PVC using ipv4 address of the NFS pod", newRandomVMIWithPVC, "nfs", k8sv1.IPv4Protocol, true),
					Entry("[Serial]with NFS Disk PVC using ipv6 address of the NFS pod", Serial, newRandomVMIWithPVC, "nfs", k8sv1.IPv6Protocol, true),
					Entry("[Serial]with NFS Disk PVC using ipv4 address of the NFS pod not owned by qemu", Serial, newRandomVMIWithPVC, "nfs", k8sv1.IPv4Protocol, false),
				)
			})

			DescribeTable("should be successfully started and stopped multiple times", func(newVMI VMICreationFunc) {
				vmi = newVMI(tests.DiskAlpineHostPath)

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					vmi := tests.RunVMIAndExpectLaunch(vmi, 90)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By(checkingVMInstanceConsoleOut)
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
			},
				Entry("[test_id:3132]with Disk PVC", newRandomVMIWithPVC),
				Entry("[test_id:3133]with CDRom PVC", newRandomVMIWithCDRom),
			)
		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With an emptyDisk defined", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			It("[test_id:3134]should create a writeable emptyDisk with the right capacity", func() {

				// Start the VirtualMachineInstance with the empty disk attached
				vmi = libvmi.NewCirros(
					libvmi.WithResourceMemory("512M"),
					libvmi.WithEmptyDisk("emptydisk1", v1.DiskBusVirtio, resource.MustParse("1G")),
				)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Checking that /dev/vdc has a capacity of 1G, aligned to 4k")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo blockdev --getsize64 /dev/vdc\n"},
					&expect.BExp{R: "999292928"}, // 1G in bytes rounded down to nearest 1MiB boundary
				}, 10)).To(Succeed())

				By("Checking if we can write to /dev/vdc")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo mkfs.ext4 -F /dev/vdc\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: tests.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
				}, 20)).To(Succeed())
			})

		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With an emptyDisk defined and a specified serial number", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			It("[test_id:3135]should create a writeable emptyDisk with the specified serial number", func() {

				// Start the VirtualMachineInstance with the empty disk attached
				vmi = libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, virtv1.Disk{
					Name:   "emptydisk1",
					Serial: diskSerial,
					DiskDevice: virtv1.DiskDevice{
						Disk: &virtv1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
					Name: "emptydisk1",
					VolumeSource: virtv1.VolumeSource{
						EmptyDisk: &virtv1.EmptyDiskSource{
							Capacity: resource.MustParse("1Gi"),
						},
					},
				})
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking for the specified serial number")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "find /sys -type f -regex \".*/block/.*/serial\" | xargs cat\n"},
					&expect.BExp{R: diskSerial},
				}, 10)).To(Succeed())
			})

		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With ephemeral alpine PVC", func() {
			var isRunOnKindInfra bool
			BeforeEach(func() {
				isRunOnKindInfra = checks.IsRunningOnKindInfra()
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
						Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())

						By("Waiting for VMI to disappear")
						libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
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
				DescribeTable("started", func(newVMI VMICreationFunc, storageEngine string, family k8sv1.IPFamily) {
					libnet.SkipWhenClusterNotSupportIPFamily(family)

					// Start the VirtualMachineInstance with the PVC attached
					if storageEngine == "nfs" {
						nfsPod = storageframework.InitNFS(testsuite.HostPathAlpine, "")
						pvName = createNFSPvAndPvc(family, nfsPod)
					} else {
						pvName = tests.DiskAlpineHostPath
					}
					vmi = newVMI(pvName)

					if storageEngine == "nfs" {
						vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 120)
					} else {
						vmi = tests.RunVMIAndExpectLaunch(vmi, 120)
					}

					By(checkingVMInstanceConsoleOut)
					Expect(console.LoginToAlpine(vmi)).To(Succeed())
				},
					Entry("[test_id:3136]with Ephemeral PVC", tests.NewRandomVMIWithEphemeralPVC, "", nil),
					Entry("[test_id:4619]with Ephemeral PVC from NFS using ipv4 address of the NFS pod", tests.NewRandomVMIWithEphemeralPVC, "nfs", k8sv1.IPv4Protocol),
					Entry("with Ephemeral PVC from NFS using ipv6 address of the NFS pod", tests.NewRandomVMIWithEphemeralPVC, "nfs", k8sv1.IPv6Protocol),
				)
			})

			// Not a candidate for testing on NFS because the VMI is restarted and NFS PVC can't be re-used
			It("[test_id:3137]should not persist data", func() {
				vmi = tests.NewRandomVMIWithEphemeralPVC(tests.DiskAlpineHostPath)

				By("Starting the VirtualMachineInstance")
				var createdVMI *virtv1.VirtualMachineInstance
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
					&expect.BSnd{S: tests.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "echo content > /mnt/checkpoint\n"},
					&expect.BExp{R: console.PromptExpression},
					// The QEMU process will be killed, therefore the write must be flushed to the disk.
					&expect.BSnd{S: "sync\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 200)).To(Succeed())

				By("Killing a VirtualMachineInstance")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForVirtualMachineToDisappearWithTimeout(createdVMI, 120)

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
					&expect.BSnd{S: tests.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/checkpoint &> /dev/null\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: tests.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("1")},
				}, 200)).To(Succeed())
			})
		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With VirtualMachineInstance with two PVCs", func() {
			BeforeEach(func() {
				// Setup second PVC to use in this context
				libstorage.CreateHostPathPv(tests.CustomHostPath, testsuite.GetTestNamespace(nil), testsuite.HostPathCustom)
				libstorage.CreateHostPathPVC(tests.CustomHostPath, testsuite.GetTestNamespace(nil), "1Gi")
			})

			// Not a candidate for testing on NFS because the VMI is restarted and NFS PVC can't be re-used
			It("[test_id:3138]should start vmi multiple times", func() {
				vmi = libvmi.New(
					libvmi.WithPersistentVolumeClaim("disk0", tests.DiskAlpineHostPath),
					libvmi.WithPersistentVolumeClaim("disk1", tests.DiskCustomHostPath),
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithRng())

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					obj := tests.RunVMIAndExpectLaunch(vmi, 240)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the second disk is present")
						Expect(console.LoginToAlpine(obj)).To(Succeed())

						Expect(console.SafeExpectBatch(obj, []expect.Batcher{
							&expect.BSnd{S: "blockdev --getsize64 /dev/vdb\n"},
							&expect.BExp{R: "1013972992"},
						}, 200)).To(Succeed())
					}

					err = virtClient.VirtualMachineInstance(obj.Namespace).Delete(context.Background(), obj.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					Eventually(ThisVMI(obj), 120).Should(BeGone())
				}
			})
		})

		Context("[Serial]With feature gates disabled for", Serial, func() {
			It("[test_id:4620]HostDisk, it should fail to start a VMI", func() {
				tests.DisableFeatureGate(virtconfig.HostDiskGate)
				vmi = tests.NewRandomVMIWithHostDisk("somepath", virtv1.HostDiskExistsOrCreate, "")
				virtClient := kubevirt.Client()
				_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("HostDisk feature gate is not enabled"))
			})
		})

		Context("[rfe_id:2298][crit:medium][vendor:cnv-qe@redhat.com][level:component] With HostDisk and PVC initialization", func() {

			BeforeEach(func() {
				if !checks.HasFeature(virtconfig.HostDiskGate) {
					Skip("Cluster has the HostDisk featuregate disabled, skipping  the tests")
				}
			})

			Context("With a HostDisk defined", func() {

				var hostDiskDir string
				var nodeName string

				BeforeEach(func() {
					hostDiskDir = tests.RandTmpDir()
					nodeName = ""
				})

				AfterEach(func() {
					if vmi != nil {
						err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
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

					DescribeTable("Should create a disk image and start", func(driver v1.DiskBus) {
						By(startingVMInstance)
						// do not choose a specific node to run the test
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, virtv1.HostDiskExistsOrCreate, "")
						vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = driver

						tests.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if disk.img has been created")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
						nodeName = vmiPod.Spec.NodeName
						output, err := exec.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir(hostDiskName), "-name", diskName, "-size", "1G", "-o", "-size", "+1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath(hostDiskName, diskPath)))
					},
						Entry("[test_id:851]with virtio driver", v1.DiskBusVirtio),
						Entry("[test_id:3057]with sata driver", v1.DiskBusSATA),
					)

					It("[test_id:3107]should start with multiple hostdisks in the same directory", func() {
						By(startingVMInstance)
						// do not choose a specific node to run the test
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, virtv1.HostDiskExistsOrCreate, "")
						tests.AddHostDisk(vmi, filepath.Join(hostDiskDir, "another.img"), virtv1.HostDiskExistsOrCreate, "anotherdisk")
						tests.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if another.img has been created")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
						nodeName = vmiPod.Spec.NodeName
						output, err := exec.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("anotherdisk"), "-size", "1G", "-o", "-size", "+1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath("anotherdisk", filepath.Join(hostDiskDir, "another.img"))))

						By("Checking if disk.img has been created")
						output, err = exec.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir(hostDiskName), "-size", "1G", "-o", "-size", "+1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath(hostDiskName, diskPath)))
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
						job, err = virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), job, metav1.CreateOptions{})
						Expect(err).ToNot(HaveOccurred())

						Eventually(ThisPod(job), 30*time.Second, 1*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
						pod, err := ThisPod(job)()
						Expect(err).NotTo(HaveOccurred())
						nodeName = pod.Spec.NodeName
					})

					It("[test_id:2306]Should use existing disk image and start", func() {
						By(startingVMInstance)
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, virtv1.HostDiskExists, nodeName)
						tests.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if disk.img exists")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
						output, err := exec.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir(hostDiskName), "-name", diskName},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(diskName))
					})

					It("[test_id:847]Should fail with a capacity option", func() {
						By(startingVMInstance)
						vmi = tests.NewRandomVMIWithHostDisk(diskPath, virtv1.HostDiskExists, nodeName)
						for i, volume := range vmi.Spec.Volumes {
							if volume.HostDisk != nil {
								vmi.Spec.Volumes[i].HostDisk.Capacity = resource.MustParse("1Gi")
								break
							}
						}
						_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
						Expect(err).To(HaveOccurred())
					})
				})

				Context("With unknown hostDisk type", func() {
					It("[test_id:852]Should fail to start VMI", func() {
						By(startingVMInstance)
						vmi = tests.NewRandomVMIWithHostDisk("/data/unknown.img", "unknown", "")
						_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("With multiple empty PVCs", func() {

				var pvcs = []string{}
				var node string

				BeforeEach(func() {
					for i := 0; i < 3; i++ {
						pvcs = append(pvcs, fmt.Sprintf("empty-pvc-%d-%s", i, rand.String(5)))
					}
					for _, pvc := range pvcs {
						hostpath := filepath.Join(testsuite.HostPathBase, pvc)
						node = libstorage.CreateHostPathPv(pvc, testsuite.GetTestNamespace(nil), hostpath)
						libstorage.CreateHostPathPVC(pvc, testsuite.GetTestNamespace(nil), "1G")
					}
				})

				AfterEach(func() {
					for _, pvc := range pvcs {
						libstorage.DeletePVC(pvc, testsuite.GetTestNamespace(nil))
						libstorage.DeletePV(pvc)
					}
				})

				// Not a candidate for NFS testing because multiple VMIs are started
				It("[test_id:868] Should initialize an empty PVC by creating a disk.img", func() {
					for _, pvc := range pvcs {
						By(startingVMInstance)
						vmi = libvmi.New(
							libvmi.WithPersistentVolumeClaim("disk0", fmt.Sprintf("disk-%s", pvc)),
							libvmi.WithResourceMemory("256Mi"),
							libvmi.WithNetwork(v1.DefaultPodNetwork()),
							libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
							libvmi.WithNodeSelectorFor(&k8sv1.Node{ObjectMeta: metav1.ObjectMeta{Name: node}}))
						tests.RunVMIAndExpectLaunch(vmi, 90)

						By("Checking if disk.img exists")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
						output, err := exec.ExecuteCommandOnPod(
							virtClient,
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", "/var/run/kubevirt-private/vmi-disks/disk0/", "-name", diskImgName, "-size", "1G", "-o", "-size", "+1G"},
						)
						Expect(err).ToNot(HaveOccurred())

						By("Checking if a disk image for PVC has been created")
						Expect(strings.Contains(output, diskImgName)).To(BeTrue())
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
					diskPath = filepath.Join(mountDir, diskImgName)
					srcDir := filepath.Join(tmpDir, "src")
					cmd := "mkdir -p " + mountDir + " && mkdir -p " + srcDir + " && chcon -t container_file_t " + srcDir + " && mount --bind " + srcDir + " " + mountDir + " && while true; do sleep 1; done"
					pod = tests.RenderHostPathPod("host-path-preparator", tmpDir, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationBidirectional, []string{tests.BinBash, "-c"}, []string{cmd})
					pod.Spec.Containers[0].Lifecycle = &k8sv1.Lifecycle{
						PreStop: &k8sv1.LifecycleHandler{
							Exec: &k8sv1.ExecAction{
								Command: []string{
									tests.BinBash, "-c",
									fmt.Sprintf("rm -f %s && umount %s", diskPath, mountDir),
								},
							},
						},
					}
					pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("Waiting for hostPath pod to prepare the mounted directory")
					Eventually(matcher.ThisPod(pod), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(k8sv1.PodReady))

					pod, err = ThisPod(pod)()
					Expect(err).ToNot(HaveOccurred())

					By("Determining the size of the mounted directory")
					diskSizeStr, _, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Spec.Containers[0].Name, []string{tests.BinBash, "-c", fmt.Sprintf("df %s | tail -n 1 | awk '{print $4}'", mountDir)})
					Expect(err).ToNot(HaveOccurred())
					diskSize, err = strconv.Atoi(strings.TrimSpace(diskSizeStr))
					diskSize = diskSize * 1000 // byte to kilobyte
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					if vmi != nil {
						Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
						libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
					}
					Expect(virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})).To(Succeed())
					waitForPodToDisappearWithTimeout(pod.Name, 120)
				})

				configureToleration := func(toleration int) {
					By("By configuring toleration")
					cfg := util.GetCurrentKv(virtClient).Spec.Configuration
					cfg.DeveloperConfiguration.LessPVCSpaceToleration = toleration
					tests.UpdateKubeVirtConfigValueAndWait(cfg)
				}

				// Not a candidate for NFS test due to usage of host disk
				It("[Serial][test_id:3108]Should not initialize an empty PVC with a disk.img when disk is too small even with toleration", Serial, func() {

					configureToleration(10)

					By(startingVMInstance)
					vmi = tests.NewRandomVMIWithHostDisk(diskPath, virtv1.HostDiskExistsOrCreate, pod.Spec.NodeName)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					tests.RunVMI(vmi, 30)

					By("Checking events")
					objectEventWatcher := watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(120) * time.Second)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					objectEventWatcher.WaitFor(ctx, watcher.WarningEvent, virtv1.SyncFailed.String())

				})

				// Not a candidate for NFS test due to usage of host disk
				It("[Serial][test_id:3109]Should initialize an empty PVC with a disk.img when disk is too small but within toleration", Serial, func() {

					configureToleration(30)

					By(startingVMInstance)
					vmi = tests.NewRandomVMIWithHostDisk(diskPath, virtv1.HostDiskExistsOrCreate, pod.Spec.NodeName)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					tests.RunVMIAndExpectLaunch(vmi, 30)

					By("Checking events")
					objectEventWatcher := watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(30) * time.Second)
					wp := watcher.WarningsPolicy{FailOnWarnings: true}
					objectEventWatcher.SetWarningsPolicy(wp)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					objectEventWatcher.WaitFor(ctx, watcher.EventType(hostdisk.EventTypeToleratedSmallPV), hostdisk.EventReasonToleratedSmallPV)
				})
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component][storage-req] With Cirros BlockMode PVC", decorators.StorageReq, func() {
			var dataVolume *cdiv1.DataVolume

			BeforeEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				dataVolume, err = createBlockDataVolume(virtClient)
				Expect(err).ToNot(HaveOccurred())
				if dataVolume == nil {
					Skip("Skip test when Block storage is not present")
				}

				libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))
			})

			AfterEach(func() {
				libstorage.DeleteDataVolume(&dataVolume)
			})

			// Not a candidate for NFS because local volumes are used in test
			It("[test_id:1015]should be successfully started", func() {
				// Start the VirtualMachineInstance with the PVC attached
				// Without userdata the hostname isn't set correctly and the login expecter fails...
				vmi = libvmi.New(
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithPersistentVolumeClaim("disk0", dataVolume.Name),
					libvmi.WithCloudInitNoCloudUserData(cirrosUserData, true),
				)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				By(checkingVMInstanceConsoleOut)
				Expect(console.LoginToCirros(vmi)).To(Succeed())
			})
		})

		Context("[storage-req][rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component]With Alpine block volume PVC", decorators.StorageReq, func() {

			It("[test_id:3139]should be successfully started", func() {
				By("Create a VMIWithPVC")
				// Start the VirtualMachineInstance with the PVC attached
				vmi, _ := tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteMany)
				By("Launching a VMI with PVC ")
				tests.RunVMIAndExpectLaunch(vmi, 180)

				By(checkingVMInstanceConsoleOut)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())
			})
		})

		Context("[rfe_id:2288][crit:high][arm64][vendor:cnv-qe@redhat.com][level:component] With not existing PVC", func() {
			// Not a candidate for NFS because the PVC in question doesn't actually exist
			It("[test_id:1040] should get unschedulable condition", func() {
				// Start the VirtualMachineInstance
				pvcName := "nonExistingPVC"
				vmi = libvmi.New(
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithPersistentVolumeClaim("disk0", pvcName),
				)
				vmi = tests.RunVMI(vmi, 10)

				virtClient := kubevirt.Client()

				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					if vmi.Status.Phase != virtv1.Pending {
						return false
					}
					if len(vmi.Status.Conditions) == 0 {
						return false
					}

					expectPodScheduledCondition := func(vmi *virtv1.VirtualMachineInstance) {
						getType := func(c virtv1.VirtualMachineInstanceCondition) string { return string(c.Type) }
						getReason := func(c virtv1.VirtualMachineInstanceCondition) string { return c.Reason }
						getStatus := func(c virtv1.VirtualMachineInstanceCondition) k8sv1.ConditionStatus { return c.Status }
						getMessage := func(c virtv1.VirtualMachineInstanceCondition) string { return c.Message }
						Expect(vmi.Status.Conditions).To(
							ContainElement(
								And(
									WithTransform(getType, Equal(string(k8sv1.PodScheduled))),
									WithTransform(getReason, Equal(k8sv1.PodReasonUnschedulable)),
									WithTransform(getStatus, Equal(k8sv1.ConditionFalse)),
									WithTransform(getMessage, Equal(fmt.Sprintf("PVC %v/%v does not exist, waiting for it to appear", vmi.Namespace, pvcName))),
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

				vmi = libvmi.NewAlpine(
					libvmi.WithEmptyDisk("emptydisk1", v1.DiskBusSCSI, resource.MustParse("1Gi")),
					libvmi.WithEmptyDisk("emptydisk2", v1.DiskBusSATA, resource.MustParse("1Gi")),
				)
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

			Context("With a USB device", func() {
				It("should successfully start and have the USB storage device attached", func() {
					vmi = libvmi.NewAlpine(
						libvmi.WithEmptyDisk("emptydisk1", v1.DiskBusUSB, resource.MustParse("128Mi")),
					)
					vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
					Expect(console.LoginToAlpine(vmi)).To(Succeed())

					By("Checking that /dev/sda has a capacity of 128Mi")
					Expect(console.ExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "blockdev --getsize64 /dev/sda\n"},
						&expect.BExp{R: "134217728"},
					}, 10*time.Second)).To(Succeed())

					By("Checking that the usb_storage kernel module has been loaded")
					Expect(console.ExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "mkdir /sys/module/usb_storage\n"},
						&expect.BExp{R: "mkdir: can't create directory '/sys/module/usb_storage': File exists"},
					}, 10*time.Second)).To(Succeed())
				})

			})

		})

		Context("[storage-req] With a volumeMode block backed ephemeral disk", decorators.StorageReq, func() {
			var dataVolume *cdiv1.DataVolume

			BeforeEach(func() {
				dataVolume, err = createBlockDataVolume(virtClient)
				Expect(err).ToNot(HaveOccurred())
				if dataVolume == nil {
					Skip("Skip test when Block storage is not present")
				}

				libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))
				vmi = nil
			})

			AfterEach(func() {
				libstorage.DeleteDataVolume(&dataVolume)
			})

			It("should generate the block backingstore disk within the domain", func() {
				vmi = tests.NewRandomVMIWithEphemeralPVC(dataVolume.Name)

				By("Initializing the VM")
				tests.RunVMIAndExpectLaunch(vmi, 90)

				runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())

				disks := runningVMISpec.Devices.Disks

				By("Checking if the disk backing store type is block")
				Expect(disks[0].BackingStore).ToNot(BeNil())
				Expect(disks[0].BackingStore.Type).To(Equal("block"))
				By("Checking if the disk backing store device path is appropriately configured")
				Expect(disks[0].BackingStore.Source.Dev).To(Equal(converter.GetBlockDeviceVolumePath("disk0")))
			})
			It("should generate the pod with the volumeDevice", func() {
				vmi = tests.NewRandomVMIWithEphemeralPVC(dataVolume.Name)
				By("Initializing the VM")

				tests.RunVMIAndExpectLaunch(vmi, 60)
				runningPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))

				By("Checking that the virt-launcher pod spec contains the volumeDevice")
				Expect(runningPod.Spec.Containers[0].VolumeDevices).NotTo(BeEmpty())
				Expect(runningPod.Spec.Containers[0].VolumeDevices[0].Name).To(Equal("disk0"))
			})
		})

		Context("disk shareable tunable", func() {
			var (
				dv         *cdiv1.DataVolume
				vmi1, vmi2 *virtv1.VirtualMachineInstance
			)
			BeforeEach(func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when Filesystem storage is not present")
				}

				dv = libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				labelKey := "testshareablekey"
				labels := map[string]string{
					labelKey: "",
				}

				// give an affinity rule to ensure the vmi's get placed on the same node.
				affinityRule := &k8sv1.Affinity{
					PodAffinity: &k8sv1.PodAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.WeightedPodAffinityTerm{
							{
								Weight: int32(1),
								PodAffinityTerm: k8sv1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      labelKey,
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{""}},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				}

				vmi1 = tests.NewRandomVMIWithDataVolume(dv.Name)
				vmi2 = tests.NewRandomVMIWithDataVolume(dv.Name)
				vmi1.Labels = labels
				vmi2.Labels = labels

				vmi1.Spec.Affinity = affinityRule
				vmi2.Spec.Affinity = affinityRule
			})

			It("should successfully start 2 VMs with a shareable disk", func() {
				setShareable(vmi1, "disk0")
				setShareable(vmi2, "disk0")

				By("Starting the VirtualMachineInstances")
				tests.RunVMIAndExpectLaunchWithDataVolume(vmi1, dv, 500)
				tests.RunVMIAndExpectLaunchWithDataVolume(vmi2, dv, 500)
			})
		})
		Context("write and read data from a shared disk", func() {
			It("should successfully write and read data", func() {
				const diskName = "disk1"
				const pvcClaim = "pvc-test-disk1"
				const labelKey = "testshareablekey"

				labels := map[string]string{
					labelKey: "",
				}

				// give an affinity rule to ensure the vmi's get placed on the same node.
				affinityRule := &k8sv1.Affinity{
					PodAffinity: &k8sv1.PodAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.WeightedPodAffinityTerm{
							{
								Weight: int32(1),
								PodAffinityTerm: k8sv1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      labelKey,
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{""}},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				}

				vmi1 := libvmi.NewAlpine(libvmi.WithPersistentVolumeClaim(diskName, pvcClaim))
				vmi2 := libvmi.NewAlpine(libvmi.WithPersistentVolumeClaim(diskName, pvcClaim))

				vmi1.Labels = labels
				vmi2.Labels = labels

				vmi1.Spec.Affinity = affinityRule
				vmi2.Spec.Affinity = affinityRule

				libstorage.CreateBlockPVC(pvcClaim, testsuite.GetTestNamespace(vmi1), "500Mi")
				setShareable(vmi1, diskName)
				setShareable(vmi2, diskName)

				By("Starting the VirtualMachineInstances")
				vmi1 = tests.RunVMIAndExpectLaunch(vmi1, 500)
				vmi2 = tests.RunVMIAndExpectLaunch(vmi2, 500)
				By("Write data from the first VMI")
				Expect(console.LoginToAlpine(vmi1)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi1, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: fmt.Sprintf("%s \n", `printf "Test awesome shareable disks" | dd  of=/dev/vdb bs=1 count=150 conv=notrunc`)},
					&expect.BExp{R: console.PromptExpression},
				}, 40)).To(Succeed())
				By("Read data from the second VMI")
				Expect(console.LoginToAlpine(vmi2)).To(Succeed())
				Expect(console.SafeExpectBatch(vmi2, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: fmt.Sprintf("dd  if=/dev/vdb bs=1 count=150 conv=notrunc \n")},
					&expect.BExp{R: "Test awesome shareable disks"},
				}, 40)).To(Succeed())

			})
		})

		Context("[Serial]with lun disk", Serial, func() {
			var (
				nodeName, address, device string
				pvc                       *k8sv1.PersistentVolumeClaim
				pv                        *k8sv1.PersistentVolume
			)
			addPVCLunDisk := func(vmi *virtv1.VirtualMachineInstance, deviceName, claimName string) {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, virtv1.Disk{
					Name: deviceName,
					DiskDevice: virtv1.DiskDevice{
						LUN: &virtv1.LunTarget{
							Bus:      v1.DiskBusSCSI,
							ReadOnly: false,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
					Name: deviceName,
					VolumeSource: virtv1.VolumeSource{
						PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: claimName,
						}},
					},
				})

			}
			addDataVolumeLunDisk := func(vmi *virtv1.VirtualMachineInstance, deviceName, claimName string) {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, virtv1.Disk{
					Name: deviceName,
					DiskDevice: virtv1.DiskDevice{
						LUN: &virtv1.LunTarget{
							Bus:      v1.DiskBusSCSI,
							ReadOnly: false,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, virtv1.Volume{
					Name: deviceName,
					VolumeSource: virtv1.VolumeSource{
						DataVolume: &virtv1.DataVolumeSource{
							Name: claimName,
						},
					},
				})

			}

			BeforeEach(func() {
				nodeName = tests.NodeNameWithHandler()
				address, device = tests.CreateSCSIDisk(nodeName, []string{})
			})

			AfterEach(func() {
				tests.RemoveSCSIDisk(nodeName, address)
				Expect(virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			})

			DescribeTable("should run the VMI using", func(addLunDisk func(*virtv1.VirtualMachineInstance, string, string)) {
				pv, pvc, err = tests.CreatePVandPVCwithSCSIDisk(nodeName, device, testsuite.GetTestNamespace(nil), "scsi-disks", "scsipv", "scsipvc")
				Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for scsi disk")

				By("Creating VMI with LUN disk")
				vmi := libvmi.NewAlpine()
				addLunDisk(vmi, "lun0", pvc.ObjectMeta.Name)
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), failedCreateVMI)

				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
			},
				Entry("PVC source", addPVCLunDisk),
				Entry("DataVolume source", addDataVolumeLunDisk),
			)

			It("should run the VMI created with a DataVolume source and use the LUN disk", func() {
				pv, err = tests.CreatePVwithSCSIDisk("scsi-disks", "scsipv", nodeName, device)
				Expect(err).ToNot(HaveOccurred())
				dv := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithPVC(libdv.PVCWithStorageClass(pv.Spec.StorageClassName),
						libdv.PVCWithBlockVolumeMode(),
						libdv.PVCWithAccessMode(k8sv1.ReadWriteOnce),
						libdv.PVCWithVolumeSize("8Mi"),
					),
				)
				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Creating VMI with LUN disk")
				vmi := libvmi.NewCirros(libvmi.WithResourceMemory("512M"))
				addDataVolumeLunDisk(vmi, "lun0", dv.Name)
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), failedCreateVMI)

				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(240),
				)
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				lunDisk := "/dev/"
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, volStatus := range vmi.Status.VolumeStatus {
						if volStatus.Name == "lun0" {
							lunDisk += volStatus.Target
							return true
						}
					}
					return false
				}, 30*time.Second, time.Second).Should(BeTrue())

				By(fmt.Sprintf("Checking that %s has a capacity of 8Mi", lunDisk))
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo blockdev --getsize64 %s\n", lunDisk)},
					&expect.BExp{R: "8388608"}, // 8Mi in bytes
				}, 30)).To(Succeed())

				By(fmt.Sprintf("Checking if we can write to %s", lunDisk))
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 -F %s\n", lunDisk)},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: tests.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
				}, 30)).To(Succeed())

				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.ObjectMeta.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
			})
		})
	})
})

func waitForPodToDisappearWithTimeout(podName string, seconds int) {
	virtClient := kubevirt.Client()
	EventuallyWithOffset(1, func() bool {
		_, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(nil)).Get(context.Background(), podName, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(BeTrue())
}

func createBlockDataVolume(virtClient kubecli.KubevirtClient) (*cdiv1.DataVolume, error) {
	sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteOnce)
	if !foundSC {
		return nil, nil
	}

	dataVolume := libdv.NewDataVolume(
		libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
		libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithBlockVolumeMode()),
	)

	return virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
}
