package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/api/core"

	"kubevirt.io/kubevirt/tests/util"

	expect "github.com/google/goexpect"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
)

var _ = SIGDescribe("[Serial]VirtualMachineRestore Tests", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	groupName := "kubevirt.io"

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	createRestoreDef := func(vm *v1.VirtualMachine, snapshotName string) *snapshotv1.VirtualMachineRestore {
		return &snapshotv1.VirtualMachineRestore{
			ObjectMeta: metav1.ObjectMeta{
				Name: "restore-" + vm.Name,
			},
			Spec: snapshotv1.VirtualMachineRestoreSpec{
				Target: corev1.TypedLocalObjectReference{
					APIGroup: &groupName,
					Kind:     "VirtualMachine",
					Name:     vm.Name,
				},
				VirtualMachineSnapshotName: snapshotName,
			},
		}
	}

	createSnapshot := func(vm *v1.VirtualMachine) *snapshotv1.VirtualMachineSnapshot {
		var err error
		s := &snapshotv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name: "snapshot-" + vm.Name,
			},
			Spec: snapshotv1.VirtualMachineSnapshotSpec{
				Source: corev1.TypedLocalObjectReference{
					APIGroup: &groupName,
					Kind:     "VirtualMachine",
					Name:     vm.Name,
				},
			},
		}

		s, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Create(context.Background(), s, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			s, err = virtClient.VirtualMachineSnapshot(s.Namespace).Get(context.Background(), s.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return s.Status != nil && s.Status.ReadyToUse != nil && *s.Status.ReadyToUse && vm.Status.SnapshotInProgress == nil
		}, 180*time.Second, time.Second).Should(BeTrue())

		return s
	}

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

	waitRestoreComplete := func(r *snapshotv1.VirtualMachineRestore, vm *v1.VirtualMachine) *snapshotv1.VirtualMachineRestore {
		var err error
		Eventually(func() bool {
			r, err = virtClient.VirtualMachineRestore(r.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return r.Status != nil && r.Status.Complete != nil && *r.Status.Complete
		}, 180*time.Second, time.Second).Should(BeTrue())
		Expect(r.OwnerReferences).To(HaveLen(1))
		Expect(r.OwnerReferences[0].APIVersion).To(Equal(v1.GroupVersion.String()))
		Expect(r.OwnerReferences[0].Kind).To(Equal("VirtualMachine"))
		Expect(r.OwnerReferences[0].Name).To(Equal(vm.Name))
		Expect(r.OwnerReferences[0].UID).To(Equal(vm.UID))
		Expect(r.Status.RestoreTime).ToNot(BeNil())
		Expect(r.Status.Conditions).To(HaveLen(2))
		Expect(r.Status.Conditions[0].Type).To(Equal(snapshotv1.ConditionProgressing))
		Expect(r.Status.Conditions[0].Status).To(Equal(corev1.ConditionFalse))
		Expect(r.Status.Conditions[1].Type).To(Equal(snapshotv1.ConditionReady))
		Expect(r.Status.Conditions[1].Status).To(Equal(corev1.ConditionTrue))
		return r
	}

	waitDVReady := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
		Eventually(func() bool {
			var err error
			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return dv.Status.Phase == cdiv1.Succeeded
		}, 180*time.Second, time.Second).Should(BeTrue())
		return dv
	}

	waitPVCReady := func(pvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
		Eventually(func() bool {
			var err error
			pvc, err = virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pvc.Annotations["cdi.kubevirt.io/storage.pod.phase"] == string(corev1.PodSucceeded)
		}, 180*time.Second, time.Second).Should(BeTrue())
		return pvc
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

	deleteVM := func(vm *v1.VirtualMachine) {
		waitDeleted(func() error {
			return virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
		})
	}

	deleteSnapshot := func(s *snapshotv1.VirtualMachineSnapshot) {
		waitDeleted(func() error {
			return virtClient.VirtualMachineSnapshot(s.Namespace).Delete(context.Background(), s.Name, metav1.DeleteOptions{})
		})
	}

	deleteRestore := func(r *snapshotv1.VirtualMachineRestore) {
		waitDeleted(func() error {
			return virtClient.VirtualMachineRestore(r.Namespace).Delete(context.Background(), r.Name, metav1.DeleteOptions{})
		})
	}

	deleteWebhook := func(wh *admissionregistrationv1.ValidatingWebhookConfiguration) {
		waitDeleted(func() error {
			return virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), wh.Name, metav1.DeleteOptions{})
		})
	}

	Context("With simple VM", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "#!/bin/bash\necho 'hello'\n")
			vm = tests.NewRandomVirtualMachine(vmi, false)
		})

		AfterEach(func() {
			deleteVM(vm)
		})

		Context("and no snapshot", func() {
			It("[test_id:5255]should reject restore", func() {
				vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				restore := createRestoreDef(vm, "foobar")

				_, err := virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("VirtualMachineSnapshot \"foobar\" does not exist"))
			})
		})

		Context("with run strategy and snapshot", func() {
			var err error
			var snapshot *snapshotv1.VirtualMachineSnapshot

			runStrategyHalted := v1.RunStrategyHalted

			AfterEach(func() {
				deleteSnapshot(snapshot)
			})

			It("should successfully restore", func() {
				vm.Spec.Running = nil
				vm.Spec.RunStrategy = &runStrategyHalted
				vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				snapshot = createSnapshot(vm)

				restore := createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm)
				Expect(restore.Status.Restores).To(HaveLen(0))
				Expect(restore.Status.DeletedDataVolumes).To(HaveLen(0))
			})
		})

		Context("and good snapshot exists", func() {
			var err error
			var snapshot *snapshotv1.VirtualMachineSnapshot
			var webhook *admissionregistrationv1.ValidatingWebhookConfiguration

			BeforeEach(func() {
				vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				snapshot = createSnapshot(vm)
			})

			AfterEach(func() {
				deleteSnapshot(snapshot)
				if webhook != nil {
					deleteWebhook(webhook)
				}
			})

			It("[test_id:5256]should successfully restore", func() {
				var origSpec *v1.VirtualMachineSpec

				Eventually(func() bool {
					var updatedVM *v1.VirtualMachine
					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					if vm.Status.SnapshotInProgress != nil {
						return false
					}

					origSpec = vm.Spec.DeepCopy()
					Expect(origSpec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("128Mi")))

					vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory] = resource.MustParse("256Mi")
					updatedVM, err = virtClient.VirtualMachine(vm.Namespace).Update(vm)
					if errors.IsConflict(err) {
						return false
					}
					vm = updatedVM
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("256Mi")))
					return true
				}, 180*time.Second, time.Second).Should(BeTrue())

				restore := createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm)
				Expect(restore.Status.Restores).To(HaveLen(0))
				Expect(restore.Status.DeletedDataVolumes).To(HaveLen(0))

				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(vm.Spec).To(Equal(*origSpec))

				deleteRestore(restore)
			})

			It("[test_id:5257]should reject restore if VM running", func() {
				patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
				vm, err := virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore := createRestoreDef(vm, snapshot.Name)

				_, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("VirtualMachine %q is not stopped", vm.Name)))
			})

			It("[test_id:5258]should reject restore if another in progress", func() {
				fp := admissionregistrationv1.Fail
				sideEffectNone := admissionregistrationv1.SideEffectClassNone
				whPath := "/foobar"
				whName := "dummy-webhook-deny-vm-update.kubevirt.io"
				wh := &admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: "temp-webhook-deny-vm-update",
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{
						{
							Name:                    whName,
							AdmissionReviewVersions: []string{"v1", "v1beta1"},
							FailurePolicy:           &fp,
							SideEffects:             &sideEffectNone,
							Rules: []admissionregistrationv1.RuleWithOperations{{
								Operations: []admissionregistrationv1.OperationType{
									admissionregistrationv1.Update,
								},
								Rule: admissionregistrationv1.Rule{
									APIGroups:   []string{core.GroupName},
									APIVersions: v1.ApiSupportedWebhookVersions,
									Resources:   []string{"virtualmachines"},
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

				restore := createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					restore, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return restore.Status != nil &&
						len(restore.Status.Conditions) == 2 &&
						restore.Status.Conditions[0].Status == corev1.ConditionFalse &&
						restore.Status.Conditions[1].Status == corev1.ConditionFalse &&
						strings.Contains(restore.Status.Conditions[0].Reason, whName) &&
						strings.Contains(restore.Status.Conditions[1].Reason, whName)
				}, 180*time.Second, time.Second).Should(BeTrue())

				r2 := restore.DeepCopy()
				r2.ObjectMeta = metav1.ObjectMeta{
					Name: "dummy",
				}

				_, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), r2, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("VirtualMachineRestore %q in progress", restore.Name)))

				deleteWebhook(webhook)
				webhook = nil

				restore = waitRestoreComplete(restore, vm)

				r2, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), r2, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				r2 = waitRestoreComplete(r2, vm)

				deleteRestore(r2)
				deleteRestore(restore)
			})
		})
	})

	Context("[rook-ceph]", func() {
		Context("With a more complicated VM", func() {
			var (
				vm                   *v1.VirtualMachine
				vmi                  *v1.VirtualMachineInstance
				snapshot             *snapshotv1.VirtualMachineSnapshot
				restore              *snapshotv1.VirtualMachineRestore
				webhook              *admissionregistrationv1.ValidatingWebhookConfiguration
				snapshotStorageClass string
			)

			BeforeEach(func() {
				sc, err := getSnapshotStorageClass(virtClient)
				Expect(err).ToNot(HaveOccurred())

				if sc == "" {
					Skip("Skiping test, no VolumeSnapshot support")
				}

				snapshotStorageClass = sc
			})

			AfterEach(func() {
				if vm != nil {
					deleteVM(vm)
				}
				if snapshot != nil {
					deleteSnapshot(snapshot)
				}
				if restore != nil {
					deleteRestore(restore)
				}

				if webhook != nil {
					deleteWebhook(webhook)
				}
			})

			doRestore := func(device string, login console.LoginToFactory, onlineSnapshot bool) {
				By("creating 'message with initial value")
				Expect(libnet.WithIPv6(login)(vmi)).To(Succeed())

				var batch []expect.Batcher
				if device != "" {
					batch = append(batch, []expect.Batcher{
						&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 %s\n", device)},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: "echo $?\n"},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: "sudo mkdir -p /test\n"},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: fmt.Sprintf("sudo mount %s /test \n", device)},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: "echo $?\n"},
						&expect.BExp{R: console.RetValue("0")},
					}...)
				}

				batch = append(batch, []expect.Batcher{
					&expect.BSnd{S: "sudo mkdir -p /test/data\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "sudo chmod a+w /test/data\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: fmt.Sprintf("echo '%s' > /test/data/message\n", vm.UID)},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /test/data/message\n"},
					&expect.BExp{R: string(vm.UID)},
					&expect.BSnd{S: "sync\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "sync\n"},
					&expect.BExp{R: console.PromptExpression},
				}...)

				Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())

				if !onlineSnapshot {
					By("Stopping VM")
					vm = tests.StopVirtualMachine(vm)
				}

				By("creating snapshot")
				snapshot = createSnapshot(vm)

				batch = nil
				if !onlineSnapshot {
					By("Starting VM")
					vm = tests.StartVirtualMachine(vm)
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					Expect(libnet.WithIPv6(login)(vmi)).To(Succeed())

					if device != "" {
						batch = append(batch, []expect.Batcher{
							&expect.BSnd{S: "sudo mkdir -p /test\n"},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: fmt.Sprintf("sudo mount %s /test \n", device)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: "echo $?\n"},
							&expect.BExp{R: console.RetValue("0")},
						}...)
					}
				}

				By("updating message")

				batch = append(batch, []expect.Batcher{
					&expect.BSnd{S: "sudo mkdir -p /test/data\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "sudo chmod a+w /test/data\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /test/data/message\n"},
					&expect.BExp{R: string(vm.UID)},
					&expect.BSnd{S: fmt.Sprintf("echo '%s' > /test/data/message\n", snapshot.UID)},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /test/data/message\n"},
					&expect.BExp{R: string(snapshot.UID)},
					&expect.BSnd{S: "sync\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "sync\n"},
					&expect.BExp{R: console.PromptExpression},
				}...)

				Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())

				By("Stopping VM")
				vm = tests.StopVirtualMachine(vm)

				By("Restoring VM")
				restore = createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm)
				Expect(restore.Status.Restores).To(HaveLen(1))

				vm = tests.StartVirtualMachine(vm)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying original file contents")
				Expect(libnet.WithIPv6(login)(vmi)).To(Succeed())

				batch = nil

				if device != "" {
					batch = append(batch, []expect.Batcher{
						&expect.BSnd{S: "sudo mkdir -p /test\n"},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: fmt.Sprintf("sudo mount %s /test \n", device)},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: "echo $?\n"},
						&expect.BExp{R: console.RetValue("0")},
					}...)
				}

				batch = append(batch, []expect.Batcher{
					&expect.BSnd{S: "sudo mkdir -p /test/data\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "sudo chmod a+w /test/data\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /test/data/message\n"},
					&expect.BExp{R: string(vm.UID)},
				}...)

				Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
			}

			It("[test_id:5259]should restore a vm multiple from the same snapshot", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					"#!/bin/bash\necho 'hello'\n",
					snapshotStorageClass,
				))

				By("Stopping VM")
				vm = tests.StopVirtualMachine(vm)

				By("creating snapshot")
				snapshot = createSnapshot(vm)

				for i := 0; i < 2; i++ {
					By(fmt.Sprintf("Restoring VM iteration %d", i))
					restore = createRestoreDef(vm, snapshot.Name)

					restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					restore = waitRestoreComplete(restore, vm)
					Expect(restore.Status.Restores).To(HaveLen(1))

					deleteRestore(restore)
					restore = nil
				}
			})

			It("[test_id:5260]should restore a vm that boots from a datavolumetemplate", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					"#!/bin/bash\necho 'hello'\n",
					snapshotStorageClass,
				))

				originalDVName := vm.Spec.DataVolumeTemplates[0].Name

				doRestore("", console.LoginToCirros, false)
				Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
				Expect(restore.Status.DeletedDataVolumes).To(ContainElement(originalDVName))

				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), originalDVName, metav1.GetOptions{})
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("[test_id:5261]should restore a vm that boots from a datavolume (not template)", func() {
				vm = tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					"#!/bin/bash\necho 'hello'\n",
					snapshotStorageClass,
				)

				var err error
				dvt := &vm.Spec.DataVolumeTemplates[0]

				dv := &cdiv1.DataVolume{}
				dv.ObjectMeta = *dvt.ObjectMeta.DeepCopy()
				dv.Spec = *dvt.Spec.DeepCopy()

				originalPVCName := dv.Name
				vm.Spec.DataVolumeTemplates = nil

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				dv = waitDVReady(dv)

				vm, vmi = createAndStartVM(vm)

				doRestore("", console.LoginToCirros, false)

				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())

				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), originalPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, v := range vm.Spec.Template.Spec.Volumes {
					if v.PersistentVolumeClaim != nil {
						Expect(v.PersistentVolumeClaim.ClaimName).ToNot(Equal(originalPVCName))
						pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), v.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						Expect(pvc.OwnerReferences[0].APIVersion).To(Equal(v1.GroupVersion.String()))
						Expect(pvc.OwnerReferences[0].Kind).To(Equal("VirtualMachine"))
						Expect(pvc.OwnerReferences[0].Name).To(Equal(vm.Name))
						Expect(pvc.OwnerReferences[0].UID).To(Equal(vm.UID))
					}
				}
			})

			It("[test_id:5262]should restore a vm that boots from a PVC", func() {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				pvc := &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "restore-pvc-" + rand.String(12),
						Namespace: util.NamespaceTestDefault,
						Annotations: map[string]string{
							"cdi.kubevirt.io/storage.import.source":   "registry",
							"cdi.kubevirt.io/storage.import.endpoint": cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"storage": quantity,
							},
						},
						StorageClassName: &snapshotStorageClass,
					},
				}

				pvc, err = virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				pvc = waitPVCReady(pvc)

				originalPVCName := pvc.Name

				vmi = tests.NewRandomVMIWithPVCAndUserData(pvc.Name, "#!/bin/bash\necho 'hello'\n")
				vm = tests.NewRandomVirtualMachine(vmi, false)

				vm, vmi = createAndStartVM(vm)

				doRestore("", console.LoginToCirros, false)

				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
				_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), originalPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, v := range vm.Spec.Template.Spec.Volumes {
					if v.PersistentVolumeClaim != nil {
						Expect(v.PersistentVolumeClaim.ClaimName).ToNot(Equal(originalPVCName))
						pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), v.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						Expect(pvc.OwnerReferences[0].APIVersion).To(Equal(v1.GroupVersion.String()))
						Expect(pvc.OwnerReferences[0].Kind).To(Equal("VirtualMachine"))
						Expect(pvc.OwnerReferences[0].Name).To(Equal(vm.Name))
						Expect(pvc.OwnerReferences[0].UID).To(Equal(vm.UID))
					}
				}
			})

			It("[test_id:5263]should restore a vm with containerdisk and blank datavolume", func() {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(
					cd.ContainerDiskFor(cd.ContainerDiskCirros),
					"#!/bin/bash\necho 'hello'\n",
				)
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

				doRestore("/dev/vdc", console.LoginToCirros, false)

				Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
				Expect(restore.Status.DeletedDataVolumes).To(ContainElement(dvName))
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), dvName, metav1.GetOptions{})
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("should reject vm start if restore in progress", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					"#!/bin/bash\necho 'hello'\n",
					snapshotStorageClass,
				))

				By("Stopping VM")
				vm = tests.StopVirtualMachine(vm)

				By("creating snapshot")
				snapshot = createSnapshot(vm)

				fp := admissionregistrationv1.Fail
				sideEffectNone := admissionregistrationv1.SideEffectClassNone
				whPath := "/foobar"
				whName := "dummy-webhook-deny-pvc-create.kubevirt.io"
				wh := &admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: "temp-webhook-deny-pvc-create",
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
									APIGroups:   []string{""},
									APIVersions: v1.ApiSupportedWebhookVersions,
									Resources:   []string{"persistentvolumeclaims"},
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

				restore := createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					restore, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return restore.Status != nil &&
						len(restore.Status.Conditions) == 2 &&
						restore.Status.Conditions[0].Status == corev1.ConditionFalse &&
						restore.Status.Conditions[1].Status == corev1.ConditionFalse &&
						strings.Contains(restore.Status.Conditions[0].Reason, whName) &&
						strings.Contains(restore.Status.Conditions[1].Reason, whName) &&
						updatedVM.Status.RestoreInProgress != nil &&
						*updatedVM.Status.RestoreInProgress == restore.Name
				}, 180*time.Second, 3*time.Second).Should(BeTrue())

				running := true
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				updatedVM.Spec.Running = &running
				_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Cannot start VM until restore %q completes", restore.Name)))

				deleteWebhook(webhook)
				webhook = nil

				restore = waitRestoreComplete(restore, vm)

				Eventually(func() bool {
					updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVM.Status.RestoreInProgress == nil
				}, 30*time.Second, 3*time.Second).Should(BeTrue())

				vm = tests.StartVirtualMachine(vm)
				deleteRestore(restore)
			})

			It("[test_id:6053]should restore a vm from an online snapshot", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					"#!/bin/bash\necho 'hello'\n",
					snapshotStorageClass,
				))

				doRestore("", console.LoginToCirros, true)

			})

			It("[test_id:6766]should restore a vm from an online snapshot with guest agent", func() {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi = tests.NewRandomFedoraVMIWithGuestAgent()
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

				doRestore("/dev/vdc", console.LoginToFedora, true)

			})

			It("[test_id:6836]should restore an online vm snapshot that boots from a datavolumetemplate with guest agent", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				tests.WaitAgentConnected(virtClient, vmi)
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed())

				originalDVName := vm.Spec.DataVolumeTemplates[0].Name

				doRestore("", console.LoginToFedora, true)
				Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
				Expect(restore.Status.DeletedDataVolumes).To(ContainElement(originalDVName))

				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), originalDVName, metav1.GetOptions{})
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("should restore vm spec at startup without new changes", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				tests.WaitAgentConnected(virtClient, vmi)
				Expect(libnet.WithIPv6(console.LoginToFedora)(vmi)).To(Succeed())

				By("Updating the VM template spec")
				initialMemory := vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]
				newMemory := resource.MustParse("2Gi")
				Expect(newMemory).ToNot(Equal(initialMemory))

				newVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				updatedVM := newVM.DeepCopy()
				updatedVM.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{
					corev1.ResourceMemory: newMemory,
				}
				updatedVM, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
				Expect(err).ToNot(HaveOccurred())

				By("creating snapshot")
				snapshot = createSnapshot(vm)

				newVM = tests.StopVirtualMachine(updatedVM)
				newVM = tests.StartVirtualMachine(newVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(newMemory))

				newVM = tests.StopVirtualMachine(newVM)

				By("Restoring VM")
				restore = createRestoreDef(newVM, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				restore = waitRestoreComplete(restore, newVM)
				Expect(restore.Status.Restores).To(HaveLen(1))

				newVM = tests.StartVirtualMachine(newVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(initialMemory))
			})

			It("[test_id:7425]should restore vm with hot plug disks", func() {
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

				doRestore("", console.LoginToFedora, true)

				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(vmi.Spec.Volumes)).To(Equal(2))
				foundHotPlug := false
				foundTempHotPlug := false
				for _, volume := range vmi.Spec.Volumes {
					if volume.Name == persistVolName {
						foundHotPlug = true
					} else if volume.Name == tempVolName {
						foundTempHotPlug = true
					}
				}
				Expect(foundHotPlug).To(BeTrue())
				Expect(foundTempHotPlug).To(BeFalse())
			})
		})
	})
})
