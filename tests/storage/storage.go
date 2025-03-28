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

	expect "github.com/google/goexpect"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
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
	customHostPath               = "custom-host-path"
	diskAlpineHostPath           = "disk-alpine-host-path"
	diskCustomHostPath           = "disk-custom-host-path"
)

const (
	diskSerial = "FB-fb_18030C10002032"
)

type VMICreationFunc func(string) *v1.VirtualMachineInstance

var _ = Describe(SIG("Storage", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		libstorage.CreateHostPathPv("alpine-host-path", testsuite.GetTestNamespace(nil), testsuite.HostPathAlpine)
		libstorage.CreateHostPathPVC("alpine-host-path", testsuite.GetTestNamespace(nil), "1Gi")
	})

	Describe("Starting a VirtualMachineInstance", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = nil
		})

		isPausedOnIOError := func(conditions []v1.VirtualMachineInstanceCondition) bool {
			for _, condition := range conditions {
				if condition.Type == v1.VirtualMachineInstancePaused {
					return condition.Status == k8sv1.ConditionTrue && condition.Reason == "PausedIOError"
				}
			}
			return false
		}

		setShareable := func(vmi *v1.VirtualMachineInstance, diskName string) {
			shareable := true
			for i, d := range vmi.Spec.Domain.Devices.Disks {
				if d.Name == diskName {
					vmi.Spec.Domain.Devices.Disks[i].Shareable = &shareable
					return
				}
			}
		}

		createAndWaitForVMIReady := func(vmi *v1.VirtualMachineInstance, dataVolume *cdiv1.DataVolume, timeoutSec int) *v1.VirtualMachineInstance {
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("Waiting until the DataVolume is ready")
			libstorage.EventuallyDV(dataVolume, timeoutSec, HaveSucceeded())
			By("Waiting until the VirtualMachineInstance starts")
			return libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running}, libwait.WithTimeout(timeoutSec))
		}

		Context("with error disk", Serial, func() {
			var (
				nodeName, address, device string

				pvc *k8sv1.PersistentVolumeClaim
				pv  *k8sv1.PersistentVolume
			)

			cleanUp := func(vmi *v1.VirtualMachineInstance) {
				By("Cleaning up")
				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.ObjectMeta.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
			}

			BeforeEach(func() {
				nodeName = libnode.GetNodeNameWithHandler()
				address, device = CreateErrorDisk(nodeName)
				pv, pvc, err = CreatePVandPVCwithFaultyDisk(nodeName, device, testsuite.GetTestNamespace(nil))
				Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for faulty disk")
			})

			AfterEach(func() {
				// In order to remove the scsi debug module, the SCSI device cannot be in used by the VM.
				// For this reason, we manually clean-up the VM  before removing the kernel module.
				RemoveSCSIDisk(nodeName, address)
				Expect(virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			})

			It("should pause VMI on IO error", func() {
				By("Creating VMI with faulty disk")
				vmi := libvmifact.NewAlpine(libvmi.WithPersistentVolumeClaim("pvc-disk", pvc.Name))
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
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
				FixErrorDevice(nodeName)

				By("Expecting VMI to NOT be paused")
				Eventually(ThisVMI(vmi), 100*time.Second, time.Second).Should(HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

				cleanUp(vmi)
			})

			It("should report IO errors in the guest with errorPolicy set to report", func() {
				const diskName = "disk1"
				By("Creating VMI with faulty disk")
				vmi := libvmifact.NewAlpine(libvmi.WithPersistentVolumeClaim(diskName, pvc.Name))
				for i, d := range vmi.Spec.Domain.Devices.Disks {
					if d.Name == diskName {
						vmi.Spec.Domain.Devices.Disks[i].ErrorPolicy = pointer.P(v1.DiskErrorPolicyReport)
					}
				}

				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), failedCreateVMI)

				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				)

				By("Writing to disk")
				Expect(console.LoginToAlpine(vmi)).To(Succeed(), "Should login")
				checkResultShellCommandOnVmi(vmi, "dd if=/dev/zero of=/dev/vdb",
					"dd: error writing '/dev/vdb': I/O error", 20)

				cleanUp(vmi)
			})
		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]with Alpine PVC", func() {
			newRandomVMIWithPVC := func(claimName string) *v1.VirtualMachineInstance {
				return libvmi.New(
					libvmi.WithPersistentVolumeClaim("disk0", claimName),
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithRng())
			}
			newRandomVMIWithCDRom := func(claimName string) *v1.VirtualMachineInstance {
				return libvmi.New(
					libvmi.WithCDRom("disk0", v1.DiskBusSATA, claimName),
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithRng())
			}

			Context("should be successfully", func() {
				DescribeTable("started", decorators.Conformance, func(newVMI VMICreationFunc, imageOwnedByQEMU bool) {
					pvcName := diskAlpineHostPath
					if !imageOwnedByQEMU {
						// Setup hostpath PV that points at non-root owned image with chmod 640
						libstorage.CreateHostPathPv(customHostPath, testsuite.GetTestNamespace(nil), testsuite.HostPathAlpineNoPriv)
						libstorage.CreateHostPathPVC(customHostPath, testsuite.GetTestNamespace(nil), "1Gi")
						pvcName = diskCustomHostPath
					}
					// Start the VirtualMachineInstance with the PVC attached
					vmi = newVMI(pvcName)

					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

					By(checkingVMInstanceConsoleOut)
					Expect(console.LoginToAlpine(vmi)).To(Succeed())
				},
					Entry("[test_id:3130]with Disk PVC", newRandomVMIWithPVC, true),
					Entry("[test_id:3131]with CDRom PVC", newRandomVMIWithCDRom, true),
					Entry("hostpath disk image file not owned by qemu", newRandomVMIWithPVC, false),
				)
			})

			DescribeTable("should be successfully started and stopped multiple times", func(newVMI VMICreationFunc) {
				vmi = newVMI(diskAlpineHostPath)

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					vmi := libvmops.RunVMIAndExpectLaunch(vmi, 90)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By(checkingVMInstanceConsoleOut)
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
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
				vmi = libvmifact.NewCirros(
					libvmi.WithResourceMemory("512M"),
					libvmi.WithEmptyDisk("emptydisk1", v1.DiskBusVirtio, resource.MustParse("1G")),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

				Expect(console.LoginToCirros(vmi)).To(Succeed())

				var emptyDiskDevice string
				Eventually(func() string {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					emptyDiskDevice = libstorage.LookupVolumeTargetPath(vmi, "emptydisk1")
					return emptyDiskDevice
				}, 30*time.Second, time.Second).ShouldNot(BeEmpty())
				By("Checking that the corresponding device has a capacity of 1G, aligned to 4k")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo blockdev --getsize64 %s\n", emptyDiskDevice)},
					&expect.BExp{R: "999292928"}, // 1G in bytes rounded down to nearest 1MiB boundary
				}, 10)).To(Succeed())

				By("Checking if we can write to the corresponding device")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 -F %s\n", emptyDiskDevice)},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: console.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
				}, 20)).To(Succeed())
			})
		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With an emptyDisk defined and a specified serial number", func() {
			// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
			It("[test_id:3135]should create a writeable emptyDisk with the specified serial number", func() {
				// Start the VirtualMachineInstance with the empty disk attached
				vmi = libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name:   "emptydisk1",
					Serial: diskSerial,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
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
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

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

				BeforeEach(func() {
					pvName = ""
				})

				AfterEach(func() {
					if vmi != nil {
						By("Deleting the VMI")
						Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed())

						By("Waiting for VMI to disappear")
						libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
					}
				})

				AfterEach(func() {
					if pvName != "" && pvName != diskAlpineHostPath {
						// PVs can't be reused
						By("Deleting PV and PVC")
						deletePvAndPvc(pvName)
					}
				})

				// The following case is mostly similar to the alpine PVC test above, except using different VirtualMachineInstance.
				It("[test_id:3136]started with Ephemeral PVC", decorators.Conformance, func() {
					pvName = diskAlpineHostPath

					vmi = libvmi.New(
						libvmi.WithResourceMemory("256Mi"),
						libvmi.WithEphemeralPersistentVolumeClaim("disk0", pvName),
					)

					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 120)

					By(checkingVMInstanceConsoleOut)
					Expect(console.LoginToAlpine(vmi)).To(Succeed())
				})
			})

			// Not a candidate for testing on NFS because the VMI is restarted and NFS PVC can't be re-used
			It("[test_id:3137]should not persist data", func() {
				vmi = libvmi.New(
					libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithEphemeralPersistentVolumeClaim("disk0", diskAlpineHostPath),
				)

				By("Starting the VirtualMachineInstance")
				var createdVMI *v1.VirtualMachineInstance
				if isRunOnKindInfra {
					createdVMI = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 90)
				} else {
					createdVMI = libvmops.RunVMIAndExpectLaunch(vmi, 90)
				}

				By("Writing an arbitrary file to it's EFI partition")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// Because "/" is mounted on tmpfs, we need something that normally persists writes - /dev/sda2 is the EFI partition formatted as vFAT.
					&expect.BSnd{S: "mount /dev/sda2 /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: console.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "echo content > /mnt/checkpoint\n"},
					&expect.BExp{R: console.PromptExpression},
					// The QEMU process will be killed, therefore the write must be flushed to the disk.
					&expect.BSnd{S: "sync\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 200)).To(Succeed())

				By("Killing a VirtualMachineInstance")
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForVirtualMachineToDisappearWithTimeout(createdVMI, 120)

				By("Starting the VirtualMachineInstance again")
				if isRunOnKindInfra {
					libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 90)
				} else {
					libvmops.RunVMIAndExpectLaunch(vmi, 90)
				}

				By("Making sure that the previously written file is not present")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// Same story as when first starting the VirtualMachineInstance - the checkpoint, if persisted, is located at /dev/sda2.
					&expect.BSnd{S: "mount /dev/sda2 /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: console.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/checkpoint &> /dev/null\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: console.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("1")},
				}, 200)).To(Succeed())
			})
		})

		Context("[rfe_id:3106][crit:medium][vendor:cnv-qe@redhat.com][level:component]With VirtualMachineInstance with two PVCs", func() {
			BeforeEach(func() {
				// Setup second PVC to use in this context
				libstorage.CreateHostPathPv(customHostPath, testsuite.GetTestNamespace(nil), testsuite.HostPathCustom)
				libstorage.CreateHostPathPVC(customHostPath, testsuite.GetTestNamespace(nil), "1Gi")
			})

			// Not a candidate for testing on NFS because the VMI is restarted and NFS PVC can't be re-used
			It("[test_id:3138]should start vmi multiple times", func() {
				vmi = libvmi.New(
					libvmi.WithPersistentVolumeClaim("disk0", diskAlpineHostPath),
					libvmi.WithPersistentVolumeClaim("disk1", diskCustomHostPath),
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithRng())

				num := 3
				By("Starting and stopping the VirtualMachineInstance number of times")
				for i := 1; i <= num; i++ {
					obj := libvmops.RunVMIAndExpectLaunch(vmi, 240)

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

					err = virtClient.VirtualMachineInstance(obj.Namespace).Delete(context.Background(), obj.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					Eventually(ThisVMI(obj), 120).Should(BeGone())
				}
			})
		})

		Context("With feature gates disabled for", Serial, func() {
			It("[test_id:4620]HostDisk, it should fail to start a VMI", func() {
				config.DisableFeatureGate(featuregate.HostDiskGate)
				vmi = libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithHostDisk("host-disk", "somepath", v1.HostDiskExistsOrCreate),
					// hostdisk needs a privileged namespace
					libvmi.WithNamespace(testsuite.NamespacePrivileged),
				)
				virtClient := kubevirt.Client()
				_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("HostDisk feature gate is not enabled"))
			})
		})

		Context("[rfe_id:2298][crit:medium][vendor:cnv-qe@redhat.com][level:component] With HostDisk and PVC initialization", func() {
			BeforeEach(func() {
				if !checks.HasFeature(featuregate.HostDiskGate) {
					Skip("Cluster has the HostDisk featuregate disabled, skipping  the tests")
				}
			})

			Context("With a HostDisk defined", func() {
				var hostDiskDir string
				var nodeName string

				BeforeEach(func() {
					hostDiskDir = RandHostDiskDir()
					nodeName = ""
				})

				AfterEach(func() {
					if vmi != nil {
						err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
						if err != nil && !errors.IsNotFound(err) {
							Expect(err).ToNot(HaveOccurred())
						}
						Eventually(ThisVMI(vmi), 30).Should(Or(BeGone(), BeInPhase(v1.Failed), BeInPhase(v1.Succeeded)))
					}
					if nodeName != "" {
						Expect(RemoveHostDisk(hostDiskDir, nodeName)).To(Succeed())
					}
				})

				Context("With 'DiskExistsOrCreate' type", func() {
					var diskName string
					var diskPath string
					BeforeEach(func() {
						diskName = fmt.Sprintf("disk-%s.img", uuid.NewString())
						diskPath = filepath.Join(hostDiskDir, diskName)
					})

					DescribeTable("Should create a disk image and start", func(driver v1.DiskBus) {
						By(startingVMInstance)
						vmi = libvmi.New(
							libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
							libvmi.WithNetwork(v1.DefaultPodNetwork()),
							libvmi.WithResourceMemory("128Mi"),
							libvmi.WithHostDisk("host-disk", diskPath, v1.HostDiskExistsOrCreate),
							// hostdisk needs a privileged namespace
							libvmi.WithNamespace(testsuite.NamespacePrivileged),
						)
						vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = driver

						vmi = libvmops.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if disk.img has been created")
						vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
						Expect(err).ToNot(HaveOccurred())

						nodeName = vmiPod.Spec.NodeName
						output, err := exec.ExecuteCommandOnPod(
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
						vmi = libvmi.New(
							libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
							libvmi.WithNetwork(v1.DefaultPodNetwork()),
							libvmi.WithResourceMemory("128Mi"),
							libvmi.WithHostDisk("host-disk", diskPath, v1.HostDiskExistsOrCreate),
							libvmi.WithHostDisk("anotherdisk", filepath.Join(hostDiskDir, "another.img"), v1.HostDiskExistsOrCreate),
							// hostdisk needs a privileged namespace
							libvmi.WithNamespace(testsuite.NamespacePrivileged),
						)
						vmi = libvmops.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if another.img has been created")
						vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
						Expect(err).ToNot(HaveOccurred())

						nodeName = vmiPod.Spec.NodeName
						output, err := exec.ExecuteCommandOnPod(
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir("anotherdisk"), "-size", "1G", "-o", "-size", "+1G"},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(hostdisk.GetMountedHostDiskPath("anotherdisk", filepath.Join(hostDiskDir, "another.img"))))

						By("Checking if disk.img has been created")
						output, err = exec.ExecuteCommandOnPod(
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
						diskName = fmt.Sprintf("disk-%s.img", uuid.NewString())
						diskPath = filepath.Join(hostDiskDir, diskName)
						// create a disk image before test
						pod := CreateHostDisk(diskPath)
						pod, err = virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), pod, metav1.CreateOptions{})
						Expect(err).ToNot(HaveOccurred())

						Eventually(ThisPod(pod), 30*time.Second, 1*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
						pod, err = ThisPod(pod)()
						Expect(err).NotTo(HaveOccurred())
						nodeName = pod.Spec.NodeName
					})

					It("[test_id:2306]Should use existing disk image and start", func() {
						By(startingVMInstance)
						vmi = libvmi.New(
							libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
							libvmi.WithNetwork(v1.DefaultPodNetwork()),
							libvmi.WithResourceMemory("128Mi"),
							libvmi.WithHostDisk("host-disk", diskPath, v1.HostDiskExists),
							libvmi.WithNodeAffinityFor(nodeName),
							// hostdisk needs a privileged namespace
							libvmi.WithNamespace(testsuite.NamespacePrivileged),
						)
						vmi = libvmops.RunVMIAndExpectLaunch(vmi, 30)

						By("Checking if disk.img exists")
						vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
						Expect(err).ToNot(HaveOccurred())

						output, err := exec.ExecuteCommandOnPod(
							vmiPod,
							vmiPod.Spec.Containers[0].Name,
							[]string{"find", hostdisk.GetMountedHostDiskDir(hostDiskName), "-name", diskName},
						)
						Expect(err).ToNot(HaveOccurred())
						Expect(output).To(ContainSubstring(diskName))
					})

					It("[test_id:847]Should fail with a capacity option", func() {
						By(startingVMInstance)
						vmi = libvmi.New(
							libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
							libvmi.WithNetwork(v1.DefaultPodNetwork()),
							libvmi.WithResourceMemory("128Mi"),
							libvmi.WithHostDisk("host-disk", diskPath, v1.HostDiskExists),
							libvmi.WithNodeAffinityFor(nodeName),
							// hostdisk needs a privileged namespace
							libvmi.WithNamespace(testsuite.NamespacePrivileged),
						)
						for i, volume := range vmi.Spec.Volumes {
							if volume.HostDisk != nil {
								vmi.Spec.Volumes[i].HostDisk.Capacity = resource.MustParse("1Gi")
								break
							}
						}
						_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
						Expect(err).To(HaveOccurred())
					})
				})

				Context("With unknown hostDisk type", func() {
					It("[test_id:852]Should fail to start VMI", func() {
						By(startingVMInstance)
						vmi = libvmi.New(
							libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
							libvmi.WithNetwork(v1.DefaultPodNetwork()),
							libvmi.WithResourceMemory("128Mi"),
							libvmi.WithHostDisk("host-disk", "/data/unknown.img", "unknown"),
							// hostdisk needs a privileged namespace
							libvmi.WithNamespace(testsuite.NamespacePrivileged),
						)
						_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("With multiple empty PVCs", func() {
				pvcs := []string{}
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
							libvmi.WithNodeSelectorFor(node))
						vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

						By("Checking if disk.img exists")
						vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
						Expect(err).ToNot(HaveOccurred())

						output, err := exec.ExecuteCommandOnPod(
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
					tmpDir := RandHostDiskDir()
					mountDir = filepath.Join(tmpDir, "mount")
					diskPath = filepath.Join(mountDir, diskImgName)
					srcDir := filepath.Join(tmpDir, "src")
					cmd := "mkdir -p " + mountDir + " && mkdir -p " + srcDir + " && chcon -t container_file_t " + srcDir + " && mount --bind " + srcDir + " " + mountDir + " && while true; do sleep 1; done"
					pod = libpod.RenderHostPathPod("host-path-preparator", tmpDir, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationBidirectional, []string{"/bin/bash", "-c"}, []string{cmd})
					pod.Spec.Containers[0].Lifecycle = &k8sv1.Lifecycle{
						PreStop: &k8sv1.LifecycleHandler{
							Exec: &k8sv1.ExecAction{
								Command: []string{
									"/bin/bash", "-c",
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
					diskSizeStr, _, err := exec.ExecuteCommandOnPodWithResults(pod, pod.Spec.Containers[0].Name, []string{"/bin/bash", "-c", fmt.Sprintf("df %s | tail -n 1 | awk '{print $4}'", mountDir)})
					Expect(err).ToNot(HaveOccurred())
					diskSize, err = strconv.Atoi(strings.TrimSpace(diskSizeStr))
					diskSize = diskSize * 1000 // byte to kilobyte
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					if vmi != nil {
						Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed())
						libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
					}
					Expect(virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})).To(Succeed())
					waitForPodToDisappearWithTimeout(pod.Name, 120)
				})

				configureToleration := func(toleration int) {
					By("By configuring toleration")
					cfg := libkubevirt.GetCurrentKv(virtClient).Spec.Configuration
					cfg.DeveloperConfiguration.LessPVCSpaceToleration = toleration
					config.UpdateKubeVirtConfigValueAndWait(cfg)
				}

				// Not a candidate for NFS test due to usage of host disk
				It("[test_id:3108]Should not initialize an empty PVC with a disk.img when disk is too small even with toleration", Serial, func() {
					configureToleration(10)

					By(startingVMInstance)
					vmi = libvmi.New(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
						libvmi.WithResourceMemory("128Mi"),
						libvmi.WithHostDisk("host-disk", diskPath, v1.HostDiskExistsOrCreate),
						libvmi.WithNodeAffinityFor(pod.Spec.NodeName),
						// hostdisk needs a privileged namespace
						libvmi.WithNamespace(testsuite.NamespacePrivileged),
					)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Checking events")
					objectEventWatcher := watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(120) * time.Second)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					objectEventWatcher.WaitFor(ctx, watcher.WarningEvent, v1.SyncFailed.String())
				})

				// Not a candidate for NFS test due to usage of host disk
				It("[test_id:3109]Should initialize an empty PVC with a disk.img when disk is too small but within toleration", Serial, func() {
					configureToleration(30)

					By(startingVMInstance)
					vmi = libvmi.New(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
						libvmi.WithResourceMemory("128Mi"),
						libvmi.WithHostDisk("host-disk", diskPath, v1.HostDiskExistsOrCreate),
						libvmi.WithNodeAffinityFor(pod.Spec.NodeName),
						// hostdisk needs a privileged namespace
						libvmi.WithNamespace(testsuite.NamespacePrivileged),
					)
					vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(strconv.Itoa(int(float64(diskSize) * 1.2)))
					libvmops.RunVMIAndExpectLaunch(vmi, 30)

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

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component][storage-req] With Cirros BlockMode PVC", decorators.RequiresBlockStorage, decorators.StorageReq, func() {
			var dataVolume *cdiv1.DataVolume
			var err error

			BeforeEach(func() {
				// create a new PV and PVC (PVs can't be reused)
				sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteOnce)
				if !foundSC {
					Fail("Fail test when Block storage is not present")
				}

				dataVolume = libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc), libdv.StorageWithBlockVolumeMode()),
				)
				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), WaitForFirstConsumer()))
			})

			// Not a candidate for NFS because local volumes are used in test
			It("[test_id:1015]should be successfully started", func() {
				// Start the VirtualMachineInstance with the PVC attached
				// Without userdata the hostname isn't set correctly and the login expecter fails...
				vmi = libvmi.New(
					libvmi.WithResourceMemory("256Mi"),
					libvmi.WithPersistentVolumeClaim("disk0", dataVolume.Name),
					libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

				By(checkingVMInstanceConsoleOut)
				Expect(console.LoginToCirros(vmi)).To(Succeed())
			})
		})

		Context("[storage-req][rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component]With Alpine block volume PVC", decorators.RequiresRWXBlock, decorators.StorageReq, func() {
			It("[test_id:3139]should be successfully started", func() {
				By("Create a VMIWithPVC")
				sc, exists := libstorage.GetRWXBlockStorageClass()
				if !exists {
					Fail("Fail test when Block storage is not present")
				}

				// Start the VirtualMachineInstance with the PVC attached
				dv := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(sc),
						libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine))),
						libdv.StorageWithAccessMode(k8sv1.ReadWriteMany),
						libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeBlock),
					),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi := libstorage.RenderVMIWithDataVolume(dv.Name, dv.Namespace)
				createAndWaitForVMIReady(vmi, dv, 240)
				By(checkingVMInstanceConsoleOut)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())
			})
		})

		Context("[rfe_id:2288][crit:high][vendor:cnv-qe@redhat.com][level:component] With not existing PVC", decorators.WgS390x, decorators.WgArm64, func() {
			// Not a candidate for NFS because the PVC in question doesn't actually exist
			It("[test_id:1040] should get unschedulable condition", func() {
				// Start the VirtualMachineInstance
				pvcName := "nonExistingPVC"
				vmi = libvmi.New(
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithPersistentVolumeClaim("disk0", pvcName),
				)
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
				vmi = libvmifact.NewAlpine(
					libvmi.WithEmptyDisk("emptydisk1", v1.DiskBusSCSI, resource.MustParse("1Gi")),
					libvmi.WithEmptyDisk("emptydisk2", v1.DiskBusSATA, resource.MustParse("1Gi")),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

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
				It("[test_id:9797]should successfully start and have the USB storage device attached", func() {
					vmi = libvmifact.NewAlpine(
						libvmi.WithEmptyDisk("emptydisk1", v1.DiskBusUSB, resource.MustParse("128Mi")),
					)
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
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

		Context("[storage-req] With a volumeMode block backed ephemeral disk", decorators.RequiresBlockStorage, decorators.StorageReq, func() {
			var dataVolume *cdiv1.DataVolume
			var err error

			BeforeEach(func() {
				sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteOnce)
				if !foundSC {
					Fail("Fail test when Block storage is not present")
				}

				dataVolume = libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc), libdv.StorageWithBlockVolumeMode()),
				)
				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), WaitForFirstConsumer()))
				vmi = nil
			})

			It("should generate the pod with the volumeDevice", func() {
				vmi = libvmifact.NewGuestless(
					libvmi.WithEphemeralPersistentVolumeClaim("disk0", dataVolume.Name),
				)

				By("Initializing the VM")

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)
				runningPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that the virt-launcher pod spec contains the volumeDevice")
				Expect(runningPod.Spec.Containers[0].VolumeDevices).NotTo(BeEmpty())
				Expect(runningPod.Spec.Containers[0].VolumeDevices[0].Name).To(Equal("disk0"))
			})
		})

		Context("disk shareable tunable", func() {
			var (
				dv         *cdiv1.DataVolume
				vmi1, vmi2 *v1.VirtualMachineInstance
			)
			BeforeEach(func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Fail("Fail test when Filesystem storage is not present")
				}

				dv = libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc)),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				labelKey := "testshareablekey"

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
												Values:   []string{""},
											},
										},
									},
									TopologyKey: k8sv1.LabelHostname,
								},
							},
						},
					},
				}
				vmi1 = libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithDataVolume("disk0", dv.Name),
					libvmi.WithResourceMemory("1Gi"),
					libvmi.WithLabel(labelKey, ""),
				)
				vmi2 = libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithDataVolume("disk0", dv.Name),
					libvmi.WithResourceMemory("1Gi"),
					libvmi.WithLabel(labelKey, ""),
				)

				vmi1.Spec.Affinity = affinityRule
				vmi2.Spec.Affinity = affinityRule
			})

			It("should successfully start 2 VMs with a shareable disk", func() {
				setShareable(vmi1, "disk0")
				setShareable(vmi2, "disk0")

				By("Starting the VirtualMachineInstances")
				createAndWaitForVMIReady(vmi1, dv, 500)
				createAndWaitForVMIReady(vmi2, dv, 500)
			})
		})
		Context("write and read data from a shared disk", func() {
			It("should successfully write and read data", decorators.RequiresBlockStorage, func() {
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
												Values:   []string{""},
											},
										},
									},
									TopologyKey: k8sv1.LabelHostname,
								},
							},
						},
					},
				}

				vmi1 := libvmifact.NewAlpine(libvmi.WithPersistentVolumeClaim(diskName, pvcClaim))
				vmi2 := libvmifact.NewAlpine(libvmi.WithPersistentVolumeClaim(diskName, pvcClaim))

				vmi1.Labels = labels
				vmi2.Labels = labels

				vmi1.Spec.Affinity = affinityRule
				vmi2.Spec.Affinity = affinityRule

				libstorage.CreateBlockPVC(pvcClaim, testsuite.GetTestNamespace(vmi1), "500Mi")
				setShareable(vmi1, diskName)
				setShareable(vmi2, diskName)

				By("Starting the VirtualMachineInstances")
				vmi1 = libvmops.RunVMIAndExpectLaunch(vmi1, 500)
				vmi2 = libvmops.RunVMIAndExpectLaunch(vmi2, 500)
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

		Context("with lun disk", Serial, func() {
			var (
				nodeName, address, device string
				pvc                       *k8sv1.PersistentVolumeClaim
				pv                        *k8sv1.PersistentVolume
			)
			addPVCLunDisk := func(vmi *v1.VirtualMachineInstance, deviceName, claimName string) {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: deviceName,
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{
							Bus:      v1.DiskBusSCSI,
							ReadOnly: false,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: deviceName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: claimName,
						}},
					},
				})
			}
			addDataVolumeLunDisk := func(vmi *v1.VirtualMachineInstance, deviceName, claimName string) {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: deviceName,
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{
							Bus:      v1.DiskBusSCSI,
							ReadOnly: false,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: deviceName,
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: claimName,
						},
					},
				})
			}

			BeforeEach(func() {
				nodeName = libnode.GetNodeNameWithHandler()
				address, device = CreateSCSIDisk(nodeName, []string{})
			})

			AfterEach(func() {
				RemoveSCSIDisk(nodeName, address)
				Expect(virtClient.CoreV1().PersistentVolumes().Delete(context.Background(), pv.Name, metav1.DeleteOptions{})).NotTo(HaveOccurred())
			})

			DescribeTable("should run the VMI using", func(addLunDisk func(*v1.VirtualMachineInstance, string, string)) {
				pv, pvc, err = CreatePVandPVCwithSCSIDisk(nodeName, device, testsuite.GetTestNamespace(nil), "scsi-disks", "scsipv", "scsipvc")
				Expect(err).NotTo(HaveOccurred(), "Failed to create PV and PVC for scsi disk")

				By("Creating VMI with LUN disk")
				vmi := libvmifact.NewAlpine()
				addLunDisk(vmi, "lun0", pvc.ObjectMeta.Name)
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), failedCreateVMI)

				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.ObjectMeta.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
			},
				Entry("PVC source", addPVCLunDisk),
				Entry("DataVolume source", addDataVolumeLunDisk),
			)

			It("should run the VMI created with a DataVolume source and use the LUN disk", func() {
				pv, err = CreatePVwithSCSIDisk("scsi-disks", "scsipv", nodeName, device)
				Expect(err).ToNot(HaveOccurred())
				dv := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithStorage(libdv.StorageWithStorageClass(pv.Spec.StorageClassName),
						libdv.StorageWithBlockVolumeMode(),
						libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
						libdv.StorageWithVolumeSize("8Mi"),
					),
				)
				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Creating VMI with LUN disk")
				vmi := libvmifact.NewCirros(libvmi.WithResourceMemory("512M"))
				addDataVolumeLunDisk(vmi, "lun0", dv.Name)
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), failedCreateVMI)

				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(240),
				)
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				var lunDisk string
				Eventually(func() string {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					lunDisk = libstorage.LookupVolumeTargetPath(vmi, "lun0")
					return lunDisk
				}, 30*time.Second, time.Second).ShouldNot(BeEmpty())

				By(fmt.Sprintf("Checking that %s has a capacity of 8Mi", lunDisk))
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo blockdev --getsize64 %s\n", lunDisk)},
					&expect.BExp{R: "8388608"}, // 8Mi in bytes
				}, 30)).To(Succeed())

				By(fmt.Sprintf("Checking if we can write to %s", lunDisk))
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 -F %s\n", lunDisk)},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: console.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
				}, 30)).To(Succeed())

				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.ObjectMeta.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 180)
			})
		})
	})
}))

