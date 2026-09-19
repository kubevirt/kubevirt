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
 * Copyright The KubeVirt Authors.
 *
 */

package storage

import (
	"context"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const vmStateEFIVar = "kvtest-12345678-1234-1234-1234-123456789abc"

var _ = Describe(SIG("VirtualMachineState", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	vmStateTemplate := func(storageClass string) *k8sv1.PersistentVolumeClaimTemplate {
		return &k8sv1.PersistentVolumeClaimTemplate{
			Spec: k8sv1.PersistentVolumeClaimSpec{
				AccessModes:      []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				StorageClassName: pointer.P(storageClass),
				Resources: k8sv1.VolumeResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceStorage: resource.MustParse("128Mi"),
					},
				},
			},
		}
	}

	newPersistentFedora := func(opts ...libvmi.Option) *v1.VirtualMachineInstance {
		vmi := libvmifact.NewFedora(opts...)
		vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{Persistent: pointer.P(true)}
		vmi.Spec.Domain.Firmware = &v1.Firmware{
			Bootloader: &v1.Bootloader{
				EFI: &v1.EFI{SecureBoot: pointer.P(false), Persistent: pointer.P(true)},
			},
		}
		return vmi
	}

	// TPM and EFI are enabled without persistent:true; when virtualMachineState is set, both are
	// treated as persistent implicitly (VEP #312).
	newImplicitFedora := func(opts ...libvmi.Option) *v1.VirtualMachineInstance {
		vmi := libvmifact.NewFedora(opts...)
		vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{}
		vmi.Spec.Domain.Firmware = &v1.Firmware{
			Bootloader: &v1.Bootloader{
				EFI: &v1.EFI{SecureBoot: pointer.P(false)},
			},
		}
		return vmi
	}

	ownedPVCName := func(vm *v1.VirtualMachine) string {
		pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", backendstorage.VMStateOwnerLabel, vm.UID),
		})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, pvcs.Items).To(HaveLen(1))
		return pvcs.Items[0].Name
	}

	createAndStartVM := func(vmi *v1.VirtualMachineInstance) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
			if err != nil && !k8serrors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
		})
		Eventually(matcher.ThisVM(vm)).WithTimeout(360 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		// Wait for the guest agent so the firmware finished booting; otherwise LoginToFedora's
		// keystrokes can hit the persistent-EFI boot menu and stall.
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(4 * time.Minute).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		return vm, vmi
	}

	restartVM := func(vm *v1.VirtualMachine) *v1.VirtualMachineInstance {
		vm = libvmops.StopVirtualMachine(vm)
		vm = libvmops.StartVirtualMachine(vm)
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		// Wait for the guest agent so the firmware finished booting; otherwise LoginToFedora's
		// keystrokes can hit the persistent-EFI boot menu and stall.
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(4 * time.Minute).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		return vmi
	}

	storeTPMSecret := func(vmi *v1.VirtualMachineInstance) {
		ExpectWithOffset(1, console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "tpm2_createprimary -Q --hierarchy=o --key-context=prim.ctx\n"},
			&expect.BExp{R: ""},
			&expect.BSnd{S: "echo MYSECRET | tpm2_create --hash-algorithm=sha256 --public=seal.pub --private=seal.priv --sealing-input=- --parent-context=prim.ctx\n"},
			&expect.BExp{R: ""},
			&expect.BSnd{S: "tpm2_load -Q --parent-context=prim.ctx --public=seal.pub --private=seal.priv --name=seal.name --key-context=seal.ctx\n"},
			&expect.BExp{R: ""},
			&expect.BSnd{S: "tpm2_evictcontrol --hierarchy=o --object-context=seal.ctx 0x81010002\n"},
			&expect.BExp{R: ""},
			&expect.BSnd{S: "tpm2_unseal -Q --object-context=0x81010002\n"},
			&expect.BExp{R: "MYSECRET"},
		}, 300)).To(Succeed(), "failed to store secret into the TPM")
	}

	checkTPMSecret := func(vmi *v1.VirtualMachineInstance) {
		ExpectWithOffset(1, console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "tpm2_unseal -Q --object-context=0x81010002\n"},
			&expect.BExp{R: "MYSECRET"},
		}, 300)).To(Succeed(), "the state of the TPM did not persist")
	}

	storeEFIVar := func(vmi *v1.VirtualMachineInstance) {
		cmd := fmt.Sprintf(`printf "\x07\x00\x00\x00\x42" > /sys/firmware/efi/efivars/%s`, vmStateEFIVar)
		ExpectWithOffset(1, console.RunCommand(vmi, cmd, 10*time.Second)).To(Succeed(), "failed to store an EFI variable")
	}

	checkEFIVar := func(vmi *v1.VirtualMachineInstance) {
		ExpectWithOffset(1, console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: fmt.Sprintf("hexdump /sys/firmware/efi/efivars/%s\n", vmStateEFIVar)},
			&expect.BExp{R: "0042"},
		}, 10)).To(Succeed(), "the EFI variable did not persist")
	}

	Context("with a volumeClaimTemplate", func() {
		It("creates a dedicated VirtualMachineState PVC and persists TPM and EFI state across restarts", func() {
			sc, found := libstorage.GetRWOFileSystemStorageClass()
			if !found {
				Skip("Skipping test, no RWO filesystem storage class available")
			}

			vmi := newPersistentFedora()
			vmi.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				VolumeClaimTemplate: vmStateTemplate(sc),
			}

			By("Starting the VM with a declarative VirtualMachineState")
			vm, vmi := createAndStartVM(vmi)

			By("Verifying a dedicated VirtualMachineState PVC was created and owned")
			var pvcs *k8sv1.PersistentVolumeClaimList
			Eventually(func() []k8sv1.PersistentVolumeClaim {
				var err error
				pvcs, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: backendstorage.VMStateOwnerLabel,
				})
				Expect(err).ToNot(HaveOccurred())
				return pvcs.Items
			}).WithTimeout(60 * time.Second).WithPolling(time.Second).Should(HaveLen(1))
			pvc := pvcs.Items[0]
			// The PVC name comes from GenerateName, which the apiserver may truncate, so only the
			// prefix is guaranteed; ownership is checked via the owner-UID label instead.
			Expect(pvc.Name).To(HavePrefix(backendstorage.PVCPrefix + "-"))
			Expect(pvc.Labels).To(HaveKeyWithValue(backendstorage.VMStateOwnerLabel, string(vm.UID)))
			Expect(pvc.Spec.VolumeMode).To(HaveValue(Equal(k8sv1.PersistentVolumeFilesystem)))

			By("Verifying the VirtualMachineState volume is reported on the VM and VMI status")
			Expect(vmi.Status.VirtualMachineStateVolume).ToNot(BeNil())
			Eventually(func() *v1.VolumeStatus {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedVM.Status.VirtualMachineStateVolume
			}).WithTimeout(60 * time.Second).WithPolling(time.Second).ShouldNot(BeNil())

			By("Storing state into the guest TPM and EFI")
			storeTPMSecret(vmi)
			storeEFIVar(vmi)

			By("Restarting the VM and verifying the state persisted")
			vmi = restartVM(vm)
			checkTPMSecret(vmi)
			checkEFIVar(vmi)

			By("Verifying the restart reused the same PVC")
			Expect(ownedPVCName(vm)).To(Equal(pvc.Name))
		})
	})

	Context("with implicit persistence", func() {
		It("persists TPM and EFI state across restarts without persistent:true", func() {
			sc, found := libstorage.GetRWOFileSystemStorageClass()
			if !found {
				Skip("Skipping test, no RWO filesystem storage class available")
			}

			vmi := newImplicitFedora()
			vmi.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				VolumeClaimTemplate: vmStateTemplate(sc),
			}

			By("Starting a VM whose TPM/EFI are implicitly persistent")
			vm, vmi := createAndStartVM(vmi)

			By("Verifying an owned VirtualMachineState PVC was created")
			pvcName := ownedPVCName(vm)

			By("Storing state into the guest TPM and EFI")
			storeTPMSecret(vmi)
			storeEFIVar(vmi)

			By("Restarting the VM and verifying the state persisted")
			vmi = restartVM(vm)
			checkTPMSecret(vmi)
			checkEFIVar(vmi)

			By("Verifying the restart reused the same PVC")
			Expect(ownedPVCName(vm)).To(Equal(pvcName))
		})
	})

	Context("garbage collection", func() {
		It("deletes a templated VirtualMachineState PVC when the VM is deleted", func() {
			sc, found := libstorage.GetRWOFileSystemStorageClass()
			if !found {
				Skip("Skipping test, no RWO filesystem storage class available")
			}

			vmi := newPersistentFedora()
			vmi.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				VolumeClaimTemplate: vmStateTemplate(sc),
			}

			By("Starting a VM that owns a templated VirtualMachineState PVC")
			vm, _ := createAndStartVM(vmi)
			pvcName := ownedPVCName(vm)

			By("Deleting the VM")
			Expect(virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})).To(Succeed())

			By("Verifying the owned PVC is garbage-collected with the VM")
			Eventually(func() bool {
				_, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
				return k8serrors.IsNotFound(err)
			}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(BeTrue())
		})

		It("keeps a source-adopted PVC after the VM is deleted", func() {
			sc, found := libstorage.GetRWOFileSystemStorageClass()
			if !found {
				Skip("Skipping test, no RWO filesystem storage class available")
			}

			By("Pre-creating a PVC to adopt as VirtualMachineState")
			pvc := libstorage.CreateFSPVC("vmstate-adopt-gc", testsuite.GetTestNamespace(nil), "128Mi", libstorage.WithStorageClass(sc))
			DeferCleanup(func() {
				err := virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.Background(), pvc.Name, metav1.DeleteOptions{})
				if err != nil && !k8serrors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			vmi := newPersistentFedora()
			vmi.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				Source: &v1.VirtualMachineStateSource{Name: pvc.Name},
			}

			By("Starting a VM that adopts the PVC via source")
			vm, _ := createAndStartVM(vmi)

			By("Deleting the VM")
			Expect(virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})).To(Succeed())
			Eventually(func() bool {
				_, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				return k8serrors.IsNotFound(err)
			}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(BeTrue())

			By("Verifying the source-adopted PVC is not garbage-collected")
			_, err := virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("adopting an existing PVC via source", func() {
		It("carries VirtualMachineState between VMs when the previous VM is stopped", func() {
			sc, found := libstorage.GetRWOFileSystemStorageClass()
			if !found {
				Skip("Skipping test, no RWO filesystem storage class available")
			}

			By("Starting a first VM that owns a VirtualMachineState PVC")
			firstVMI := newPersistentFedora()
			firstVMI.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				VolumeClaimTemplate: vmStateTemplate(sc),
			}
			firstVM, firstVMI := createAndStartVM(firstVMI)

			By("Writing state into the first VM")
			storeTPMSecret(firstVMI)
			storeEFIVar(firstVMI)

			pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(firstVM.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: backendstorage.VMStateOwnerLabel,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pvcs.Items).To(HaveLen(1))
			pvcName := pvcs.Items[0].Name

			By("Stopping the first VM so its VirtualMachineState PVC is free")
			libvmops.StopVirtualMachine(firstVM)

			By("Starting a second VM that adopts the PVC via source")
			secondVMI := newPersistentFedora()
			secondVMI.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				Source: &v1.VirtualMachineStateSource{Name: pvcName},
			}
			_, secondVMI = createAndStartVM(secondVMI)

			By("Verifying the adopted state is visible in the second VM")
			checkTPMSecret(secondVMI)
			checkEFIVar(secondVMI)
		})

		It("marks the VM as ErrorPvcNotFound when the source PVC does not exist", func() {
			vmi := newPersistentFedora()
			vmi.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				Source: &v1.VirtualMachineStateSource{Name: "does-not-exist"},
			}
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
				if err != nil && !k8serrors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			By("Expecting the VM to report the missing PVC")
			Eventually(func() v1.VirtualMachinePrintableStatus {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedVM.Status.PrintableStatus
			}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(Equal(v1.VirtualMachineStatusPvcNotFound))
		})

		It("marks the VM as ErrorVMStateInUse when the source PVC is held by a running VM", func() {
			sc, found := libstorage.GetRWOFileSystemStorageClass()
			if !found {
				Skip("Skipping test, no RWO filesystem storage class available")
			}

			By("Starting a first VM that keeps its VirtualMachineState PVC in use")
			firstVMI := newPersistentFedora()
			firstVMI.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				VolumeClaimTemplate: vmStateTemplate(sc),
			}
			firstVM, _ := createAndStartVM(firstVMI)

			pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(firstVM.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: backendstorage.VMStateOwnerLabel,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pvcs.Items).To(HaveLen(1))
			pvcName := pvcs.Items[0].Name

			By("Starting a second VM that references the still-in-use PVC")
			secondVMI := newPersistentFedora()
			secondVMI.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				Source: &v1.VirtualMachineStateSource{Name: pvcName},
			}
			secondVM := libvmi.NewVirtualMachine(secondVMI, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			secondVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(secondVM)).Create(context.Background(), secondVM, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				err := virtClient.VirtualMachine(secondVM.Namespace).Delete(context.Background(), secondVM.Name, metav1.DeleteOptions{})
				if err != nil && !k8serrors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			By("Expecting the second VM to report the PVC as in use")
			Eventually(func() v1.VirtualMachinePrintableStatus {
				updatedVM, err := virtClient.VirtualMachine(secondVM.Namespace).Get(context.Background(), secondVM.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedVM.Status.PrintableStatus
			}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(Equal(v1.VirtualMachineStatusVMStateInUse))
		})

		It("lets a waiting VM start once the holder releases the PVC", func() {
			sc, found := libstorage.GetRWOFileSystemStorageClass()
			if !found {
				Skip("Skipping test, no RWO filesystem storage class available")
			}

			By("Starting a first VM that holds a VirtualMachineState PVC")
			firstVMI := newPersistentFedora()
			firstVMI.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				VolumeClaimTemplate: vmStateTemplate(sc),
			}
			firstVM, _ := createAndStartVM(firstVMI)
			pvcName := ownedPVCName(firstVM)

			By("Starting a second VM that references the still-in-use PVC")
			secondVMI := newPersistentFedora()
			secondVMI.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				Source: &v1.VirtualMachineStateSource{Name: pvcName},
			}
			secondVM := libvmi.NewVirtualMachine(secondVMI, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			secondVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(secondVM)).Create(context.Background(), secondVM, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				err := virtClient.VirtualMachine(secondVM.Namespace).Delete(context.Background(), secondVM.Name, metav1.DeleteOptions{})
				if err != nil && !k8serrors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			By("Waiting for the second VM to report the PVC as in use")
			Eventually(func() v1.VirtualMachinePrintableStatus {
				updatedVM, err := virtClient.VirtualMachine(secondVM.Namespace).Get(context.Background(), secondVM.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedVM.Status.PrintableStatus
			}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).Should(Equal(v1.VirtualMachineStatusVMStateInUse))

			By("Stopping the first VM to release the PVC")
			libvmops.StopVirtualMachine(firstVM)

			By("Verifying the second VM starts once the PVC is free")
			Eventually(matcher.ThisVM(secondVM)).WithTimeout(360 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		})
	})

	Context("adopting a legacy PVC that does not match the canonical layout", func() {
		It("normalizes a legacy backend-storage PVC and preserves its state", func() {
			if _, found := libstorage.GetVMStateStorageClass(); !found {
				Skip("Skipping test, no VirtualMachineState storage class configured")
			}

			By("Starting a VM the implicit way to produce a legacy-layout PVC")
			legacyVMI := newPersistentFedora()
			legacyVM, legacyVMI := createAndStartVM(legacyVMI)

			By("Writing state into the legacy VM")
			storeTPMSecret(legacyVMI)
			storeEFIVar(legacyVMI)

			// The implicit backend-storage PVC is discoverable by the legacy label.
			pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(legacyVM.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", backendstorage.PVCPrefix, legacyVM.Name),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pvcs.Items).To(HaveLen(1))
			legacyPVCName := pvcs.Items[0].Name

			By("Stopping the legacy VM to release its PVC")
			libvmops.StopVirtualMachine(legacyVM)

			By("Adopting the legacy PVC through the declarative source field")
			adoptingVMI := newPersistentFedora()
			adoptingVMI.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				Source: &v1.VirtualMachineStateSource{Name: legacyPVCName},
			}
			_, adoptingVMI = createAndStartVM(adoptingVMI)

			By("Verifying the normalized state is intact in the adopting VM")
			checkTPMSecret(adoptingVMI)
			checkEFIVar(adoptingVMI)
		})

		It("boots with fresh state when the source PVC has no recognizable layout", func() {
			sc, found := libstorage.GetRWOFileSystemStorageClass()
			if !found {
				Skip("Skipping test, no RWO filesystem storage class available")
			}

			By("Pre-creating an empty PVC with no VirtualMachineState layout")
			pvc := libstorage.CreateFSPVC("vmstate-unrecognized", testsuite.GetTestNamespace(nil), "128Mi", libstorage.WithStorageClass(sc))
			DeferCleanup(func() {
				err := virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.Background(), pvc.Name, metav1.DeleteOptions{})
				if err != nil && !k8serrors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			By("Adopting the empty PVC and booting with a fresh TPM")
			vmi := newPersistentFedora()
			vmi.Spec.VirtualMachineState = &v1.VirtualMachineStateSpec{
				Source: &v1.VirtualMachineStateSource{Name: pvc.Name},
			}
			_, vmi = createAndStartVM(vmi)

			By("Verifying the VM initializes fresh state on the adopted PVC")
			storeTPMSecret(vmi)
			checkTPMSecret(vmi)
		})
	})
}))
