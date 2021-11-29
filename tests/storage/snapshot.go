package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("[Serial]VirtualMachineSnapshot Tests", func() {

	var (
		err        error
		virtClient kubecli.KubevirtClient
		vm         *v1.VirtualMachine
		snapshot   *snapshotv1.VirtualMachineSnapshot
		webhook    *admissionregistrationv1.ValidatingWebhookConfiguration
	)

	groupName := "kubevirt.io"

	newSnapshot := func() *snapshotv1.VirtualMachineSnapshot {
		return &snapshotv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-" + vm.Name,
				Namespace: vm.Namespace,
			},
			Spec: snapshotv1.VirtualMachineSnapshotSpec{
				Source: corev1.TypedLocalObjectReference{
					APIGroup: &groupName,
					Kind:     "VirtualMachine",
					Name:     vm.Name,
				},
			},
		}
	}

	waitSnapshotReady := func() {
		Eventually(func() bool {
			snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse
		}, 180*time.Second, time.Second).Should(BeTrue())
	}

	waitDeleted := func(deleteFunc func() error) {
		Eventually(func() bool {
			err := deleteFunc()
			if errors.IsNotFound(err) {
				return true
			}
			Expect(err).ToNot(HaveOccurred())
			return false
		}, 180*time.Second, time.Second).Should(BeTrue())
	}

	deleteVM := func() {
		waitDeleted(func() error {
			return virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
		})
		vm = nil
	}

	deleteSnapshot := func() {
		waitDeleted(func() error {
			return virtClient.VirtualMachineSnapshot(snapshot.Namespace).Delete(context.Background(), snapshot.Name, metav1.DeleteOptions{})
		})
		snapshot = nil
	}

	deleteWebhook := func() {
		waitDeleted(func() error {
			return virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), webhook.Name, metav1.DeleteOptions{})
		})
		webhook = nil
	}

	createDenyVolumeSnapshotCreateWebhook := func() {
		fp := admissionregistrationv1.Fail
		sideEffectNone := admissionregistrationv1.SideEffectClassNone
		whPath := "/foobar"
		whName := "dummy-webhook-deny-volume-snapshot-create.kubevirt.io"
		wh := &admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "temp-webhook-deny-volume-snapshot-create",
			},
			Webhooks: []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    whName,
					AdmissionReviewVersions: []string{"v1", "v1beta1"},
					FailurePolicy:           &fp,
					SideEffects:             &sideEffectNone,
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{vsv1beta1.GroupName},
							APIVersions: []string{vsv1beta1.SchemeGroupVersion.Version},
							Resources:   []string{"volumesnapshots"},
						},
					}},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: util.NamespaceTestDefault,
							Name:      "nonexistant",
							Path:      &whPath,
						},
					},
				},
			},
		}
		wh, err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), wh, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		webhook = wh
	}

	checkOnlineSnapshotExpectedContentSource := func(vm *v1.VirtualMachine, contentName string, expectVolumeBackups bool) {
		content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi.Spec.Volumes = vm.Spec.Template.Spec.Volumes
		vmi.Spec.Domain.Devices.Disks = vm.Spec.Template.Spec.Domain.Devices.Disks
		vm.Spec.Template.Spec = vmi.Spec

		Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
		Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
		if expectVolumeBackups {
			Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))
		} else {
			Expect(content.Spec.VolumeBackups).Should(BeEmpty())
		}
	}

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	AfterEach(func() {
		if vm != nil {
			deleteVM()
		}
		if snapshot != nil {
			deleteSnapshot()
		}
		if webhook != nil {
			deleteWebhook()
		}
	})

	Context("With simple VM", func() {
		BeforeEach(func() {
			var err error
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "#!/bin/bash\necho 'hello'\n")
			vm = tests.NewRandomVirtualMachine(vmi, false)
			vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		createAndVerifyVMSnapshot := func(vm *v1.VirtualMachine) {
			snapshot = newSnapshot()

			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitSnapshotReady()

			Expect(snapshot.Status.SourceUID).ToNot(BeNil())
			Expect(*snapshot.Status.SourceUID).To(Equal(vm.UID))

			contentName := *snapshot.Status.VirtualMachineSnapshotContentName
			if *vm.Spec.Running {
				expectedIndications := []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotNoGuestAgentIndication}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))
				checkOnlineSnapshotExpectedContentSource(vm, contentName, false)
			} else {
				Expect(snapshot.Status.Indications).To(BeEmpty())
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.VolumeBackups).To(BeEmpty())
			}
		}

		It("[test_id:4609]should successfully create a snapshot", func() {
			createAndVerifyVMSnapshot(vm)
		})

		It("[test_id:4610]create a snapshot when VM is running should succeed", func() {
			patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(*vm.Spec.Running).Should(BeTrue())

			createAndVerifyVMSnapshot(vm)
		})

		It("VM should contain snapshot status for all volumes", func() {
			patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
			vm, err := virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				vm2, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("VM Statuses: %+v", vm2.Status))
				return len(vm2.Status.VolumeSnapshotStatuses) == 2 &&
					vm2.Status.VolumeSnapshotStatuses[0].Name == "disk0" &&
					vm2.Status.VolumeSnapshotStatuses[1].Name == "disk1"
			}, 180*time.Second, time.Second).Should(BeTrue())
		})
	})

	Context("[rook-ceph]", func() {
		Context("With online vm snapshot", func() {
			var (
				snapshotStorageClass string
			)
			const VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION = "pre.hook.backup.velero.io/container"
			const VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION = "pre.hook.backup.velero.io/command"
			const VELERO_POSTBACKUP_HOOK_CONTAINER_ANNOTATION = "post.hook.backup.velero.io/container"
			const VELERO_POSTBACKUP_HOOK_COMMAND_ANNOTATION = "post.hook.backup.velero.io/command"

			BeforeEach(func() {
				sc, err := getSnapshotStorageClass(virtClient)
				Expect(err).ToNot(HaveOccurred())

				if sc == "" {
					Skip("Skiping test, no VolumeSnapshot support")
				}

				snapshotStorageClass = sc
			})

			createAndStartVM := func(vm *v1.VirtualMachine) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
				var vmi *v1.VirtualMachineInstance
				t := true
				vm.Spec.Running = &t
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 180*time.Second, time.Second).Should(BeTrue())

				return vm, vmi
			}

			checkVMFreeze := func(snapshot *snapshotv1.VirtualMachineSnapshot, vmi *v1.VirtualMachineInstance, hasGuestAgent, shouldFreeze bool) {
				var expectedIndications []snapshotv1.Indication
				if hasGuestAgent {
					expectedIndications = []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotGuestAgentIndication}
				} else {
					expectedIndications = []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotNoGuestAgentIndication}
				}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))

				conditionsLength := 2
				Expect(snapshot.Status.Conditions).To(HaveLen(conditionsLength))
				Expect(snapshot.Status.Conditions[0].Type).To(Equal(snapshotv1.ConditionProgressing))
				Expect(snapshot.Status.Conditions[0].Status).To(Equal(corev1.ConditionFalse))
				Expect(snapshot.Status.Conditions[1].Type).To(Equal(snapshotv1.ConditionReady))
				Expect(snapshot.Status.Conditions[1].Status).To(Equal(corev1.ConditionTrue))

				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed())
				journalctlCheck := "journalctl --file /var/log/journal/*/system.journal"
				expectedFreezeOutput := "executing fsfreeze hook with arg 'freeze'"
				expectedThawOutput := "executing fsfreeze hook with arg 'thaw'"
				if hasGuestAgent {
					if shouldFreeze {
						Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
							&expect.BSnd{S: fmt.Sprintf("%s | grep \"%s\"\n", journalctlCheck, expectedFreezeOutput)},
							&expect.BExp{R: fmt.Sprintf(".*qemu-ga.*%s.*", expectedFreezeOutput)},
							&expect.BSnd{S: "echo $?\n"},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: fmt.Sprintf("%s | grep \"%s\"\n", journalctlCheck, expectedThawOutput)},
							&expect.BExp{R: fmt.Sprintf(".*qemu-ga.*%s.*", expectedThawOutput)},
							&expect.BSnd{S: "echo $?\n"},
							&expect.BExp{R: console.RetValue("0")},
						}, 30)).To(Succeed())
					} else {
						Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
							&expect.BSnd{S: fmt.Sprintf("%s | grep \"%s\"\n", journalctlCheck, expectedFreezeOutput)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: "echo $?\n"},
							&expect.BExp{R: console.RetValue("1")},
						}, 30)).To(Succeed())
					}
				}
			}

			checkContentSourceAndMemory := func(vm *v1.VirtualMachine, contentName string, expectedMemory resource.Quantity) {
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				contentSourceSpec := content.Spec.Source.VirtualMachine.Spec
				snapshotSourceMemory := contentSourceSpec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory]
				Expect(snapshotSourceMemory).To(Equal(expectedMemory))
				checkOnlineSnapshotExpectedContentSource(vm, contentName, true)
			}

			callVeleroHook := func(vmi *v1.VirtualMachineInstance, annoContainer, annoCommand string) error {
				pod := tests.GetPodByVirtualMachineInstance(vmi)

				command := pod.Annotations[annoCommand]
				command = strings.Trim(command, "[]")
				commandSlice := []string{}
				for _, c := range strings.Split(command, ",") {
					commandSlice = append(commandSlice, strings.Trim(c, "\" "))
				}
				virtClient, err := kubecli.GetKubevirtClient()
				if err != nil {
					return err
				}
				_, _, err = tests.ExecuteCommandOnPodV2(virtClient, pod, pod.Annotations[annoContainer], commandSlice)
				return err
			}

			It("[test_id:6767]with volumes and guest agent available", func() {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Namespace = util.NamespaceTestDefault
				vm = tests.NewRandomVirtualMachine(vmi, false)
				dvName := "dv-" + vm.Name
				vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: dvName,
						},
						Spec: cdiv1.DataVolumeSpec{
							Source: &cdiv1.DataVolumeSource{
								Blank: &cdiv1.DataVolumeBlankImage{},
							},
							PVC: &corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": quantity,
									},
								},
								StorageClassName: &snapshotStorageClass,
							},
						},
					},
				}
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blank",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "blank",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv-" + vm.Name,
						},
					},
				})

				vm, vmi = createAndStartVM(vm)
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 300)
				tests.WaitAgentConnected(virtClient, vmi)

				initialMemory := vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]
				newMemory := resource.MustParse("1Gi")
				Expect(newMemory).ToNot(Equal(initialMemory))

				//update vm to make sure vm revision is saved in the snapshot
				By("Updating the VM template spec")
				newVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				updatedVM := newVM.DeepCopy()
				updatedVM.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{
					corev1.ResourceMemory: newMemory,
				}
				updatedVM, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
				Expect(err).ToNot(HaveOccurred())

				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()
				shouldFreeze := true
				checkVMFreeze(snapshot, vmi, true, shouldFreeze)

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				checkContentSourceAndMemory(vm.DeepCopy(), contentName, initialMemory)
			})

			It("[test_id:6768]with volumes and no guest agent available", func() {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi := tests.NewRandomFedoraVMI()
				vmi.Namespace = util.NamespaceTestDefault
				vm = tests.NewRandomVirtualMachine(vmi, false)
				dvName := "dv-" + vm.Name
				vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: dvName,
						},
						Spec: cdiv1.DataVolumeSpec{
							Source: &cdiv1.DataVolumeSource{
								Blank: &cdiv1.DataVolumeBlankImage{},
							},
							PVC: &corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": quantity,
									},
								},
								StorageClassName: &snapshotStorageClass,
							},
						},
					},
				}
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blank",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				})
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "blank",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv-" + vm.Name,
						},
					},
				})

				vm, vmi = createAndStartVM(vm)
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 300)

				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()
				shouldFreeze := false
				checkVMFreeze(snapshot, vmi, false, shouldFreeze)

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				checkOnlineSnapshotExpectedContentSource(vm.DeepCopy(), contentName, true)
			})

			It("[test_id:6769]without volumes with guest agent available", func() {
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Namespace = util.NamespaceTestDefault
				vm = tests.NewRandomVirtualMachine(vmi, false)

				vm, vmi = createAndStartVM(vm)
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 300)
				tests.WaitAgentConnected(virtClient, vmi)

				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()
				shouldFreeze := false
				checkVMFreeze(snapshot, vmi, true, shouldFreeze)

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				checkOnlineSnapshotExpectedContentSource(vm.DeepCopy(), contentName, false)
			})

			It("[test_id:6837]delete snapshot after freeze, expect vm unfreeze", func() {
				var vmi *v1.VirtualMachineInstance
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				tests.WaitAgentConnected(virtClient, vmi)
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed())

				createDenyVolumeSnapshotCreateWebhook()
				defer deleteWebhook()
				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVMI.Status.FSFreezeStatus == "frozen"
				}, time.Minute, 2*time.Second).Should(BeTrue())

				deleteSnapshot()
				Eventually(func() bool {
					updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVMI.Status.FSFreezeStatus == ""
				}, time.Minute, 2*time.Second).Should(BeTrue())
			})

			It("[test_id:6949]should unfreeze vm if snapshot fails when deadline exceeded", func() {
				var vmi *v1.VirtualMachineInstance
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				tests.WaitAgentConnected(virtClient, vmi)
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed())

				createDenyVolumeSnapshotCreateWebhook()
				snapshot = newSnapshot()
				snapshot.Spec.FailureDeadline = &metav1.Duration{Duration: 40 * time.Second}

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return snapshot.Status != nil &&
						snapshot.Status.Phase == snapshotv1.InProgress &&
						updatedVMI.Status.FSFreezeStatus == "frozen"
				}, 30*time.Second, 2*time.Second).Should(BeTrue())

				contentName := fmt.Sprintf("%s-%s", "vmsnapshot-content", snapshot.UID)
				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					_, contentErr := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
					return snapshot.Status != nil &&
						len(snapshot.Status.Conditions) == 3 &&
						snapshot.Status.Conditions[0].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[0].Reason, "snapshot deadline exceeded") &&
						snapshot.Status.Conditions[1].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[1].Reason, "Not ready") &&
						snapshot.Status.Conditions[2].Status == corev1.ConditionTrue &&
						snapshot.Status.Conditions[2].Type == snapshotv1.ConditionFailure &&
						strings.Contains(snapshot.Status.Conditions[2].Reason, "snapshot deadline exceeded") &&
						snapshot.Status.Phase == snapshotv1.Failed &&
						updatedVMI.Status.FSFreezeStatus == "" &&
						errors.IsNotFound(contentErr)
				}, time.Minute, 2*time.Second).Should(BeTrue())
			})

			It("[test_id:7472]should succeed online snapshot with hot plug disk", func() {
				var vmi *v1.VirtualMachineInstance
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				tests.WaitAgentConnected(virtClient, vmi)
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed())

				By("Add persistent hotplug disk")
				persistVolName := tests.AddVolumeAndVerify(virtClient, snapshotStorageClass, vm, false)
				By("Add temporary hotplug disk")
				tempVolName := tests.AddVolumeAndVerify(virtClient, snapshotStorageClass, vm, true)
				By("Create Snapshot")
				snapshot = newSnapshot()
				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return snapshot.Status != nil &&
						len(snapshot.Status.Conditions) == 2 &&
						snapshot.Status.Conditions[0].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[0].Reason, "Operation complete") &&
						snapshot.Status.Conditions[1].Status == corev1.ConditionTrue &&
						strings.Contains(snapshot.Status.Conditions[1].Reason, "Operation complete") &&
						snapshot.Status.Phase == snapshotv1.Succeeded
				}, 30*time.Second, 2*time.Second).Should(BeTrue())
				expectedIndications := []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotGuestAgentIndication}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))

				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				contentVMTemplate := content.Spec.Source.VirtualMachine.Spec.Template
				Expect(len(contentVMTemplate.Spec.Volumes)).To(Equal(2))
				foundHotPlug := false
				foundTempHotPlug := false
				for _, volume := range contentVMTemplate.Spec.Volumes {
					if volume.Name == persistVolName {
						foundHotPlug = true
					} else if volume.Name == tempVolName {
						foundTempHotPlug = true
					}
				}
				Expect(foundHotPlug).To(BeTrue())
				Expect(foundTempHotPlug).To(BeFalse())
			})

			It("Calling Velero hooks should freeze/unfreeze VM", func() {
				By("Creating VM")
				vmi := tests.NewRandomFedoraVMIWithGuestAgent()
				vmi.Namespace = util.NamespaceTestDefault
				vm = tests.NewRandomVirtualMachine(vmi, false)

				vm, vmi = createAndStartVM(vm)
				tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 300)
				tests.WaitAgentConnected(virtClient, vmi)

				By("Logging into Fedora")
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed())

				By("Calling Velero pre-backup hook")
				err := callVeleroHook(vmi, VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION, VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION)
				Expect(err).ToNot(HaveOccurred())

				By("Veryfing the VM was frozen")
				journalctlCheck := "journalctl --file /var/log/journal/*/system.journal"
				expectedFreezeOutput := "executing fsfreeze hook with arg 'freeze'"
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("%s | grep \"%s\"\n", journalctlCheck, expectedFreezeOutput)},
					&expect.BExp{R: fmt.Sprintf(".*qemu-ga.*%s.*", expectedFreezeOutput)},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 30)).To(Succeed())
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.FSFreezeStatus == "frozen"
				}, 180*time.Second, time.Second).Should(BeTrue())

				By("Calling Velero post-backup hook")
				err = callVeleroHook(vmi, VELERO_POSTBACKUP_HOOK_CONTAINER_ANNOTATION, VELERO_POSTBACKUP_HOOK_COMMAND_ANNOTATION)
				Expect(err).ToNot(HaveOccurred())

				By("Veryfing the VM was thawed")
				expectedThawOutput := "executing fsfreeze hook with arg 'thaw'"
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("%s | grep \"%s\"\n", journalctlCheck, expectedThawOutput)},
					&expect.BExp{R: fmt.Sprintf(".*qemu-ga.*%s.*", expectedThawOutput)},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 30)).To(Succeed())
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.FSFreezeStatus == ""
				}, 180*time.Second, time.Second).Should(BeTrue())
			})
		})

		Context("With more complicated VM", func() {
			BeforeEach(func() {
				sc, err := getSnapshotStorageClass(virtClient)
				Expect(err).ToNot(HaveOccurred())

				if sc == "" {
					Skip("Skiping test, no VolumeSnapshot support")
				}

				running := false
				vm = tests.NewRandomVMWithDataVolumeInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
					util.NamespaceTestDefault,
					sc,
				)
				vm.Spec.Running = &running

				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					Eventually(func() bool {
						dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), dvt.Name, metav1.GetOptions{})
						if errors.IsNotFound(err) {
							return false
						}
						Expect(err).ToNot(HaveOccurred())
						Expect(dv.Status.Phase).ShouldNot(Equal(cdiv1.Failed))
						return dv.Status.Phase == cdiv1.Succeeded
					}, 180*time.Second, time.Second).Should(BeTrue())
				}
			})

			It("[test_id:4611]should successfully create a snapshot", func() {
				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))

				for _, vol := range vm.Spec.Template.Spec.Volumes {
					if vol.DataVolume == nil {
						continue
					}
					found := false
					for _, vb := range content.Spec.VolumeBackups {
						if vol.DataVolume.Name == vb.PersistentVolumeClaim.Name {
							found = true
							Expect(vol.Name).To(Equal(vb.VolumeName))

							pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), vol.DataVolume.Name, metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							Expect(pvc.Spec).To(Equal(vb.PersistentVolumeClaim.Spec))

							Expect(vb.VolumeSnapshotName).ToNot(BeNil())
							vs, err := virtClient.
								KubernetesSnapshotClient().
								SnapshotV1beta1().
								VolumeSnapshots(vm.Namespace).
								Get(context.Background(), *vb.VolumeSnapshotName, metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							Expect(*vs.Spec.Source.PersistentVolumeClaimName).Should(Equal(vol.DataVolume.Name))
							Expect(vs.Status.Error).To(BeNil())
							Expect(*vs.Status.ReadyToUse).To(BeTrue())
						}
					}
					Expect(found).To(BeTrue())
				}
			})

			It("should successfully recreate status", func() {
				snapshot = newSnapshot()

				ss, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				ss, err = virtClient.VirtualMachineSnapshot(ss.Namespace).Get(context.Background(), ss.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				origStatus := ss.Status
				ss.Status = nil
				ss, err = virtClient.VirtualMachineSnapshot(ss.Namespace).Update(context.Background(), ss, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(ss.Status).To(BeNil())

				Eventually(func() bool {
					ss, err = virtClient.VirtualMachineSnapshot(ss.Namespace).Get(context.Background(), ss.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if ss.Status == nil {
						return false
					}
					if *ss.Status.SourceUID != *origStatus.SourceUID ||
						*ss.Status.VirtualMachineSnapshotContentName != *origStatus.VirtualMachineSnapshotContentName ||
						*ss.Status.CreationTime != *origStatus.CreationTime ||
						ss.Status.Phase != origStatus.Phase ||
						*ss.Status.ReadyToUse != *origStatus.ReadyToUse {
						return false
					}
					if len(ss.Status.Conditions) != len(origStatus.Conditions) {
						return false
					}
					for i, c := range ss.Status.Conditions {
						oc := origStatus.Conditions[i]
						if c.Type != oc.Type ||
							c.Status != oc.Status {
							return false
						}
					}
					if len(ss.Status.Indications) != len(origStatus.Indications) {
						return false
					}
					for i := range ss.Status.Indications {
						if ss.Status.Indications[i] != origStatus.Indications[i] {
							return false
						}
					}
					return true
				}, 180*time.Second, time.Second).Should(BeTrue())
			})

			It("VM should contain snapshot status for all volumes", func() {
				volumes := len(vm.Spec.Template.Spec.Volumes)
				Eventually(func() int {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return len(vm.Status.VolumeSnapshotStatuses)
				}, 180*time.Second, time.Second).Should(Equal(volumes))

				Eventually(func() bool {
					vm2, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					By(fmt.Sprintf("VM Statuses: %+v", vm2.Status))
					return len(vm2.Status.VolumeSnapshotStatuses) == 1 &&
						vm2.Status.VolumeSnapshotStatuses[0].Enabled == true
				}, 180*time.Second, time.Second).Should(BeTrue())
			})

			It("should error if VolumeSnapshot deleted", func() {
				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				cn := snapshot.Status.VirtualMachineSnapshotContentName
				Expect(cn).ToNot(BeNil())
				vmSnapshotContent, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *cn, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				vb := vmSnapshotContent.Spec.VolumeBackups[0]
				Expect(vb.VolumeSnapshotName).ToNot(BeNil())

				err = virtClient.KubernetesSnapshotClient().
					SnapshotV1beta1().
					VolumeSnapshots(vm.Namespace).
					Delete(context.Background(), *vb.VolumeSnapshotName, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return *snapshot.Status.ReadyToUse
				}, 180*time.Second, time.Second).Should(BeFalse())

				errStr := fmt.Sprintf("VolumeSnapshots (%s) missing", *vb.VolumeSnapshotName)
				Expect(snapshot.Status.Error).ToNot(BeNil())
				Expect(snapshot.Status.Error.Message).ToNot(Equal(errStr))
			})

			It("should not error if VolumeSnapshot has error", func() {
				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				cn := snapshot.Status.VirtualMachineSnapshotContentName
				Expect(cn).ToNot(BeNil())
				vmSnapshotContent, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *cn, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				vb := vmSnapshotContent.Spec.VolumeBackups[0]
				Expect(vb.VolumeSnapshotName).ToNot(BeNil())

				m := "bad stuff"
				Eventually(func() bool {
					vs, err := virtClient.KubernetesSnapshotClient().
						SnapshotV1beta1().
						VolumeSnapshots(vm.Namespace).
						Get(context.Background(), *vb.VolumeSnapshotName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vsc := vs.DeepCopy()
					t := metav1.Now()
					vsc.Status.Error = &vsv1beta1.VolumeSnapshotError{
						Time:    &t,
						Message: &m,
					}

					_, err = virtClient.KubernetesSnapshotClient().
						SnapshotV1beta1().
						VolumeSnapshots(vs.Namespace).
						UpdateStatus(context.Background(), vsc, metav1.UpdateOptions{})
					if errors.IsConflict(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return true
				}, 180*time.Second, time.Second).Should(BeTrue())

				Eventually(func() bool {
					vmSnapshotContent, err = virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *cn, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					vss := vmSnapshotContent.Status.VolumeSnapshotStatus[0]
					if vss.Error != nil {
						Expect(*vss.Error.Message).To(Equal(m))
						Expect(vmSnapshotContent.Status.Error).To(BeNil())
						Expect(*vmSnapshotContent.Status.ReadyToUse).To(BeTrue())
						return true
					}
					return false
				}, 180*time.Second, time.Second).Should(BeTrue())

				snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(snapshot.Status.Error).To(BeNil())
				Expect(*snapshot.Status.ReadyToUse).To(BeTrue())
			})

			It("[test_id:6952]snapshot change phase to in progress and succeeded and then should not fail", func() {
				createDenyVolumeSnapshotCreateWebhook()
				snapshot = newSnapshot()
				snapshot.Spec.FailureDeadline = &metav1.Duration{Duration: time.Minute}

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return snapshot.Status != nil &&
						len(snapshot.Status.Conditions) == 2 &&
						snapshot.Status.Conditions[0].Status == corev1.ConditionTrue &&
						strings.Contains(snapshot.Status.Conditions[0].Reason, "Source locked and operation in progress") &&
						snapshot.Status.Conditions[1].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[1].Reason, "Not ready") &&
						snapshot.Status.Phase == snapshotv1.InProgress
				}, 30*time.Second, 2*time.Second).Should(BeTrue())

				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedVM.Status.SnapshotInProgress).To(Equal(snapshot.Name))

				Expect(snapshot.Status.CreationTime).To(BeNil())

				contentName := fmt.Sprintf("%s-%s", "vmsnapshot-content", snapshot.UID)
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(content.Status).To(BeNil())

				deleteWebhook()

				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return snapshot.Status != nil &&
						len(snapshot.Status.Conditions) == 2 &&
						snapshot.Status.Conditions[0].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[0].Reason, "Operation complete") &&
						snapshot.Status.Conditions[1].Status == corev1.ConditionTrue &&
						strings.Contains(snapshot.Status.Conditions[1].Reason, "Operation complete") &&
						snapshot.Status.Phase == snapshotv1.Succeeded
				}, 30*time.Second, 2*time.Second).Should(BeTrue())

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				content, err = virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))

				// Sleep to pass the time of the deadline
				time.Sleep(time.Second)
				// If snapshot succeeded it should not change to failure when deadline exceeded
				Expect(snapshot.Status.Phase).To(Equal(snapshotv1.Succeeded))
			})

			It("[test_id:6838]snapshot should fail when deadline exceeded due to volume snapshots failure", func() {
				createDenyVolumeSnapshotCreateWebhook()
				snapshot = newSnapshot()
				snapshot.Spec.FailureDeadline = &metav1.Duration{Duration: 40 * time.Second}

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				contentName := fmt.Sprintf("%s-%s", "vmsnapshot-content", snapshot.UID)
				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					_, contentErr := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
					return snapshot.Status != nil &&
						len(snapshot.Status.Conditions) == 3 &&
						snapshot.Status.Conditions[0].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[0].Reason, "snapshot deadline exceeded") &&
						snapshot.Status.Conditions[1].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[1].Reason, "Not ready") &&
						snapshot.Status.Conditions[2].Status == corev1.ConditionTrue &&
						snapshot.Status.Conditions[2].Type == snapshotv1.ConditionFailure &&
						strings.Contains(snapshot.Status.Conditions[2].Reason, "snapshot deadline exceeded") &&
						snapshot.Status.Phase == snapshotv1.Failed &&
						errors.IsNotFound(contentErr)
				}, time.Minute, 2*time.Second).Should(BeTrue())

				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedVM.Status.SnapshotInProgress).To(BeNil())
				Expect(updatedVM.Finalizers).To(BeEmpty())

				Expect(snapshot.Status.CreationTime).To(BeNil())
			})
		})
	})
})

func getSnapshotStorageClass(client kubecli.KubevirtClient) (string, error) {
	crd, err := client.
		ExtensionsClient().
		ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.Background(), "volumesnapshotclasses.snapshot.storage.k8s.io", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}

		return "", err
	}

	hasV1beta1 := false
	for _, v := range crd.Spec.Versions {
		if v.Name == "v1beta1" && v.Served {
			hasV1beta1 = true
		}
	}

	if !hasV1beta1 {
		return "", nil
	}

	volumeSnapshotClasses, err := client.KubernetesSnapshotClient().SnapshotV1beta1().VolumeSnapshotClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(volumeSnapshotClasses.Items) == 0 {
		return "", nil
	}
	defaultSnapClass := volumeSnapshotClasses.Items[0]
	for _, snapClass := range volumeSnapshotClasses.Items {
		if snapClass.Annotations["snapshot.storage.kubernetes.io/is-default-class"] == "true" {
			defaultSnapClass = snapClass
		}
	}

	storageClasses, err := client.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, sc := range storageClasses.Items {
		if sc.Provisioner == defaultSnapClass.Driver {
			return sc.Name, nil
		}
	}

	return "", nil
}