func waitForPodToDisappearWithTimeout(podName string, seconds int) {
	virtClient := kubevirt.Client()
	EventuallyWithOffset(1, func() error {
		_, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(nil)).Get(context.Background(), podName, metav1.GetOptions{})
		return err
	}, seconds, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
}

func checkResultShellCommandOnVmi(vmi *v1.VirtualMachineInstance, cmd, output string, timeout int) {
	res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
		&expect.BSnd{S: fmt.Sprintf("%s\n", cmd)},
		&expect.BExp{R: console.PromptExpression},
	}, timeout)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, res).ToNot(BeEmpty())
	ExpectWithOffset(1, res[0].Output).To(ContainSubstring(output))
}

func deletePvAndPvc(name string) {
	virtCli := kubevirt.Client()

	err := virtCli.CoreV1().PersistentVolumes().Delete(context.Background(), name, metav1.DeleteOptions{})
	Expect(err).To(Or(
		Not(HaveOccurred()),
		MatchError(errors.IsNotFound, "errors.IsNotFound"),
	))

	err = virtCli.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Delete(context.Background(), name, metav1.DeleteOptions{})
	Expect(err).To(Or(
		Not(HaveOccurred()),
		MatchError(errors.IsNotFound, "errors.IsNotFound"),
	))
}

func runPodAndExpectPhase(pod *k8sv1.Pod, phase k8sv1.PodPhase) *k8sv1.Pod {
	virtClient := kubevirt.Client()

	var err error
	pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(ThisPod(pod), 120).Should(BeInPhase(phase))

	pod, err = ThisPod(pod)()
	Expect(err).ToNot(HaveOccurred())
	Expect(pod).ToNot(BeNil())
	return pod
}
