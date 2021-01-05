package tests_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("VirtualMachineRestore Tests", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	groupName := "kubevirt.io"

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)
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

		s, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Create(s)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			s, err = virtClient.VirtualMachineSnapshot(s.Namespace).Get(s.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return s.Status != nil && s.Status.ReadyToUse != nil && *s.Status.ReadyToUse
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
			r, err = virtClient.VirtualMachineRestore(r.Namespace).Get(r.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return r.Status != nil && r.Status.Complete != nil && *r.Status.Complete
		}, 180*time.Second, time.Second).Should(BeTrue())
		Expect(r.OwnerReferences).To(HaveLen(1))
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
			dv, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Get(dv.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return dv.Status.Phase == cdiv1.Succeeded
		}, 180*time.Second, time.Second).Should(BeTrue())
		return dv
	}

	waitPVCReady := func(pvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
		Eventually(func() bool {
			var err error
			pvc, err = virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(pvc.Name, metav1.GetOptions{})
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
			return virtClient.VirtualMachineSnapshot(s.Namespace).Delete(s.Name, &metav1.DeleteOptions{})
		})
	}

	deleteRestore := func(r *snapshotv1.VirtualMachineRestore) {
		waitDeleted(func() error {
			return virtClient.VirtualMachineRestore(r.Namespace).Delete(r.Name, &metav1.DeleteOptions{})
		})
	}

	deleteWebhook := func(wh *admissionregistrationv1beta1.ValidatingWebhookConfiguration) {
		waitDeleted(func() error {
			return virtClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(wh.Name, &metav1.DeleteOptions{})
		})
	}

	Context("With simple VM", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			var err error
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "#!/bin/bash\necho 'hello'\n")
			vm = tests.NewRandomVirtualMachine(vmi, false)
			vm, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			deleteVM(vm)
		})

		Context("and no snapshot", func() {
			It("should reject restore", func() {
				restore := createRestoreDef(vm, "foobar")

				_, err := virtClient.VirtualMachineRestore(vm.Namespace).Create(restore)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("VirtualMachineSnapshot \"foobar\" does not exist"))
			})
		})

		Context("and good snapshot exists", func() {
			var err error
			var snapshot *snapshotv1.VirtualMachineSnapshot
			var webhook *admissionregistrationv1beta1.ValidatingWebhookConfiguration

			BeforeEach(func() {
				snapshot = createSnapshot(vm)
			})

			AfterEach(func() {
				deleteSnapshot(snapshot)
				if webhook != nil {
					deleteWebhook(webhook)
				}
			})

			It("should successfully restore", func() {
				var origSpec *v1.VirtualMachineSpec

				Eventually(func() bool {
					var updatedVM *v1.VirtualMachine
					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					origSpec = vm.Spec.DeepCopy()
					Expect(origSpec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("64M")))

					vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory] = resource.MustParse("128M")
					updatedVM, err = virtClient.VirtualMachine(vm.Namespace).Update(vm)
					if errors.IsConflict(err) {
						return false
					}
					vm = updatedVM
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("128M")))
					return true
				}, 180*time.Second, time.Second).Should(BeTrue())

				restore := createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(restore)
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm)
				Expect(restore.Status.Restores).To(HaveLen(0))
				Expect(restore.Status.DeletedDataVolumes).To(HaveLen(0))

				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(vm.Spec).To(Equal(*origSpec))

				deleteRestore(restore)
			})

			It("should reject restore if VM running", func() {
				patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
				vm, err := virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch)
				Expect(err).ToNot(HaveOccurred())

				restore := createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(restore)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("VirtualMachine %q is running", vm.Name)))
			})

			It("should reject restore if another in progress", func() {
				fp := admissionregistrationv1beta1.Fail
				whPath := "/foobar"
				whName := "dummy-webhook-deny-vm-update.kubevirt.io"
				wh := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: "temp-webhook-deny-vm-update",
					},
					Webhooks: []admissionregistrationv1beta1.ValidatingWebhook{
						{
							Name:          whName,
							FailurePolicy: &fp,
							Rules: []admissionregistrationv1beta1.RuleWithOperations{{
								Operations: []admissionregistrationv1beta1.OperationType{
									admissionregistrationv1beta1.Update,
								},
								Rule: admissionregistrationv1beta1.Rule{
									APIGroups:   []string{v1.GroupName},
									APIVersions: v1.ApiSupportedWebhookVersions,
									Resources:   []string{"virtualmachines"},
								},
							}},
							ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
								Service: &admissionregistrationv1beta1.ServiceReference{
									Namespace: tests.NamespaceTestDefault,
									Name:      "nonexistant",
									Path:      &whPath,
								},
							},
						},
					},
				}
				wh, err := virtClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(wh)
				Expect(err).ToNot(HaveOccurred())
				webhook = wh

				restore := createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(restore)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					restore, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(restore.Name, metav1.GetOptions{})
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

				_, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(r2)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("VirtualMachineRestore %q in progress", restore.Name)))

				deleteWebhook(webhook)
				webhook = nil

				restore = waitRestoreComplete(restore, vm)

				r2, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(r2)
				Expect(err).ToNot(HaveOccurred())

				r2 = waitRestoreComplete(r2, vm)

				deleteRestore(r2)
				deleteRestore(restore)
			})
		})
	})

	Context("With a more complicated VM", func() {
		var (
			vm                   *v1.VirtualMachine
			vmi                  *v1.VirtualMachineInstance
			snapshot             *snapshotv1.VirtualMachineSnapshot
			restore              *snapshotv1.VirtualMachineRestore
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
		})

		doRestore := func(device string) {
			By("creating 'message with initial value")
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())

			var batch []expect.Batcher
			if device != "" {
				batch = append(batch, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 %s\n", device)},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
					&expect.BSnd{S: "sudo mkdir -p /test\n"},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: fmt.Sprintf("sudo mount %s /test \n", device)},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
				}...)
			}

			batch = append(batch, []expect.Batcher{
				&expect.BSnd{S: "sudo mkdir -p /test/data\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "sudo chmod a+w /test/data\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("echo '%s' > /test/data/message\n", vm.UID)},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "cat /test/data/message\n"},
				&expect.BExp{R: string(vm.UID)},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: "\\$ "},
			}...)

			res, err := expecter.ExpectBatch(batch, 20*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)
			expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			By("Stopping VM")
			vm = tests.StopVirtualMachine(vm)

			By("creating snapshot")
			snapshot = createSnapshot(vm)

			By("Starting VM")
			vm = tests.StartVirtualMachine(vm)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("updating message")
			expecter, err = tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())

			batch = nil

			if device != "" {
				batch = append(batch, []expect.Batcher{
					&expect.BSnd{S: "sudo mkdir -p /test\n"},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: fmt.Sprintf("sudo mount %s /test \n", device)},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
				}...)
			}

			batch = append(batch, []expect.Batcher{
				&expect.BSnd{S: "sudo mkdir -p /test/data\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "sudo chmod a+w /test/data\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "cat /test/data/message\n"},
				&expect.BExp{R: string(vm.UID)},
				&expect.BSnd{S: fmt.Sprintf("echo '%s' > /test/data/message\n", snapshot.UID)},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "cat /test/data/message\n"},
				&expect.BExp{R: string(snapshot.UID)},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: "\\$ "},
			}...)

			res, err = expecter.ExpectBatch(batch, 20*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)
			expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			By("Stopping VM")
			vm = tests.StopVirtualMachine(vm)

			By("Restoring VM")
			restore = createRestoreDef(vm, snapshot.Name)

			restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(restore)
			Expect(err).ToNot(HaveOccurred())

			restore = waitRestoreComplete(restore, vm)
			Expect(restore.Status.Restores).To(HaveLen(1))

			vm = tests.StartVirtualMachine(vm)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying original file contents")
			expecter, err = tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())

			batch = nil

			if device != "" {
				batch = append(batch, []expect.Batcher{
					&expect.BSnd{S: "sudo mkdir -p /test\n"},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: fmt.Sprintf("sudo mount %s /test \n", device)},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
				}...)
			}

			batch = append(batch, []expect.Batcher{
				&expect.BSnd{S: "sudo mkdir -p /test/data\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "sudo chmod a+w /test/data\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "cat /test/data/message\n"},
				&expect.BExp{R: string(vm.UID)},
			}...)

			res, err = expecter.ExpectBatch(batch, 20*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)
			expecter.Close()
			Expect(err).ToNot(HaveOccurred())
		}

		It("should restore a vm that boots from a datavolumetemplate", func() {
			vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
				tests.GetUrl(tests.CirrosHttpUrl),
				tests.NamespaceTestDefault,
				"#!/bin/bash\necho 'hello'\n",
				snapshotStorageClass,
			))

			originalDVName := vm.Spec.DataVolumeTemplates[0].Name

			doRestore("")

			Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
			Expect(restore.Status.DeletedDataVolumes).To(ContainElement(originalDVName))
			dvs, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).List(metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(dvs.Items).To(HaveLen(1))
			Expect(dvs.Items[0].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
		})

		It("should restore a vm that boots from a datavolume (not template)", func() {
			vm = tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
				tests.GetUrl(tests.CirrosHttpUrl),
				tests.NamespaceTestDefault,
				"#!/bin/bash\necho 'hello'\n",
				snapshotStorageClass,
			)

			var err error
			dv := &vm.Spec.DataVolumeTemplates[0]
			originalPVCName := dv.Name
			vm.Spec.DataVolumeTemplates = nil

			dv, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Create(dv)
			Expect(err).ToNot(HaveOccurred())
			dv = waitDVReady(dv)

			vm, vmi = createAndStartVM(vm)

			doRestore("")

			Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
			dvs, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).List(metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(dvs.Items).To(HaveLen(1))
			_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(originalPVCName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			for _, v := range vm.Spec.Template.Spec.Volumes {
				if v.PersistentVolumeClaim != nil {
					Expect(v.PersistentVolumeClaim.ClaimName).ToNot(Equal(originalPVCName))
					pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(v.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(pvc.OwnerReferences[0].Name).To(Equal(vm.Name))
					Expect(pvc.OwnerReferences[0].UID).To(Equal(vm.UID))
				}
			}
		})

		It("should restore a vm that boots from a PVC", func() {
			quantity, err := resource.ParseQuantity("1Gi")
			Expect(err).ToNot(HaveOccurred())
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "restore-pvc-" + rand.String(12),
					Namespace: tests.NamespaceTestDefault,
					Annotations: map[string]string{
						"cdi.kubevirt.io/storage.import.source":   "http",
						"cdi.kubevirt.io/storage.import.endpoint": tests.GetUrl(tests.CirrosHttpUrl),
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

			pvc, err = virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(pvc)
			Expect(err).ToNot(HaveOccurred())
			pvc = waitPVCReady(pvc)

			originalPVCName := pvc.Name

			vmi = tests.NewRandomVMIWithPVCAndUserData(pvc.Name, "#!/bin/bash\necho 'hello'\n")
			vm = tests.NewRandomVirtualMachine(vmi, false)

			vm, vmi = createAndStartVM(vm)

			doRestore("")

			Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
			_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(originalPVCName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			for _, v := range vm.Spec.Template.Spec.Volumes {
				if v.PersistentVolumeClaim != nil {
					Expect(v.PersistentVolumeClaim.ClaimName).ToNot(Equal(originalPVCName))
					pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(v.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(pvc.OwnerReferences[0].Name).To(Equal(vm.Name))
					Expect(pvc.OwnerReferences[0].UID).To(Equal(vm.UID))
				}
			}
		})

		It("should restore a vm with containerdisk and blank datavolume", func() {
			quantity, err := resource.ParseQuantity("1Gi")
			Expect(err).ToNot(HaveOccurred())
			vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(
				cd.ContainerDiskFor(cd.ContainerDiskCirros),
				"#!/bin/bash\necho 'hello'\n",
			)
			vm = tests.NewRandomVirtualMachine(vmi, false)
			dvName := "dv-" + vm.Name
			vm.Spec.DataVolumeTemplates = []cdiv1.DataVolume{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: dvName,
					},
					Spec: cdiv1.DataVolumeSpec{
						Source: cdiv1.DataVolumeSource{
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

			doRestore("/dev/vdc")

			Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
			Expect(restore.Status.DeletedDataVolumes).To(ContainElement(dvName))
			_, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Get(dvName, metav1.GetOptions{})
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})
