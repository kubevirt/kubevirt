package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinstancetype"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	typesStorage "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	makeTestDirectoryCmd      = "sudo mkdir -p /test\n"
	mountTestDirectoryCmd     = "sudo mount %s /test \n"
	makeTestDataDirectoryCmd  = "sudo mkdir -p /test/data\n"
	chmodTestDataDirectoryCmd = "sudo chmod a+w /test/data\n"
	catTestDataMessageCmd     = "cat /test/data/message\n"
	stoppingVM                = "Stopping VM"
	creatingSnapshot          = "creating snapshot"

	macAddressCloningPatchPattern   = `{"op": "replace", "path": "/spec/template/spec/domain/devices/interfaces/0/macAddress", "value": "%s"}`
	firmwareUUIDCloningPatchPattern = `{"op": "replace", "path": "/spec/template/spec/domain/firmware/uuid", "value": "%s"}`

	bashHelloScript = "#!/bin/bash\necho 'hello'\n"

	onlineSnapshot = true
	offlineSnaphot = false
)

var _ = SIGDescribe("VirtualMachineRestore Tests", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	groupName := "kubevirt.io"

	BeforeEach(func() {
		virtClient = kubevirt.Client()
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
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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

		// sometimes it takes a bit for permission to actually be applied so eventually
		Eventually(func() bool {
			_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			if err != nil {
				fmt.Printf("command should have succeeded maybe new permissions not applied yet\nerror\n%s\n", err)
				return false
			}
			return true
		}, 90*time.Second, time.Second).Should(BeTrue())

		vm, err := ThisVM(vm)()
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false
			}
			Expect(err).ToNot(HaveOccurred())
			return vmi.Status.Phase == v1.Running
		}, 360*time.Second, time.Second).Should(BeTrue())

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
		Expect(r).To(matcher.HaveConditionMissingOrFalse(snapshotv1.ConditionProgressing))
		Expect(r).To(matcher.HaveConditionTrue(snapshotv1.ConditionReady))
		return r
	}

	waitDVReady := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
		libstorage.EventuallyDV(dv, 180, matcher.HaveSucceeded())
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
		err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
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

	deletePVC := func(pvc *corev1.PersistentVolumeClaim) {
		err := virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.Background(), pvc.Name, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			err = nil
		}
		Expect(err).ToNot(HaveOccurred())
		pvc = nil
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

	createRestoreDefWithMacAddressPatch := func(sourceVM *v1.VirtualMachine, vmName string, snapshotName string) *snapshotv1.VirtualMachineRestore {
		r := createRestoreDef(vmName, snapshotName)
		if vmName != sourceVM.Name {
			r.Spec.Patches = []string{
				getMacAddressCloningPatch(sourceVM),
			}
		}

		return r
	}

	Context("With simple VM", func() {
		var vm *v1.VirtualMachine

		expectNewVMCreation := func(vmName string) (createdVM *v1.VirtualMachine) {
			Eventually(func() error {
				createdVM, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vmName, &metav1.GetOptions{})
				return err
			}, 90*time.Second, 5*time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("new VM (%s) is not being created", vmName))
			return createdVM
		}

		BeforeEach(func() {
			vm = tests.NewRandomVirtualMachine(
				libvmi.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				), false)
			vm.Labels = map[string]string{
				"kubevirt.io/dummy-webhook-identifier": vm.Name,
			}
		})

		AfterEach(func() {
			deleteVM(vm)
		})

		Context("and no snapshot", func() {
			It("[test_id:5255]should reject restore", func() {
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
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
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
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
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
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

				By("Wait for snapshot to be finished")
				Eventually(func() *string {
					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return vm.Status.SnapshotInProgress
				}, 180*time.Second, time.Second).Should(BeNil())

				origSpec = vm.Spec.DeepCopy()

				initialRequestedMemory := resource.MustParse("128Mi")
				increasedRequestedMemory := resource.MustParse("256Mi")
				patchData, err := patch.GenerateTestReplacePatch(
					"/spec/template/spec/domain/resources/requests/"+string(corev1.ResourceMemory),
					initialRequestedMemory,
					increasedRequestedMemory,
				)
				Expect(err).ToNot(HaveOccurred())

				vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore := createRestoreDef(vm.Name, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(BeEmpty())
				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())

				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vm.Spec).To(Equal(*origSpec))

				deleteRestore(restore)
			})

			It("[test_id:5257]should reject restore if VM running", func() {
				patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
				vm, err := virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{})
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
									Namespace: testsuite.GetTestNamespace(nil),
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
				newVM := tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
				newVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), newVM)
				Expect(err).ToNot(HaveOccurred())
				defer deleteVM(newVM)

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
					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(context.Background(), newVM.Name, &metav1.GetOptions{})
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
		Context("with instancetype and preferences", func() {
			var (
				instancetype *instancetypev1beta1.VirtualMachineInstancetype
				preference   *instancetypev1beta1.VirtualMachinePreference
				snapshot     *snapshotv1.VirtualMachineSnapshot
				restore      *snapshotv1.VirtualMachineRestore
			)

			BeforeEach(func() {
				snapshotStorageClass, err := libstorage.GetSnapshotStorageClass(virtClient)
				Expect(err).ToNot(HaveOccurred())

				if snapshotStorageClass == "" {
					Skip("Skiping test, no VolumeSnapshot support")
				}

				instancetype = &instancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-instancetype-",
						Namespace:    testsuite.GetTestNamespace(nil),
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: 1,
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}
				instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(nil)).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				preferredCPUTopology := instancetypev1beta1.PreferSockets
				preference = &instancetypev1beta1.VirtualMachinePreference{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-preference-",
						Namespace:    testsuite.GetTestNamespace(nil),
					},
					Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1beta1.CPUPreferences{
							PreferredCPUTopology: &preferredCPUTopology,
						},
					},
				}
				preference, err := virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(nil)).Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Template.Spec.Domain.Resources = virtv1.ResourceRequirements{}
				vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
					Name: instancetype.Name,
					Kind: "VirtualMachineInstanceType",
				}
				vm.Spec.Preference = &virtv1.PreferenceMatcher{
					Name: preference.Name,
					Kind: "VirtualMachinePreference",
				}

				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the VM has instancetype and preference RevisionNames")
				libinstancetype.WaitForVMInstanceTypeRevisionNames(vm.Name, virtClient)

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					libstorage.EventuallyDVWith(vm.Namespace, dvt.Name, 180, HaveSucceeded())
				}
			})

			DescribeTable("should use existing ControllerRevisions for an existing VM restore", Label("instancetype", "preference", "restore"), func(toRunSourceVM bool) {
				originalVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if toRunSourceVM {
					By("Starting the VM and expecting it to run")
					vm = tests.StartVMAndExpectRunning(virtClient, vm)
				}

				By("Creating a VirtualMachineSnapshot")
				snapshot = createSnapshot(vm)

				if toRunSourceVM {
					By("Stopping the VM")
					vm = tests.StopVirtualMachine(vm)
				}

				By("Creating a VirtualMachineRestore")
				restore = createRestoreDef(vm.Name, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the restore completes")
				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)

				By("Asserting that the restored VM has the same instancetype and preference controllerRevisions")
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(originalVM.Spec.Instancetype.RevisionName))
				Expect(vm.Spec.Preference.RevisionName).To(Equal(originalVM.Spec.Preference.RevisionName))
			},
				Entry("with a running VM", true),
				Entry("with a stopped VM", false),
			)

			DescribeTable("should create new ControllerRevisions for newly restored VM", Label("instancetype", "preference", "restore"), func(toRunSourceVM bool) {
				if toRunSourceVM {
					By("Starting the VM and expecting it to run")
					vm = tests.StartVMAndExpectRunning(virtClient, vm)
				}

				By("Creating a VirtualMachineSnapshot")
				snapshot = createSnapshot(vm)

				if toRunSourceVM {
					By("Stopping the VM")
					vm = tests.StopVirtualMachine(vm)
				}

				By("Creating a VirtualMachineRestore")
				restoreVMName := vm.Name + "-new"
				restore = createRestoreDefWithMacAddressPatch(vm, restoreVMName, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(testsuite.GetTestNamespace(nil)).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the targetVM is finally created")
				_ = expectNewVMCreation(restoreVMName)

				By("Waiting until the restoreVM has instancetype and preference RevisionNames")
				libinstancetype.WaitForVMInstanceTypeRevisionNames(restoreVMName, virtClient)

				By("Asserting that the restoreVM has new instancetype and preference controllerRevisions")
				sourceVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				restoreVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), restoreVMName, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(restoreVM.Spec.Instancetype.RevisionName).ToNot(Equal(sourceVM.Spec.Instancetype.RevisionName))
				Expect(restoreVM.Spec.Preference.RevisionName).ToNot(Equal(sourceVM.Spec.Preference.RevisionName))

				By("Asserting that the source and target ControllerRevisions contain the same Object")
				Expect(libinstancetype.EnsureControllerRevisionObjectsEqual(sourceVM.Spec.Instancetype.RevisionName, restoreVM.Spec.Instancetype.RevisionName, virtClient)).To(BeTrue(), "source and target instance type controller revisions are expected to be equal")
				Expect(libinstancetype.EnsureControllerRevisionObjectsEqual(sourceVM.Spec.Preference.RevisionName, restoreVM.Spec.Preference.RevisionName, virtClient)).To(BeTrue(), "source and target preference controller revisions are expected to be equal")
			},
				Entry("with a running VM", true),
				Entry("with a stopped VM", false),
			)
		})
	})

	Context("[storage-req]", decorators.StorageReq, func() {
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
				sc, err := libstorage.GetSnapshotStorageClass(virtClient)
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
							&expect.BSnd{S: fmt.Sprintf("sudo mkfs.ext4 -F %s\n", device)},
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

			getFirmwareUUIDCloningPatch := func(sourceVM *v1.VirtualMachine) string {
				return fmt.Sprintf(firmwareUUIDCloningPatchPattern, "")
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

			checkNewVMEquality := func() {
				Expect(newVM.Spec.Template.Spec.Volumes).To(HaveLen(len(vm.Spec.Template.Spec.Volumes)))
				Expect(newVM.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(len(vm.Spec.Template.Spec.Domain.Devices.Disks)))
				Expect(newVM.Spec.Template.Spec.Domain.Devices.Interfaces).To(HaveLen(len(vm.Spec.Template.Spec.Domain.Devices.Interfaces)))
				Expect(newVM.Spec.DataVolumeTemplates).To(HaveLen(len(vm.Spec.DataVolumeTemplates)))
			}

			doRestoreNoVMStart := func(device string, login console.LoginToFunction, onlineSnapshot bool, targetVMName string) {
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
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					Expect(login(vmi)).To(Succeed())
				}

				if !isRestoreToDifferentVM {
					updateMessage(device, onlineSnapshot, vmi)
				}

				By(stoppingVM)
				vm = tests.StopVirtualMachine(vm)

				By("Restoring VM")
				restore = createRestoreDefWithMacAddressPatch(vm, targetVMName, snapshot.Name)
				if vm.Spec.Template.Spec.Domain.Firmware != nil {
					restore.Spec.Patches = append(restore.Spec.Patches, getFirmwareUUIDCloningPatch(vm))
				}

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, targetVMName, targetUID)
			}

			startVMAfterRestore := func(targetVMName, device string, login console.LoginToFunction) {
				isRestoreToDifferentVM := targetVMName != vm.Name
				targetVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), targetVMName, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				targetVM = tests.StartVirtualMachine(targetVM)
				targetVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(context.Background(), targetVM.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying original file contents")
				Expect(login(targetVMI)).To(Succeed())
				verifyOriginalContent(device, targetVMI)

				if isRestoreToDifferentVM {
					newVM = targetVM
				} else {
					vm = targetVM
				}
			}

			doRestore := func(device string, login console.LoginToFunction, onlineSnapshot bool, targetVMName string) {
				doRestoreNoVMStart(device, login, onlineSnapshot, targetVMName)
				startVMAfterRestore(targetVMName, device, login)
			}

			orphanDataVolumeTemplate := func(vm *v1.VirtualMachine, index int) *cdiv1.DataVolume {
				dvt := &vm.Spec.DataVolumeTemplates[index]
				dv := &cdiv1.DataVolume{}
				dv.ObjectMeta = *dvt.ObjectMeta.DeepCopy()
				dv.Spec = *dvt.Spec.DeepCopy()
				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates[:index], vm.Spec.DataVolumeTemplates[index+1:]...)
				return dv
			}

			verifyOwnerRef := func(obj metav1.Object, apiVersion, kind, name string, uid types.UID) {
				ownerRefs := obj.GetOwnerReferences()
				Expect(ownerRefs).To(HaveLen(1))
				ownerRef := ownerRefs[0]
				Expect(ownerRef.APIVersion).To(Equal(apiVersion))
				Expect(ownerRef.Kind).To(Equal(kind))
				Expect(ownerRef.Name).To(Equal(name))
				Expect(ownerRef.UID).To(Equal(uid))
			}

			verifyRestore := func(restoreToNewVM bool, originalDVName string) {
				if restoreToNewVM {
					checkNewVMEquality()
					Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
				} else {
					Expect(restore.Status.DeletedDataVolumes).To(HaveLen(1))
					Expect(restore.Status.DeletedDataVolumes).To(ContainElement(originalDVName))
					_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), originalDVName, metav1.GetOptions{})
					Expect(errors.IsNotFound(err)).To(BeTrue())
				}

				restores := restore.Status.Restores
				Expect(restores).To(HaveLen(1))

				pvcName := restores[0].PersistentVolumeClaimName
				Expect(pvcName).ToNot(BeEmpty())
				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), pvcName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				dvName := restores[0].DataVolumeName
				Expect(dvName).ToNot(BeNil())
				Expect(*dvName).ToNot(BeEmpty())

				targetVM := getTargetVM(restoreToNewVM)

				if libstorage.IsDataVolumeGC(virtClient) {
					Eventually(func() bool {
						_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), *dvName, metav1.GetOptions{})
						return errors.IsNotFound(err)
					}, 30*time.Second, time.Second).Should(BeTrue())
					verifyOwnerRef(pvc, targetVM.APIVersion, targetVM.Kind, targetVM.Name, targetVM.UID)
					return
				}

				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), *dvName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyOwnerRef(dv, targetVM.APIVersion, targetVM.Kind, targetVM.Name, targetVM.UID)
				verifyOwnerRef(pvc, "cdi.kubevirt.io/v1beta1", "DataVolume", dv.Name, dv.UID)
			}

			cloneVM := func(sourceVMName, targetVMName string) {
				By("Creating VM clone")
				vmClone := kubecli.NewMinimalCloneWithNS("testclone", testsuite.GetTestNamespace(nil))
				cloneSourceRef := &corev1.TypedLocalObjectReference{
					APIGroup: pointer.String(groupName),
					Kind:     "VirtualMachine",
					Name:     sourceVMName,
				}
				cloneTargetRef := cloneSourceRef.DeepCopy()
				cloneTargetRef.Name = targetVMName
				vmClone.Spec.Source = cloneSourceRef
				vmClone.Spec.Target = cloneTargetRef

				By(fmt.Sprintf("Creating clone object %s", vmClone.Name))
				vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Create(context.Background(), vmClone, metav1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By(fmt.Sprintf("Waiting for the clone %s to finish", vmClone.Name))
				Eventually(func() clonev1alpha1.VirtualMachineClonePhase {
					vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Get(context.Background(), vmClone.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					return vmClone.Status.Phase
				}, 3*time.Minute, 3*time.Second).Should(Equal(clonev1alpha1.Succeeded), "clone should finish successfully")
			}

			It("[test_id:5259]should restore a vm multiple from the same snapshot", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					testsuite.GetTestNamespace(nil),
					bashHelloScript,
					snapshotStorageClass,
				))

				By(stoppingVM)
				vm = tests.StopVirtualMachine(vm)

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)
				for i := 0; i < 2; i++ {
					By(fmt.Sprintf("Restoring VM iteration %d", i))
					restore = createRestoreDefWithMacAddressPatch(vm, vm.Name, snapshot.Name)
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
					testsuite.GetTestNamespace(nil),
					bashHelloScript,
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

				doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(1))

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
					testsuite.GetTestNamespace(nil),
					bashHelloScript,
					snapshotStorageClass,
				))

				originalDVName := vm.Spec.DataVolumeTemplates[0].Name
				doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
				verifyRestore(restoreToNewVM, originalDVName)
			},
				Entry("[test_id:5260] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm that boots from a datavolume (not template)", func(restoreToNewVM bool) {
				vm = tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					testsuite.GetTestNamespace(nil),
					bashHelloScript,
					snapshotStorageClass,
				)
				dv := orphanDataVolumeTemplate(vm, 0)
				originalPVCName := dv.Name

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				dv = waitDVReady(dv)

				vm, vmi = createAndStartVM(vm)
				doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(1))
				if restoreToNewVM {
					checkNewVMEquality()
				}
				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())

				if !libstorage.IsDataVolumeGC(virtClient) {
					_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
				_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), originalPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, v := range vm.Spec.Template.Spec.Volumes {
					if v.PersistentVolumeClaim != nil {
						Expect(v.PersistentVolumeClaim.ClaimName).ToNot(Equal(originalPVCName))
						pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), v.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						verifyOwnerRef(pvc, v1.GroupVersion.String(), "VirtualMachine", vm.Name, vm.UID)
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
						Namespace: testsuite.GetTestNamespace(nil),
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

				vmi = tests.NewRandomVMIWithPVCAndUserData(pvc.Name, bashHelloScript)
				vm = tests.NewRandomVirtualMachine(vmi, false)

				vm, vmi = createAndStartVM(vm)

				doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(1))
				if restoreToNewVM {
					checkNewVMEquality()
				}

				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
				_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), originalPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				targetVM := getTargetVM(restoreToNewVM)
				targetVM, err = virtClient.VirtualMachine(targetVM.Namespace).Get(context.Background(), targetVM.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if !restoreToNewVM {
					for _, v := range targetVM.Spec.Template.Spec.Volumes {
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
				vmi = libvmi.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
				vm = tests.NewRandomVirtualMachine(vmi, false)
				vm.Namespace = testsuite.GetTestNamespace(nil)

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

				doRestore("/dev/vdc", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(1))

				if restoreToNewVM {
					checkNewVMEquality()
					Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
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
					testsuite.GetTestNamespace(nil),
					bashHelloScript,
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
									Namespace: testsuite.GetTestNamespace(nil),
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

				restore := createRestoreDefWithMacAddressPatch(vm, vm.Name, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					restore, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
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

				patchData, err := patch.GeneratePatchPayload(
					patch.PatchOperation{
						Op:    patch.PatchAddOp,
						Path:  "/spec/running",
						Value: true,
					},
				)
				Expect(err).NotTo(HaveOccurred())
				_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &metav1.PatchOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Cannot start VM until restore %q completes", restore.Name)))

				deleteWebhook(webhook)
				webhook = nil

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)

				Eventually(func() bool {
					updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVM.Status.RestoreInProgress == nil
				}, 30*time.Second, 3*time.Second).Should(BeTrue())

				vm = tests.StartVirtualMachine(vm)
				deleteRestore(restore)
			})

			DescribeTable("should restore a vm from an online snapshot", func(restoreToNewVM bool) {
				vm = tests.NewRandomVMWithDataVolumeAndUserDataInStorageClass(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
					testsuite.GetTestNamespace(nil),
					bashHelloScript,
					snapshotStorageClass,
				)
				vm.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{}
				vm, vmi = createAndStartVM(vm)

				doRestore("", console.LoginToCirros, onlineSnapshot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(1))
				if restoreToNewVM {
					checkNewVMEquality()
				}

			},
				Entry("[test_id:6053] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm from an online snapshot with guest agent", func(restoreToNewVM bool) {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi = tests.NewRandomFedoraVMI()
				vmi.Namespace = testsuite.GetTestNamespace(nil)
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
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(300),
				)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				doRestore("/dev/vdc", console.LoginToFedora, onlineSnapshot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(1))
				if restoreToNewVM {
					checkNewVMEquality()
				}

			},
				Entry("[test_id:6766] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore an online vm snapshot that boots from a datavolumetemplate with guest agent", func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				originalDVName := vm.Spec.DataVolumeTemplates[0].Name
				doRestore("", console.LoginToFedora, onlineSnapshot, getTargetVMName(restoreToNewVM, newVmName))
				verifyRestore(restoreToNewVM, originalDVName)
			},
				Entry("[test_id:6836] to the same VM", false),
				Entry("to a new VM", true),
			)

			It("should restore vm spec at startup without new changes", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Updating the VM template spec")
				initialMemory := vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]
				newMemory := resource.MustParse("2Gi")
				Expect(newMemory).ToNot(Equal(initialMemory))

				patchData, err := patch.GeneratePatchPayload(
					patch.PatchOperation{
						Op:    patch.PatchReplaceOp,
						Path:  "/spec/template/spec/domain/resources/requests/" + string(corev1.ResourceMemory),
						Value: newMemory,
					},
				)
				Expect(err).NotTo(HaveOccurred())
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &metav1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				newVM = tests.StopVirtualMachine(updatedVM)
				newVM = tests.StartVirtualMachine(newVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(newMemory))

				newVM = tests.StopVirtualMachine(newVM)

				By("Restoring VM")
				restore = createRestoreDefWithMacAddressPatch(vm, newVM.Name, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				restore = waitRestoreComplete(restore, newVM.Name, &newVM.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))

				tests.StartVirtualMachine(newVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(initialMemory))
			})

			It("should restore an already cloned virtual machine", func() {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce))

				targetVMName := vm.Name + "-clone"
				cloneVM(vm.Name, targetVMName)

				By(fmt.Sprintf("Getting the cloned VM %s", targetVMName))
				targetVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), targetVMName, &metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By(creatingSnapshot)
				snapshot = createSnapshot(targetVM)
				newVM = tests.StopVirtualMachine(targetVM)

				By("Restoring cloned VM")
				restore = createRestoreDefWithMacAddressPatch(vm, newVM.Name, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(targetVM.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				restore = waitRestoreComplete(restore, newVM.Name, &newVM.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))
			})

			DescribeTable("should restore vm with hot plug disks", func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Add persistent hotplug disk")
				persistVolName := AddVolumeAndVerify(virtClient, snapshotStorageClass, vm, false)
				By("Add temporary hotplug disk")
				tempVolName := AddVolumeAndVerify(virtClient, snapshotStorageClass, vm, true)

				doRestore("", console.LoginToFedora, onlineSnapshot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(2))
				if restoreToNewVM {
					checkNewVMEquality()
				}

				targetVM := getTargetVM(restoreToNewVM)
				targetVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(context.Background(), targetVM.Name, &metav1.GetOptions{})
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

			Context("with memory dump", func() {
				var memoryDumpPVC *corev1.PersistentVolumeClaim
				var memoryDumpPVCName string

				BeforeEach(func() {
					memoryDumpPVCName = "fs-pvc" + rand.String(5)
					memoryDumpPVC = libstorage.NewPVC(memoryDumpPVCName, "1.5Gi", snapshotStorageClass)
					volumeMode := corev1.PersistentVolumeFilesystem
					memoryDumpPVC.Spec.VolumeMode = &volumeMode
					var err error
					memoryDumpPVC, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Create(context.Background(), memoryDumpPVC, metav1.CreateOptions{})
					if err != nil {
						Skip(fmt.Sprintf("Skiping test, no filesystem pvc available, err: %s", err))
					}
				})

				AfterEach(func() {
					if memoryDumpPVC != nil {
						deletePVC(memoryDumpPVC)
					}
				})

				getMemoryDump := func(vmName, namespace, claimName string) {
					Eventually(func() error {
						memoryDumpRequest := &v1.VirtualMachineMemoryDumpRequest{
							ClaimName: claimName,
						}

						return virtClient.VirtualMachine(namespace).MemoryDump(context.Background(), vmName, memoryDumpRequest)
					}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				}

				waitMemoryDumpCompletion := func(vm *v1.VirtualMachine) {
					Eventually(func() bool {
						updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						if updatedVM.Status.MemoryDumpRequest == nil ||
							updatedVM.Status.MemoryDumpRequest.Phase != v1.MemoryDumpCompleted {
							return false
						}

						return true
					}, 60*time.Second, time.Second).Should(BeTrue())
				}

				DescribeTable("should not restore memory dump volume", func(restoreToNewVM bool) {
					vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
						cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
						testsuite.GetTestNamespace(nil),
						snapshotStorageClass,
						corev1.ReadWriteOnce))
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					By("Get VM memory dump")
					getMemoryDump(vm.Name, vm.Namespace, memoryDumpPVCName)
					waitMemoryDumpCompletion(vm)

					doRestoreNoVMStart("", console.LoginToFedora, onlineSnapshot, getTargetVMName(restoreToNewVM, newVmName))
					Expect(restore.Status.Restores).To(HaveLen(1))
					Expect(restore.Status.Restores[0].VolumeName).ToNot(Equal(memoryDumpPVCName))

					restorePVC, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Get(context.Background(), restore.Status.Restores[0].PersistentVolumeClaimName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *snapshot.Status.VirtualMachineSnapshotContentName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					expectedSource := ""
					for _, vb := range content.Spec.VolumeBackups {
						if vb.VolumeName == restore.Status.Restores[0].VolumeName {
							expectedSource = *vb.VolumeSnapshotName
						}
					}
					Expect(restorePVC.Spec.DataSource.Name).To(Equal(expectedSource))

					startVMAfterRestore(getTargetVMName(restoreToNewVM, newVmName), "", console.LoginToFedora)

					targetVM := getTargetVM(restoreToNewVM)
					targetVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(context.Background(), targetVM.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(targetVMI.Spec.Volumes).To(HaveLen(1))
					foundMemoryDump := false
					for _, volume := range targetVMI.Spec.Volumes {
						if volume.Name == memoryDumpPVCName {
							foundMemoryDump = true
							break
						}
					}
					Expect(foundMemoryDump).To(BeFalse())
				},
					Entry("[test_id:8923]to the same VM", false),
					Entry("[test_id:8924]to a new VM", true),
				)
			})

			Context("with cross namespace clone ability", func() {
				var sourceDV *cdiv1.DataVolume
				var cloneRole *rbacv1.Role
				var cloneRoleBinding *rbacv1.RoleBinding

				BeforeEach(func() {
					sourceSC, exists := libstorage.GetRWOFileSystemStorageClass()
					if !exists || sourceSC == snapshotStorageClass {
						Skip("Two storageclasses required for this test")
					}

					source := libdv.NewDataVolume(
						libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), cdiv1.RegistryPullNode),
						libdv.WithPVC(libdv.PVCWithStorageClass(sourceSC)),
						libdv.WithForceBindAnnotation(),
					)

					source, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespaceTestAlternative).Create(context.Background(), source, metav1.CreateOptions{})
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
						libstorage.DeleteDataVolume(&sourceDV)
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
						n := typesStorage.PVCNameFromVirtVolume(&v)
						if n != "" {
							Expect(pvcName).Should(Equal(""))
							pvcName = n
						}
					}
					pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if pvc.Spec.DataSourceRef != nil {
						// These annotations only exist pre-k8s-populators flows
						return
					}
					for _, a := range []string{"k8s.io/CloneRequest", "k8s.io/CloneOf"} {
						_, ok := pvc.Annotations[a]
						Expect(ok).Should(Equal(shouldExist))
					}
				}

				createNetworkCloneVMFromSource := func() *v1.VirtualMachine {
					// TODO: consider ensuring network clone gets done here using StorageProfile CloneStrategy
					dataVolume := libdv.NewDataVolume(
						libdv.WithPVCSource(sourceDV.Namespace, sourceDV.Name),
						libdv.WithPVC(libdv.PVCWithStorageClass(snapshotStorageClass), libdv.PVCWithVolumeSize("1Gi")),
					)

					vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
					tests.AddUserData(vmi, "cloud-init", bashHelloScript)
					vm := tests.NewRandomVirtualMachine(vmi, false)
					libstorage.AddDataVolumeTemplate(vm, dataVolume)
					return vm
				}

				DescribeTable("should restore a vm that boots from a network cloned datavolumetemplate", func(restoreToNewVM, deleteSourcePVC bool) {
					vm, vmi = createAndStartVM(createNetworkCloneVMFromSource())

					checkCloneAnnotations(vm, true)
					if deleteSourcePVC {
						libstorage.DeleteDataVolume(&sourceDV)
					}

					doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
					checkCloneAnnotations(getTargetVM(restoreToNewVM), false)
				},
					Entry("to the same VM", false, false),
					Entry("to a new VM", true, false),
					Entry("to the same VM, no source pvc", false, true),
					Entry("to a new VM, no source pvc", true, true),
				)

				DescribeTable("should restore a vm that boots from a network cloned datavolume (not template)", func(restoreToNewVM, deleteSourcePVC bool) {
					vm = createNetworkCloneVMFromSource()
					dv := orphanDataVolumeTemplate(vm, 0)

					dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					defer libstorage.DeleteDataVolume(&dv)
					waitDVReady(dv)

					vm, vmi = createAndStartVM(vm)

					checkCloneAnnotations(vm, true)
					if deleteSourcePVC {
						libstorage.DeleteDataVolume(&sourceDV)
					}
					doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
					checkCloneAnnotations(getTargetVM(restoreToNewVM), false)
				},
					Entry("to the same VM", false, false),
					Entry("to a new VM", true, false),
					Entry("to the same VM, no source pvc", false, true),
					Entry("to a new VM, no source pvc", true, true),
				)
			})
		})
	})
})
