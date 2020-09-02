package tests_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
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

	waitRestoreComplete := func(r *snapshotv1.VirtualMachineRestore) *snapshotv1.VirtualMachineRestore {
		var err error
		Eventually(func() bool {
			r, err = virtClient.VirtualMachineRestore(r.Namespace).Get(r.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return r.Status != nil && r.Status.Complete != nil && *r.Status.Complete
		}, 180*time.Second, time.Second).Should(BeTrue())
		Expect(r.Status.RestoreTime).ToNot(BeNil())
		Expect(r.Status.Conditions).To(HaveLen(2))
		Expect(r.Status.Conditions[0].Type).To(Equal(snapshotv1.ConditionProgressing))
		Expect(r.Status.Conditions[0].Status).To(Equal(corev1.ConditionFalse))
		Expect(r.Status.Conditions[1].Type).To(Equal(snapshotv1.ConditionReady))
		Expect(r.Status.Conditions[1].Status).To(Equal(corev1.ConditionTrue))
		return r
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

			BeforeEach(func() {
				snapshot = createSnapshot(vm)
			})

			AfterEach(func() {
				deleteSnapshot(snapshot)
			})

			It("should successfully restore", func() {
				var origSpec *v1.VirtualMachineSpec

				Eventually(func() bool {
					var updatedVM *v1.VirtualMachine
					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					origSpec = vm.Spec.DeepCopy()
					Expect(origSpec.Template.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]).To(Equal(resource.MustParse("64M")))

					vm.Spec.Template.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128M")
					updatedVM, err = virtClient.VirtualMachine(vm.Namespace).Update(vm)
					if errors.IsConflict(err) {
						return false
					}
					vm = updatedVM
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.Template.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]).To(Equal(resource.MustParse("128M")))
					return true
				}, 180*time.Second, time.Second).Should(BeTrue())

				restore := createRestoreDef(vm, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(restore)
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore)
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

		doRestore := func() {
			By("creating 'hello.txt with initial value")
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())

			res, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "echo 'hello' > /home/cirros/hello.txt\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "cat /home/cirros/hello.txt\n"},
				&expect.BExp{R: "hello"},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: "\\$ "},
			}, 20*time.Second)
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

			By("updating hello.txt")
			expecter, err = tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())

			res, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "cat /home/cirros/hello.txt\n"},
				&expect.BExp{R: "hello"},
				&expect.BSnd{S: "echo 'goodbye' > /home/cirros/hello.txt\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "cat /home/cirros/hello.txt\n"},
				&expect.BExp{R: "goodbye"},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: "\\$ "},
			}, 20*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)
			expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			By("Stopping VM")
			vm = tests.StopVirtualMachine(vm)

			By("Restoring VM")
			restore = createRestoreDef(vm, snapshot.Name)

			restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(restore)
			Expect(err).ToNot(HaveOccurred())

			restore = waitRestoreComplete(restore)
			Expect(restore.Status.Restores).To(HaveLen(1))
			Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))

			vm = tests.StartVirtualMachine(vm)
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying original file contents")
			expecter, err = tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())

			res, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "cat /home/cirros/hello.txt\n"},
				&expect.BExp{R: "hello"},
			}, 20*time.Second)
			log.DefaultLogger().Object(vmi).Infof("%v", res)
			expecter.Close()
			Expect(err).ToNot(HaveOccurred())
		}

		It("should restore a vm that boots from a datavolume", func() {
			vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
				tests.GetUrl(tests.CirrosHttpUrl),
				tests.NamespaceTestDefault,
				"#!/bin/bash\necho 'hello'\n",
				snapshotStorageClass,
			))

			originalDVName := vm.Spec.DataVolumeTemplates[0].Name

			doRestore()

			Expect(restore.Status.DeletedDataVolumes).To(ContainElement(originalDVName))
			dvs, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).List(metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(dvs.Items).To(HaveLen(1))
			Expect(dvs.Items[0].Name).To(Equal(vm.Spec.DataVolumeTemplates[0].Name))
		})
	})
})
