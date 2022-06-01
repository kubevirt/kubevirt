package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/api/core"

	"kubevirt.io/kubevirt/tests/util"

	expect "github.com/google/goexpect"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	typesutil "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

const (
	makeTestDirectoryCmd      = "sudo mkdir -p /test\n"
	mountTestDirectoryCmd     = "sudo mount %s /test \n"
	makeTestDataDirectoryCmd  = "sudo mkdir -p /test/data\n"
	chmodTestDataDirectoryCmd = "sudo chmod a+w /test/data\n"
	catTestDataMessageCmd     = "cat /test/data/message\n"
	stoppingVM                = "Stopping VM"
	creatingSnapshot          = "creating snapshot"

	macAddressCloningPatchPattern = `{"op": "replace", "path": "/spec/template/spec/domain/devices/interfaces/0/macAddress", "value": "%s"}`

	offlineSnaphot = false
)

var _ = SIGDescribe("VirtualMachineRestore Tests", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	groupName := "kubevirt.io"

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	createRestoreDef := func(vmName, snapshotName string) *snapshotv1.VirtualMachineRestore {
		return &snapshotv1.VirtualMachineRestore{
			ObjectMeta: metav1.ObjectMeta{
				Name: "restore-" + vmName,
			},
			Spec: snapshotv1.VirtualMachineRestoreSpec{
				Target: corev1.TypedLocalObjectReference{
					APIGroup: &groupName,
					Kind:     "VirtualMachine",
					Name:     vmName,
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
		var gracePeriod int64 = 10
		vm.Spec.Template.Spec.TerminationGracePeriodSeconds = &gracePeriod
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

	waitRestoreComplete := func(r *snapshotv1.VirtualMachineRestore, vmName string, vmUID *types.UID) *snapshotv1.VirtualMachineRestore {
		var err error
		Eventually(func() bool {
			r, err = virtClient.VirtualMachineRestore(r.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return r.Status != nil && r.Status.Complete != nil && *r.Status.Complete
		}, 180*time.Second, time.Second).Should(BeTrue())
		Expect(r.OwnerReferences).To(HaveLen(1))
		Expect(r.OwnerReferences[0].APIVersion).To(Equal(v1.GroupVersion.String()))
		Expect(r.OwnerReferences[0].Kind).To(Equal("VirtualMachine"))
		Expect(r.OwnerReferences[0].Name).To(Equal(vmName))
		if vmUID != nil {
			Expect(r.OwnerReferences[0].UID).To(Equal(*vmUID))
		}
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

	deleteVM := func(vm *v1.VirtualMachine) {
		err := virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			err = nil
		}
		Expect(err).ToNot(HaveOccurred())
	}

	deleteSnapshot := func(s *snapshotv1.VirtualMachineSnapshot) {
		err := virtClient.VirtualMachineSnapshot(s.Namespace).Delete(context.Background(), s.Name, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			err = nil
		}
		Expect(err).ToNot(HaveOccurred())
	}

	deleteRestore := func(r *snapshotv1.VirtualMachineRestore) {
		err := virtClient.VirtualMachineRestore(r.Namespace).Delete(context.Background(), r.Name, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			err = nil
		}
		Expect(err).ToNot(HaveOccurred())
	}

	deleteWebhook := func(wh *admissionregistrationv1.ValidatingWebhookConfiguration) {
		err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), wh.Name, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			err = nil
		}
		Expect(err).ToNot(HaveOccurred())
	}

	getMacAddressCloningPatch := func(sourceVM *v1.VirtualMachine) string {
		interfaces := sourceVM.Spec.Template.Spec.Domain.Devices.Interfaces
		Expect(interfaces).ToNot(BeEmpty())
		isMacAddressEmpty := interfaces[0].MacAddress == ""

		if isMacAddressEmpty {
			// This means there is no KubeMacPool running. Therefore, we can simply choose a random address
			return fmt.Sprintf(macAddressCloningPatchPattern, "DE-AD-00-00-BE-AF")
		} else {
			// KubeMacPool is active. Therefore, we can return an empty address and KubeMacPool would assign a real address for us
			return fmt.Sprintf(macAddressCloningPatchPattern, "")
		}
	}

	Context("With simple VM", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, tests.BashHelloScript)
			vm = tests.NewRandomVirtualMachine(vmi, false)
			vm.Labels = map[string]string{
				"kubevirt.io/dummy-webhook-identifier": vm.Name,
			}
		})

		Context("and no snapshot", func() {
			It("[test_id:5255]should reject restore", func() {
				vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				restore := createRestoreDef(vm.Name, "foobar")

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

				restore := createRestoreDef(vm.Name, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(BeEmpty())
				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
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

				restore := createRestoreDef(vm.Name, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(BeEmpty())
				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())

				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(vm.Spec).To(Equal(*origSpec))

				deleteRestore(restore)
			})

			It("[test_id:5257]should reject restore if VM running", func() {
				patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
				vm, err := virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore := createRestoreDef(vm.Name, snapshot.Name)

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
						Name: "temp-webhook-deny-vm-update" + rand.String(5),
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
							ObjectSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubevirt.io/dummy-webhook-identifier": vm.Name,
								},
							},
						},
					},
				}
				wh, err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), wh, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				webhook = wh

				restore := createRestoreDef(vm.Name, snapshot.Name)

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

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)

				r2, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), r2, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				r2 = waitRestoreComplete(r2, vm.Name, &vm.UID)

				deleteRestore(r2)
				deleteRestore(restore)
			})

			It("should fail restoring to a different VM that already exists", func() {
				By("Creating a new VM")
				newVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), tests.BashHelloScript)
				newVM := tests.NewRandomVirtualMachine(newVMI, false)
				newVM, err = virtClient.VirtualMachine(newVM.Namespace).Create(newVM)
				Expect(err).ToNot(HaveOccurred())

				By("Creating a VM restore")
				restore := createRestoreDef(newVM.Name, snapshot.Name)
				_, err = virtClient.VirtualMachineRestore(newVM.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred(), "Admission webhooks should reject this restore since target VM is different then source and already exists")
			})

			Context("restore to a new VM that does not exist", func() {

				var (
					newVmName string
					newVM     *v1.VirtualMachine
					restore   *snapshotv1.VirtualMachineRestore
				)

				BeforeEach(func() {
					newVmName = "new-vm-" + rand.String(12)
					restore = createRestoreDef(newVmName, snapshot.Name)
				})

				expectNewVMCreation := func(vmName string) (createdVM *v1.VirtualMachine) {
					Eventually(func() error {
						createdVM, err = virtClient.VirtualMachine(vm.Namespace).Get(vmName, &metav1.GetOptions{})
						return err
					}, 90*time.Second, 5*time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("new VM (%s) is not being created", newVmName))

					return createdVM
				}

				waitRestoreComplete := func(r *snapshotv1.VirtualMachineRestore, vm *v1.VirtualMachine) *snapshotv1.VirtualMachineRestore {
					r = waitRestoreComplete(r, vm.Name, &vm.UID)
					Expect(r.Status.Restores).To(BeEmpty())
					Expect(r.Status.DeletedDataVolumes).To(BeEmpty())
					return r
				}

				It("with changed name and MAC address", func() {
					By("Creating a VM restore with patches to change name and MAC address")
					restore.Spec.Patches = []string{getMacAddressCloningPatch(vm)}
					restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					defer deleteRestore(restore)

					By("Making sure that new VM is finally created")
					newVM = expectNewVMCreation(newVmName)
					defer deleteVM(newVM)

					By("Waiting for VM restore to complete")
					restore = waitRestoreComplete(restore, newVM)

					By("Expecting the restore to be owned by target VM")
					isOwnedByTargetVM := false
					for _, ownReference := range restore.ObjectMeta.OwnerReferences {
						if ownReference.Kind == "VirtualMachine" && ownReference.UID == newVM.UID && ownReference.Name == newVM.Name {
							isOwnedByTargetVM = true
							break
						}
					}
					Expect(isOwnedByTargetVM).To(BeTrue(), "restore is expected to be owned by target VM")

					By("Verifying both VMs exist")
					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					By("Verifying newly created VM is set properly")
					Expect(newVM.Name).To(Equal(newVmName), "newly created VM should have correct name")
					newVMInterfaces := newVM.Spec.Template.Spec.Domain.Devices.Interfaces
					Expect(newVMInterfaces).ToNot(BeEmpty())

					By("Verifying both VMs have different spec")
					oldVMInterfaces := vm.Spec.Template.Spec.Domain.Devices.Interfaces
					Expect(oldVMInterfaces).ToNot(BeEmpty())
					Expect(newVMInterfaces[0].MacAddress).ToNot(Equal(oldVMInterfaces[0].MacAddress))

					By("Making sure new VM is runnable")
					tests.StartVMAndExpectRunning(virtClient, newVM)
				})

			})
		})
	})

	Context("[storage-req]", func() {
		Context("With a more complicated VM", func() {
			var (
				newVmName            string
				vm                   *v1.VirtualMachine
				vmi                  *v1.VirtualMachineInstance
				newVM                *v1.VirtualMachine
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
				newVmName = "new-vm-" + rand.String(12)
			})

			AfterEach(func() {
				if vm != nil {
					deleteVM(vm)
				}
				if newVM != nil {
					deleteVM(newVM)
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

				vm = nil
				vmi = nil
				newVM = nil
				snapshot = nil
				restore = nil
				webhook = nil
			})

			createMessageWithInitialValue := func(login console.LoginToFunction, device string, vmis ...*v1.VirtualMachineInstance) {
				for _, vmi := range vmis {
					if vmi == nil {
						continue
					}

					By(fmt.Sprintf("creating 'message with initial value for VMI %s", vmi.Name))
					Expect(login(vmi)).To(Succeed())

					var batch []expect.Batcher
					if device != "" {
						batch = append(batch, []expect.Batcher{
							&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 %s\n", device)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: tests.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: makeTestDirectoryCmd},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: fmt.Sprintf(mountTestDirectoryCmd, device)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: tests.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
						}...)
					}

					batch = append(batch, []expect.Batcher{
						&expect.BSnd{S: makeTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: chmodTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: fmt.Sprintf("echo '%s' > /test/data/message\n", vm.UID)},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: catTestDataMessageCmd},
						&expect.BExp{R: string(vm.UID)},
						&expect.BSnd{S: syncName},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: syncName},
						&expect.BExp{R: console.PromptExpression},
					}...)

					Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
				}
			}

			updateMessage := func(device string, onlineSnapshot bool, vmis ...*v1.VirtualMachineInstance) {
				for _, vmi := range vmis {
					if vmi == nil {
						continue
					}
					By(fmt.Sprintf("updating message for vmi %s", vmi.Name))

					var batch []expect.Batcher
					if !onlineSnapshot && device != "" {
						batch = append(batch, []expect.Batcher{
							&expect.BSnd{S: makeTestDirectoryCmd},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: fmt.Sprintf(mountTestDirectoryCmd, device)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: tests.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
						}...)
					}

					batch = append(batch, []expect.Batcher{
						&expect.BSnd{S: makeTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: chmodTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: catTestDataMessageCmd},
						&expect.BExp{R: string(vm.UID)},
						&expect.BSnd{S: fmt.Sprintf("echo '%s' > /test/data/message\n", snapshot.UID)},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: catTestDataMessageCmd},
						&expect.BExp{R: string(snapshot.UID)},
						&expect.BSnd{S: syncName},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: syncName},
						&expect.BExp{R: console.PromptExpression},
					}...)

					Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
				}
			}

			verifyOriginalContent := func(device string, vmis ...*v1.VirtualMachineInstance) {
				for _, vmi := range vmis {
					if vmi == nil {
						continue
					}

					var batch []expect.Batcher
					batch = nil

					if device != "" {
						batch = append(batch, []expect.Batcher{
							&expect.BSnd{S: makeTestDirectoryCmd},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: fmt.Sprintf(mountTestDirectoryCmd, device)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: tests.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
						}...)
					}

					batch = append(batch, []expect.Batcher{
						&expect.BSnd{S: makeTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: chmodTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: tests.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: catTestDataMessageCmd},
						&expect.BExp{R: string(vm.UID)},
					}...)

					Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
				}
			}

			createRestoreDef := func(vmName string, snapshotName string) *snapshotv1.VirtualMachineRestore {
				r := createRestoreDef(vmName, snapshotName)
				if vmName != vm.Name {
					r.Spec.Patches = []string{getMacAddressCloningPatch(vm)}
				}

				return r
			}

			getTargetVMName := func(restoreToNewVM bool, newVmName string) string {
				if restoreToNewVM {
					return newVmName
				}

				return vm.Name
			}

			getTargetVM := func(restoreToNewVM bool) *v1.VirtualMachine {
				if restoreToNewVM {
					return newVM
				}

				return vm
			}

			doRestore := func(device string, login console.LoginToFunction, onlineSnapshot bool, expectedRestores int, targetVMName string) {
				isRestoreToDifferentVM := targetVMName != vm.Name

				var targetUID *types.UID
				if !isRestoreToDifferentVM {
					targetUID = &vm.UID
				}

				createMessageWithInitialValue(login, device, vmi)

				if !onlineSnapshot {
					By(stoppingVM)
					vm = tests.StopVirtualMachine(vm)
				}

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				if !onlineSnapshot {
					By("Starting VM")
					vm = tests.StartVirtualMachine(vm)
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					Expect(login(vmi)).To(Succeed())
				}

				if !isRestoreToDifferentVM {
					updateMessage(device, onlineSnapshot, vmi)
				}

				By(stoppingVM)
				vm = tests.StopVirtualMachine(vm)

				By("Restoring VM")
				restore = createRestoreDef(targetVMName, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, targetVMName, targetUID)
				Expect(restore.Status.Restores).To(HaveLen(expectedRestores))

				targetVM, err := virtClient.VirtualMachine(vm.Namespace).Get(targetVMName, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				targetVM = tests.StartVirtualMachine(targetVM)
				targetVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(targetVM.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying original file contents")
				Expect(login(targetVMI)).To(Succeed())
				verifyOriginalContent(device, targetVMI)

				if isRestoreToDifferentVM {
					Expect(targetVM.Spec.Template.Spec.Volumes).To(HaveLen(len(vm.Spec.Template.Spec.Volumes)))
					Expect(targetVM.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(len(vm.Spec.Template.Spec.Domain.Devices.Disks)))
					Expect(targetVM.Spec.Template.Spec.Domain.Devices.Interfaces).To(HaveLen(len(vm.Spec.Template.Spec.Domain.Devices.Interfaces)))
					Expect(targetVM.Spec.DataVolumeTemplates).To(HaveLen(len(vm.Spec.DataVolumeTemplates)))

					newVM = targetVM
				} else {
					vm = targetVM
				}
			}

			orphanDataVolumeTemplate := func(vm *v1.VirtualMachine, index int) *cdiv1.DataVolume {
				dvt := &vm.Spec.DataVolumeTemplates[index]
				dv := &cdiv1.DataVolume{}
				dv.ObjectMeta = *dvt.ObjectMeta.DeepCopy()
				dv.Spec = *dvt.Spec.DeepCopy()
				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates[:index], vm.Spec.DataVolumeTemplates[index+1:]...)
				return dv
			}

			It("[test_id:5259]should restore a vm multiple from the same snapshot", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					tests.BashHelloScript,
					snapshotStorageClass,
				))

				By(stoppingVM)
				vm = tests.StopVirtualMachine(vm)

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				for i := 0; i < 2; i++ {
					By(fmt.Sprintf("Restoring VM iteration %d", i))
					restore = createRestoreDef(vm.Name, snapshot.Name)
					restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
					Expect(restore.Status.Restores).To(HaveLen(1))

					deleteRestore(restore)
					restore = nil
				}
			})

			// This test is relevant to provisioner which round up the recieved size of
			// the PVC. Currently we only test vmsnapshot tests which ceph which has this
			// behavior. In case of running this test with other provisioner or if ceph
			// will change this behavior it will fail.
			DescribeTable("should restore a vm with restore size bigger then PVC size", func(restoreToNewVM bool) {
				vm = tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					tests.BashHelloScript,
					snapshotStorageClass,
				)
				quantity, err := resource.ParseQuantity("1528Mi")
				Expect(err).ToNot(HaveOccurred())
				vm.Spec.DataVolumeTemplates[0].Spec.PVC.Resources.Requests["storage"] = quantity
				vm, vmi = createAndStartVM(vm)
				expectedCapacity, err := resource.ParseQuantity("2Gi")
				Expect(err).ToNot(HaveOccurred())
				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), vm.Spec.DataVolumeTemplates[0].Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc.Status.Capacity["storage"]).To(Equal(expectedCapacity))

				doRestore("", console.LoginToCirros, false, 1, getTargetVMName(restoreToNewVM))

				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *snapshot.Status.VirtualMachineSnapshotContentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(content.Spec.VolumeBackups[0].PersistentVolumeClaim.Spec.Resources.Requests["storage"]).To(Equal(quantity))
				vs, err := virtClient.KubernetesSnapshotClient().SnapshotV1().VolumeSnapshots(vm.Namespace).Get(context.Background(), *content.Spec.VolumeBackups[0].VolumeSnapshotName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*vs.Status.RestoreSize).To(Equal(expectedCapacity))

				pvc, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), restore.Status.Restores[0].PersistentVolumeClaimName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc.Status.Capacity["storage"]).To(Equal(expectedCapacity))
				Expect(pvc.Spec.Resources.Requests["storage"]).To(Equal(expectedCapacity))

			},
				Entry("to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm that boots from a datavolumetemplate", func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					tests.BashHelloScript,
					snapshotStorageClass,
				))

				originalDVName := vm.Spec.DataVolumeTemplates[0].Name

				doRestore("", console.LoginToCirros, false, 1, getTargetVMName(restoreToNewVM, newVmName))
				if restoreToNewVM {
					Expect(restore.Status.DeletedDataVolumes).To(HaveLen(0))
				} else {
					Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
					Expect(restore.Status.DeletedDataVolumes).To(ContainElement(originalDVName))

					_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(util.NamespaceTestDefault).Get(context.Background(), originalDVName, metav1.GetOptions{})
					Expect(errors.IsNotFound(err)).To(BeTrue())
				}
			},
				Entry("[test_id:5260] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm that boots from a datavolume (not template)", func(restoreToNewVM bool) {
				vm = tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					tests.BashHelloScript,
					snapshotStorageClass,
				)
				dv := orphanDataVolumeTemplate(vm, 0)
				originalPVCName := dv.Name

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				dv = waitDVReady(dv)

				vm, vmi = createAndStartVM(vm)
				doRestore("", console.LoginToCirros, false, 1, getTargetVMName(restoreToNewVM, newVmName))
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
			},
				Entry("[test_id:5261] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm that boots from a PVC", func(restoreToNewVM bool) {
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

				vmi = tests.NewRandomVMIWithPVCAndUserData(pvc.Name, tests.BashHelloScript)
				vm = tests.NewRandomVirtualMachine(vmi, false)

				vm, vmi = createAndStartVM(vm)

				doRestore("", console.LoginToCirros, false, 1, getTargetVMName(restoreToNewVM, newVmName))

				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
				_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), originalPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				targetVM := getTargetVM(restoreToNewVM)
				targetVM, err = virtClient.VirtualMachine(targetVM.Namespace).Get(targetVM.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if !restoreToNewVM {
					for _, v := range vm.Spec.Template.Spec.Volumes {
						if v.PersistentVolumeClaim != nil {
							Expect(v.PersistentVolumeClaim.ClaimName).ToNot(Equal(originalPVCName))
							pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), v.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							Expect(pvc.OwnerReferences[0].APIVersion).To(Equal(v1.GroupVersion.String()))
							Expect(pvc.OwnerReferences[0].Kind).To(Equal("VirtualMachine"))
							Expect(pvc.OwnerReferences[0].Name).To(Equal(vm.Name))
							Expect(pvc.OwnerReferences[0].UID).To(Equal(vm.UID))
							Expect(pvc.Labels["restore.kubevirt.io/source-vm-name"]).To(Equal(vm.Name))
						}
					}
				}
			},
				Entry("[test_id:5262] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm with containerdisk and blank datavolume", func(restoreToNewVM bool) {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(
					cd.ContainerDiskFor(cd.ContainerDiskCirros),
					tests.BashHelloScript,
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
							Bus: v1.DiskBusVirtio,
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

				doRestore("/dev/vdc", console.LoginToCirros, false, 1, getTargetVMName(restoreToNewVM, newVmName))

				if restoreToNewVM {
					Expect(restore.Status.DeletedDataVolumes).To(HaveLen(0))
				} else {
					Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
					Expect(restore.Status.DeletedDataVolumes).To(ContainElement(dvName))
					_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), dvName, metav1.GetOptions{})
					Expect(errors.IsNotFound(err)).To(BeTrue())
				}
			},
				Entry("[test_id:5263] to the same VM", false),
				Entry("to a new VM", true),
			)

			It("should reject vm start if restore in progress", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					tests.BashHelloScript,
					snapshotStorageClass,
				))

				By(stoppingVM)
				vm = tests.StopVirtualMachine(vm)

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				fp := admissionregistrationv1.Fail
				sideEffectNone := admissionregistrationv1.SideEffectClassNone
				whPath := "/foobar"
				whName := "dummy-webhook-deny-pvc-create.kubevirt.io"
				wh := &admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: "temp-webhook-deny-pvc-create" + rand.String(5),
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
							ObjectSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"restore.kubevirt.io/source-vm-name": vm.Name,
								},
							},
						},
					},
				}
				wh, err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), wh, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				webhook = wh

				restore := createRestoreDef(vm.Name, snapshot.Name)

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

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)

				Eventually(func() bool {
					updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVM.Status.RestoreInProgress == nil
				}, 30*time.Second, 3*time.Second).Should(BeTrue())

				vm = tests.StartVirtualMachine(vm)
				deleteRestore(restore)
			})

			DescribeTable("should restore a vm from an online snapshot", func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					tests.BashHelloScript,
					snapshotStorageClass,
				))

				doRestore("", console.LoginToCirros, true, 1, getTargetVMName(restoreToNewVM, newVmName))

			},
				Entry("[test_id:6053] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm from an online snapshot with guest agent", func(restoreToNewVM bool) {
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
							Bus: v1.DiskBusVirtio,
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

				doRestore("/dev/vdc", console.LoginToFedora, true, 1, getTargetVMName(restoreToNewVM, newVmName))

			},
				Entry("[test_id:6766] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore an online vm snapshot that boots from a datavolumetemplate with guest agent", func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				tests.WaitAgentConnected(virtClient, vmi)
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				originalDVName := vm.Spec.DataVolumeTemplates[0].Name

				doRestore("", console.LoginToFedora, true, 1, getTargetVMName(restoreToNewVM, newVmName))
				if restoreToNewVM {
					Expect(restore.Status.DeletedDataVolumes).To(HaveLen(0))
				} else {
					Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
					Expect(restore.Status.DeletedDataVolumes).To(ContainElement(originalDVName))

					_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(util.NamespaceTestDefault).Get(context.Background(), originalDVName, metav1.GetOptions{})
					Expect(errors.IsNotFound(err)).To(BeTrue())
				}
			},
				Entry("[test_id:6836] to the same VM", false),
				Entry("to a new VM", true),
			)

			It("should restore vm spec at startup without new changes", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				tests.WaitAgentConnected(virtClient, vmi)
				Expect(console.LoginToFedora(vmi)).To(Succeed())

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

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				newVM = tests.StopVirtualMachine(updatedVM)
				newVM = tests.StartVirtualMachine(newVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(newMemory))

				newVM = tests.StopVirtualMachine(newVM)

				By("Restoring VM")
				restore = createRestoreDef(newVM.Name, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				restore = waitRestoreComplete(restore, newVM.Name, &newVM.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))

				newVM = tests.StartVirtualMachine(newVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(initialMemory))
			})

			It("should restore a virtual machine using a restore PVC with populated dataSourceRef", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)
				newVM = tests.StopVirtualMachine(vm)

				// We add an invalid dataSourceRef into the virtualMachineSnapshotContent so it's inherited by the restore PVC
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *snapshot.Status.VirtualMachineSnapshotContentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				content.Spec.VolumeBackups[0].PersistentVolumeClaim.Spec.DataSourceRef = &corev1.TypedLocalObjectReference{
					Kind: "test",
					Name: "test",
				}
				_, err = virtClient.VirtualMachineSnapshotContent(vm.Namespace).Update(context.Background(), content, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Restoring VM")
				restore = createRestoreDef(newVM.Name, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				restore = waitRestoreComplete(restore, newVM.Name, &newVM.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))
			})

			DescribeTable("should restore vm with hot plug disks", func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					util.NamespaceTestDefault,
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				tests.WaitAgentConnected(virtClient, vmi)
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Add persistent hotplug disk")
				persistVolName := tests.AddVolumeAndVerify(virtClient, snapshotStorageClass, vm, false)
				By("Add temporary hotplug disk")
				tempVolName := tests.AddVolumeAndVerify(virtClient, snapshotStorageClass, vm, true)

				doRestore("", console.LoginToFedora, true, 2, getTargetVMName(restoreToNewVM, newVmName))

				targetVM := getTargetVM(restoreToNewVM)
				targetVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(targetVM.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(targetVMI.Spec.Volumes).To(HaveLen(2))
				foundHotPlug := false
				foundTempHotPlug := false
				for _, volume := range targetVMI.Spec.Volumes {
					if volume.Name == persistVolName {
						foundHotPlug = true
					} else if volume.Name == tempVolName {
						foundTempHotPlug = true
					}
				}
				Expect(foundHotPlug).To(BeTrue())
				Expect(foundTempHotPlug).To(BeFalse())
			},
				Entry("[test_id:7425] to the same VM", false),
				Entry("to a new VM", true),
			)

			Context("with cross namespace clone ability", func() {
				var sourceDV *cdiv1.DataVolume
				var cloneRole *rbacv1.Role
				var cloneRoleBinding *rbacv1.RoleBinding

				BeforeEach(func() {
					sourceSC, exists := libstorage.GetRWOFileSystemStorageClass()
					if !exists || sourceSC == snapshotStorageClass {
						Skip("Two storageclasses required for this test")
					}

					source := libstorage.NewRandomDataVolumeWithRegistryImportInStorageClass(
						cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
						testsuite.NamespaceTestAlternative,
						sourceSC,
						corev1.ReadWriteOnce,
						corev1.PersistentVolumeFilesystem,
					)
					if source.Annotations == nil {
						source.Annotations = make(map[string]string)
					}
					source.Annotations["cdi.kubevirt.io/storage.bind.immediate.requested"] = "true"

					source, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(source.Namespace).Create(context.Background(), source, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					sourceDV = source
					sourceDV = waitDVReady(sourceDV)

					role, roleBinding := libstorage.GoldenImageRBAC(testsuite.NamespaceTestAlternative)
					role, err = virtClient.RbacV1().Roles(role.Namespace).Create(context.TODO(), role, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					cloneRole = role
					roleBinding, err = virtClient.RbacV1().RoleBindings(roleBinding.Namespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					cloneRoleBinding = roleBinding
				})

				AfterEach(func() {
					if sourceDV != nil {
						err := virtClient.CdiClient().CdiV1beta1().DataVolumes(sourceDV.Namespace).Delete(context.TODO(), sourceDV.Name, metav1.DeleteOptions{})
						Expect(err).ToNot(HaveOccurred())
					}

					if cloneRole != nil {
						err := virtClient.RbacV1().Roles(cloneRole.Namespace).Delete(context.TODO(), cloneRole.Name, metav1.DeleteOptions{})
						Expect(err).ToNot(HaveOccurred())
					}
					if cloneRoleBinding != nil {
						err = virtClient.RbacV1().RoleBindings(cloneRoleBinding.Namespace).Delete(context.TODO(), cloneRoleBinding.Name, metav1.DeleteOptions{})
						Expect(err).ToNot(HaveOccurred())
					}
				})

				checkCloneAnnotations := func(vm *v1.VirtualMachine, shouldExist bool) {
					pvcName := ""
					for _, v := range vm.Spec.Template.Spec.Volumes {
						n := typesutil.PVCNameFromVirtVolume(&v)
						if n != "" {
							Expect(pvcName).Should(Equal(""))
							pvcName = n
						}
					}
					pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, a := range []string{"k8s.io/CloneRequest", "k8s.io/CloneOf"} {
						_, ok := pvc.Annotations[a]
						Expect(ok).Should(Equal(shouldExist))
					}
				}

				createVMFromSource := func() *v1.VirtualMachine {
					dataVolume := libstorage.NewRandomDataVolumeWithPVCSource(
						sourceDV.Namespace,
						sourceDV.Name,
						util.NamespaceTestDefault,
						corev1.ReadWriteOnce,
					)
					libstorage.SetDataVolumePVCStorageClass(dataVolume, snapshotStorageClass)
					libstorage.SetDataVolumePVCSize(dataVolume, "6Gi")
					vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
					tests.AddUserData(vmi, "cloud-init", tests.BashHelloScript)
					vm := tests.NewRandomVirtualMachine(vmi, false)
					libstorage.AddDataVolumeTemplate(vm, dataVolume)
					return vm
				}

				DescribeTable("should restore a vm that boots from a network cloned datavolume (not template)", func(restoreToNewVM bool) {
					vm = createVMFromSource()
					dv := orphanDataVolumeTemplate(vm, 0)

					dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					defer func() {
						err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.TODO(), dv.Name, metav1.DeleteOptions{})
						Expect(err).ToNot(HaveOccurred())
					}()
					dv = waitDVReady(dv)

					vm, vmi = createAndStartVM(vm)

					checkCloneAnnotations(vm, true)
					doRestore("", console.LoginToCirros, offlineSnaphot, 1, getTargetVMName(restoreToNewVM))
					checkCloneAnnotations(getTargetVM(restoreToNewVM), false)
				},
					Entry("to the same VM", false),
					Entry("to a new VM", true),
				)
			})
		})
	})
})
