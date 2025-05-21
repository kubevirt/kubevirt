package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/libvmops"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	expect "github.com/google/goexpect"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	clone "kubevirt.io/api/clone/v1beta1"
	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinstancetype"

	"k8s.io/utils/ptr"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"
	typesStorage "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
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

	onlineSnapshot      = true
	offlineSnaphot      = false
	stopVMBeforeRestore = true
	stopVMAfterRestore  = false
)

var _ = Describe(SIG("VirtualMachineRestore Tests", func() {

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
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return s.Status != nil && s.Status.ReadyToUse != nil && *s.Status.ReadyToUse && vm.Status.SnapshotInProgress == nil
		}, 180*time.Second, time.Second).Should(BeTrue())

		return s
	}

	createAndStartVM := func(vm *v1.VirtualMachine) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
		var vmi *v1.VirtualMachineInstance

		vm.Spec.RunStrategy = virtpointer.P(v1.RunStrategyAlways)
		var gracePeriod int64 = 10
		vm.Spec.Template.Spec.TerminationGracePeriodSeconds = &gracePeriod

		// sometimes it takes a bit for permission to actually be applied so eventually
		Eventually(func() bool {
			_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			if err != nil {
				fmt.Printf("command should have succeeded maybe new permissions not applied yet\nerror\n%s\n", err)
				return false
			}
			return true
		}, 90*time.Second, time.Second).Should(BeTrue())

		vm, err := ThisVM(vm)()
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() bool {
			vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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

	deleteVM := func(vm *v1.VirtualMachine) {
		err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
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
		Eventually(func() error {
			_, err = virtClient.VirtualMachineRestore(r.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			return err
		}, 30*time.Second, 2*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
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

	createVMWithCloudInit := func(containerDisk cd.ContainerDisk, storageClass string, opts ...libvmi.Option) *v1.VirtualMachine {
		defaultOpts := []libvmi.Option{
			libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
		}
		opts = append(defaultOpts, opts...)
		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(containerDisk)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithStorageClass(storageClass),
				libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(cd.DataVolumeImportUrlForContainerDisk(containerDisk))),
			),
		)
		return libvmi.NewVirtualMachine(
			libstorage.RenderVMIWithDataVolume(dv.Name, dv.Namespace, opts...),
			libvmi.WithDataVolumeTemplate(dv),
		)
	}

	Context("With simple VM", func() {
		var vm *v1.VirtualMachine

		expectNewVMCreation := func(vmName string) (createdVM *v1.VirtualMachine) {
			Eventually(func() error {
				createdVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vmName, metav1.GetOptions{})
				return err
			}, 90*time.Second, 5*time.Second).ShouldNot(HaveOccurred(), fmt.Sprintf("new VM (%s) is not being created", vmName))
			return createdVM
		}

		BeforeEach(func() {
			vm = libvmi.NewVirtualMachine(
				libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				))
			vm.Labels = map[string]string{
				"kubevirt.io/dummy-webhook-identifier": vm.Name,
			}
		})

		AfterEach(func() {
			deleteVM(vm)
		})

		Context("and no snapshot", func() {
			It("should wait for snapshot to exist and be ready", func() {
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				restore := createRestoreDef(vm.Name, fmt.Sprintf("snapshot-%s", vm.Name))

				restore, err := virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				events.ExpectEvent(restore, corev1.EventTypeWarning, "VirtualMachineRestoreError")
				createSnapshot(vm)
				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
			})
		})

		Context("and good snapshot exists", func() {
			var err error
			var snapshot *snapshotv1.VirtualMachineSnapshot
			var webhook *admissionregistrationv1.ValidatingWebhookConfiguration

			BeforeEach(func() {
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
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
					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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

				vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore := createRestoreDef(vm.Name, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(BeEmpty())
				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())

				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vm.Spec).To(Equal(*origSpec))

				deleteRestore(restore)
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
									Name:      "nonexistent",
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
				newVM := libvmi.NewVirtualMachine(libvmifact.NewCirros())
				newVM, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), newVM, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				defer deleteVM(newVM)

				By("Creating a VM restore")
				restore := createRestoreDef(newVM.Name, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(newVM.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				events.ExpectEvent(restore, corev1.EventTypeWarning, "VirtualMachineRestoreError")
				Eventually(func() *snapshotv1.VirtualMachineRestoreStatus {
					restore, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return restore.Status
				}, 30*time.Second, 2*time.Second).Should(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Conditions": ContainElements(
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Type":   Equal(snapshotv1.ConditionReady),
							"Status": Equal(corev1.ConditionFalse),
							"Reason": Equal("restore source and restore target are different but restore target already exists")}),
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Type":   Equal(snapshotv1.ConditionProgressing),
							"Status": Equal(corev1.ConditionFalse),
							"Reason": Equal("restore source and restore target are different but restore target already exists")}),
					),
				})))
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

				It("with changed name and MAC address", decorators.StorageCritical, func() {
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
					vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(context.Background(), newVM.Name, metav1.GetOptions{})
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
					_ = libvmops.StartVirtualMachine(newVM)
				})

			})
		})
		Context("with instancetype and preferences", decorators.RequiresSnapshotStorageClass, func() {
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
					Fail("Failing test, no VolumeSnapshot support")
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

				preferredCPUTopology := instancetypev1beta1.Sockets
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

				vm.Spec.Template.Spec.Domain.Resources = v1.ResourceRequirements{}
				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: instancetype.Name,
					Kind: "VirtualMachineInstanceType",
				}
				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: "VirtualMachinePreference",
				}
			})

			DescribeTable("should use existing ControllerRevisions for an existing VM restore", decorators.StorageCritical, Label("instancetype", "preference", "restore"), func(runStrategy v1.VirtualMachineRunStrategy) {
				libvmi.WithRunStrategy(runStrategy)(vm)
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the VM has instancetype and preference RevisionNames")
				Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.HaveControllerRevisionRefs())

				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				originalVMInstancetypeRevisionName := vm.Status.InstancetypeRef.ControllerRevisionRef.Name
				originalVMPreferenceRevisionName := vm.Status.PreferenceRef.ControllerRevisionRef.Name

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					libstorage.EventuallyDVWith(vm.Namespace, dvt.Name, 180, HaveSucceeded())
				}

				By("Creating a VirtualMachineSnapshot")
				snapshot = createSnapshot(vm)

				if runStrategy == v1.RunStrategyAlways {
					By("Stopping the VM")
					vm = libvmops.StopVirtualMachine(vm)
				}

				By("Creating a VirtualMachineRestore")
				restore = createRestoreDef(vm.Name, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the restore completes")
				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)

				By("Asserting that the restored VM has the same instancetype and preference controllerRevisions")
				currVm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(currVm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(originalVMInstancetypeRevisionName))
				Expect(currVm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(originalVMPreferenceRevisionName))
			},
				Entry("with a running VM", v1.RunStrategyAlways),
				Entry("with a stopped VM", v1.RunStrategyHalted),
			)

			DescribeTable("should create new ControllerRevisions for newly restored VM", decorators.StorageCritical, Label("instancetype", "preference", "restore"), func(runStrategy v1.VirtualMachineRunStrategy) {
				libvmi.WithRunStrategy(runStrategy)(vm)
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the VM has instancetype and preference RevisionNames")
				Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.HaveControllerRevisionRefs())

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					libstorage.EventuallyDVWith(vm.Namespace, dvt.Name, 180, HaveSucceeded())
				}

				By("Creating a VirtualMachineSnapshot")
				snapshot = createSnapshot(vm)

				By("Creating a VirtualMachineRestore")
				restoreVMName := vm.Name + "-new"
				restore = createRestoreDefWithMacAddressPatch(vm, restoreVMName, snapshot.Name)

				restore, err = virtClient.VirtualMachineRestore(testsuite.GetTestNamespace(nil)).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the targetVM is finally created")
				_ = expectNewVMCreation(restoreVMName)

				By("Waiting until the restoreVM has instancetype and preference RevisionNames")
				Eventually(matcher.ThisVMWith(testsuite.GetTestNamespace(vm), restoreVMName)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.HaveControllerRevisionRefs())

				By("Asserting that the restoreVM has new instancetype and preference controllerRevisions")
				sourceVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				restoreVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), restoreVMName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(restoreVM.Status.InstancetypeRef.ControllerRevisionRef.Name).ToNot(Equal(sourceVM.Status.InstancetypeRef.ControllerRevisionRef.Name))
				Expect(restoreVM.Status.PreferenceRef.ControllerRevisionRef.Name).ToNot(Equal(sourceVM.Status.PreferenceRef.ControllerRevisionRef.Name))

				By("Asserting that the source and target ControllerRevisions contain the same Object")
				Expect(libinstancetype.EnsureControllerRevisionObjectsEqual(sourceVM.Status.InstancetypeRef.ControllerRevisionRef.Name, restoreVM.Status.InstancetypeRef.ControllerRevisionRef.Name, virtClient)).To(BeTrue(), "source and target instance type controller revisions are expected to be equal")
				Expect(libinstancetype.EnsureControllerRevisionObjectsEqual(sourceVM.Status.PreferenceRef.ControllerRevisionRef.Name, restoreVM.Status.PreferenceRef.ControllerRevisionRef.Name, virtClient)).To(BeTrue(), "source and target preference controller revisions are expected to be equal")
			},
				Entry("with a running VM", v1.RunStrategyAlways),
				Entry("with a stopped VM", v1.RunStrategyHalted),
			)
		})
	})

	Context("[storage-req]", decorators.StorageReq, func() {
		Context("With a more complicated VM", decorators.RequiresSnapshotStorageClass, func() {
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
					Fail("Failing test, no VolumeSnapshot support")
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

			createMessageWithInitialValue := func(login console.LoginToFunction, device string, tpm bool, vmis ...*v1.VirtualMachineInstance) {
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
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: makeTestDirectoryCmd},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: fmt.Sprintf(mountTestDirectoryCmd, device)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
						}...)
					}

					batch = append(batch, []expect.Batcher{
						&expect.BSnd{S: makeTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: chmodTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: fmt.Sprintf("echo '%s' > /test/data/message\n", vm.UID)},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: catTestDataMessageCmd},
						&expect.BExp{R: string(vm.UID)},
						&expect.BSnd{S: syncName},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: syncName},
						&expect.BExp{R: console.PromptExpression},
					}...)

					if tpm {
						batch = append(batch, []expect.Batcher{
							&expect.BSnd{S: fmt.Sprintf("sudo tpm2_createprimary -C o -c %s.ctx\n", "/dev/tpm0")},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},

							&expect.BSnd{S: fmt.Sprintf("sudo tpm2_nvdefine -C o -s %d 1\n", len(string(vm.UID))+1)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},

							&expect.BSnd{S: fmt.Sprintf("sudo tpm2_nvwrite -C o -i /test/data/message 1\n")},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},

							&expect.BSnd{S: fmt.Sprintf("sudo tpm2_nvread -s %d -C o 1\n", len(string(vm.UID)))},
							&expect.BExp{R: string(vm.UID)},
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: syncName},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: syncName},
							&expect.BExp{R: console.PromptExpression},
						}...)
					}

					Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
				}
			}

			updateMessage := func(device string, onlineSnapshot, tpm bool, vmis ...*v1.VirtualMachineInstance) {
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
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
						}...)
					}

					batch = append(batch, []expect.Batcher{
						&expect.BSnd{S: makeTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: chmodTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: catTestDataMessageCmd},
						&expect.BExp{R: string(vm.UID)},
						&expect.BSnd{S: fmt.Sprintf("echo '%s' > /test/data/message\n", snapshot.UID)},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: catTestDataMessageCmd},
						&expect.BExp{R: string(snapshot.UID)},
						&expect.BSnd{S: syncName},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: syncName},
						&expect.BExp{R: console.PromptExpression},
					}...)

					if tpm {
						batch = append(batch, []expect.Batcher{
							&expect.BSnd{S: fmt.Sprintf("sudo tpm2_nvread -s %d -C o 1\n", len(string(vm.UID)))},
							&expect.BExp{R: string(vm.UID)},
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: fmt.Sprintf("sudo tpm2_nvwrite -C o -i /test/data/message 1\n")},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: syncName},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: syncName},
							&expect.BExp{R: console.PromptExpression},
						}...)
					}

					Expect(console.SafeExpectBatch(vmi, batch, 20)).To(Succeed())
				}
			}

			verifyOriginalContent := func(device string, tpm bool, vmis ...*v1.VirtualMachineInstance) {
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
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
						}...)
					}

					batch = append(batch, []expect.Batcher{
						&expect.BSnd{S: makeTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: chmodTestDataDirectoryCmd},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
						&expect.BSnd{S: catTestDataMessageCmd},
						&expect.BExp{R: string(vm.UID)},
						&expect.BSnd{S: console.EchoLastReturnValue},
						&expect.BExp{R: console.RetValue("0")},
					}...)

					if tpm {
						batch = append(batch, []expect.Batcher{
							&expect.BSnd{S: fmt.Sprintf("sudo tpm2_nvread -s %d -C o 1\n", len(string(vm.UID)))},
							&expect.BExp{R: string(vm.UID)},
							&expect.BSnd{S: console.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: syncName},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: syncName},
							&expect.BExp{R: console.PromptExpression},
						}...)
					}

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
				Expect(newVM.Spec.Template.Spec.Domain.Devices.TPM).To(Equal(vm.Spec.Template.Spec.Domain.Devices.TPM))
			}

			createSnapshotAndRestore := func(device string, login console.LoginToFunction, onlineSnapshot bool, tpm bool, targetVMName string, stopVMBeforeRestore bool) {
				isRestoreToDifferentVM := targetVMName != vm.Name

				var targetUID *types.UID
				if !isRestoreToDifferentVM {
					targetUID = &vm.UID
				}

				createMessageWithInitialValue(login, device, tpm, vmi)

				if !onlineSnapshot {
					By(stoppingVM)
					vm = libvmops.StopVirtualMachine(vm)
				}

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				if !onlineSnapshot {
					By("Starting VM")
					vm = libvmops.StartVirtualMachine(vm)
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					Expect(login(vmi)).To(Succeed())
				}

				if !isRestoreToDifferentVM {
					updateMessage(device, onlineSnapshot, tpm, vmi)
				}

				if stopVMBeforeRestore {
					By(stoppingVM)
					vm = libvmops.StopVirtualMachine(vm)
				}

				By("Restoring VM")
				restore = createRestoreDefWithMacAddressPatch(vm, targetVMName, snapshot.Name)
				if vm.Spec.Template.Spec.Domain.Firmware != nil {
					restore.Spec.Patches = append(restore.Spec.Patches, getFirmwareUUIDCloningPatch(vm))
				}

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				if !stopVMBeforeRestore {
					events.ExpectEvent(restore, corev1.EventTypeNormal, "RestoreTargetNotReady")
					By(stoppingVM)
					vm = libvmops.StopVirtualMachine(vm)
				}

				restore = waitRestoreComplete(restore, targetVMName, targetUID)
			}

			doRestoreNoVMStart := func(device string, login console.LoginToFunction, onlineSnapshot, tpm bool, targetVMName string) {
				createSnapshotAndRestore(device, login, onlineSnapshot, tpm, targetVMName, stopVMBeforeRestore)
			}

			doRestoreStopVMAfterRestoreCreate := func(device string, login console.LoginToFunction, onlineSnapshot bool, targetVMName string) {
				createSnapshotAndRestore(device, login, onlineSnapshot, false, targetVMName, stopVMAfterRestore)
			}

			startVMAfterRestore := func(targetVMName, device string, tpm bool, login console.LoginToFunction) {
				isRestoreToDifferentVM := targetVMName != vm.Name
				targetVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), targetVMName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				targetVM = libvmops.StartVirtualMachine(targetVM)
				targetVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(context.Background(), targetVM.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying original file contents")
				Expect(login(targetVMI)).To(Succeed())

				verifyOriginalContent(device, tpm, targetVMI)

				if isRestoreToDifferentVM {
					newVM = targetVM
				} else {
					vm = targetVM
				}
			}

			doRestore := func(device string, login console.LoginToFunction, onlineSnapshot bool, targetVMName string) {
				doRestoreNoVMStart(device, login, onlineSnapshot, false, targetVMName)
				startVMAfterRestore(targetVMName, device, false, login)
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
					Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
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

				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Get(context.Background(), *dvName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				verifyOwnerRef(dv, targetVM.APIVersion, targetVM.Kind, targetVM.Name, targetVM.UID)
				verifyOwnerRef(pvc, "cdi.kubevirt.io/v1beta1", "DataVolume", dv.Name, dv.UID)
			}

			cloneVM := func(sourceVMName, targetVMName string) {
				By("Creating VM clone")
				vmClone := kubecli.NewMinimalCloneWithNS("testclone", testsuite.GetTestNamespace(nil))
				cloneSourceRef := &corev1.TypedLocalObjectReference{
					APIGroup: virtpointer.P(groupName),
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
				Eventually(func() clone.VirtualMachineClonePhase {
					vmClone, err = virtClient.VirtualMachineClone(vmClone.Namespace).Get(context.Background(), vmClone.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					return vmClone.Status.Phase
				}, 3*time.Minute, 3*time.Second).Should(Equal(clone.Succeeded), "clone should finish successfully")
			}

			It("[test_id:5259]should restore a vm multiple from the same snapshot", func() {
				vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskCirros, snapshotStorageClass))

				By(stoppingVM)
				vm = libvmops.StopVirtualMachine(vm)

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

			It("restore should allow grace period for the target to be ready", func() {
				vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskCirros, snapshotStorageClass))

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)
				restore = createRestoreDefWithMacAddressPatch(vm, vm.Name, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				events.ExpectEvent(restore, corev1.EventTypeNormal, "RestoreTargetNotReady")
				By(stoppingVM)
				vm = libvmops.StopVirtualMachine(vm)

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))
			})

			It("restore should stop target if targetReadinessPolicy is StopTarget", func() {
				vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskCirros, snapshotStorageClass))

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)
				restore = createRestoreDefWithMacAddressPatch(vm, vm.Name, snapshot.Name)
				restore.Spec.TargetReadinessPolicy = pointer.P(snapshotv1.VirtualMachineRestoreStopTarget)
				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))

				By("Making sure VM was stopped in the restore")
				Consistently(ThisVM(vm), 60*time.Second, 5*time.Second).Should(Not(BeReady()))

				By("Making sure restored VM is runnable")
				libvmops.StartVirtualMachine(vm)
			})

			// This test is relevant to provisioner which round up the received size of
			// the PVC. Currently we only test vmsnapshot tests which ceph which has this
			// behavior. In case of running this test with other provisioner or if ceph
			// will change this behavior it will fail.
			DescribeTable("should restore a vm with restore size bigger then PVC size", decorators.RequiresSizeRoundUp, func(restoreToNewVM bool) {
				vm = createVMWithCloudInit(cd.ContainerDiskCirros, snapshotStorageClass)
				quantity, err := resource.ParseQuantity("1528Mi")
				Expect(err).ToNot(HaveOccurred())
				vm.Spec.DataVolumeTemplates[0].Spec.Storage.Resources.Requests["storage"] = quantity
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
				resQuantity := content.Spec.VolumeBackups[0].PersistentVolumeClaim.Spec.Resources.Requests["storage"]
				Expect(resQuantity.Value()).To(Equal(quantity.Value()))
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

			DescribeTable("should restore a vm that boots from a datavolumetemplate", decorators.StorageCritical, func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(createVMWithCloudInit(cd.ContainerDiskCirros, snapshotStorageClass))

				originalDVName := vm.Spec.DataVolumeTemplates[0].Name
				doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
				verifyRestore(restoreToNewVM, originalDVName)
			},
				Entry("[test_id:5260] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm that boots from a datavolume (not template)", func(restoreToNewVM bool) {
				vm = createVMWithCloudInit(cd.ContainerDiskCirros, snapshotStorageClass)
				dv := orphanDataVolumeTemplate(vm, 0)
				originalPVCName := dv.Name

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm, vmi = createAndStartVM(vm)
				dv = waitDVReady(dv)
				doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(1))
				if restoreToNewVM {
					checkNewVMEquality()
				}
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
						verifyOwnerRef(pvc, v1.GroupVersion.String(), "VirtualMachine", vm.Name, vm.UID)
					}
				}
			},
				Entry("[test_id:5261] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm that boots from a PVC", func(restoreToNewVM bool) {
				dv := libdv.NewDataVolume(
					libdv.WithName("restore-pvc-"+rand.String(12)),
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), cdiv1.RegistryPullNode),
					libdv.WithStorage(libdv.StorageWithStorageClass(snapshotStorageClass)),
				)

				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				originalPVCName := dv.Name

				memory := "128Mi"
				if checks.IsARM64(testsuite.Arch) {
					memory = "256Mi"
				}
				vmi = libstorage.RenderVMIWithDataVolume(originalPVCName, testsuite.GetTestNamespace(nil),
					libvmi.WithResourceMemory(memory), libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()))
				vm, vmi = createAndStartVM(libvmi.NewVirtualMachine(vmi))

				doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
				Expect(restore.Status.Restores).To(HaveLen(1))
				if restoreToNewVM {
					checkNewVMEquality()
				}

				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty())
				_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), originalPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				targetVM := getTargetVM(restoreToNewVM)
				targetVM, err = virtClient.VirtualMachine(targetVM.Namespace).Get(context.Background(), targetVM.Name, metav1.GetOptions{})
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
				vmi = libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
				vm = libvmi.NewVirtualMachine(vmi)
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
								Resources: corev1.VolumeResourceRequirements{
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
					Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
				}
			},
				Entry("[test_id:5263] to the same VM", false),
				Entry("to a new VM", true),
			)

			DescribeTable("Should restore a vm with backend storage", func(onlineSnapshot bool) {
				vm = createVMWithCloudInit(cd.ContainerDiskFedoraTestTooling, snapshotStorageClass)
				vm.Spec.Template.Spec.Domain.Devices.TPM = &v1.TPMDevice{Persistent: pointer.P(true)}
				vm, vmi = createAndStartVM(vm)
				Eventually(ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(BeReady())

				By("Expecting the creation of a backend storage PVC with the right storage class")
				pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: "persistent-state-for=" + vmi.Name,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(pvcs.Items).To(HaveLen(1))
				pvc := pvcs.Items[0]

				loginFunc := func(vmi *v1.VirtualMachineInstance, timeout ...time.Duration) error {
					// Wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi)).WithTimeout(4 * time.Minute).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
					return console.LoginToFedora(vmi)
				}

				doRestoreNoVMStart("", loginFunc, onlineSnapshot, true, vm.Name)
				startVMAfterRestore(vm.Name, "", true, loginFunc)
				Expect(restore.Status.Restores).To(HaveLen(2))

				By("Expect original backend PVC to be deleted")
				Eventually(func() error {
					_, err := virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), pvc.Name, metav1.GetOptions{})
					return err
				}, 60*time.Second, 5*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
			},
				Entry("with offline snapshot", false),
				Entry("with online snapshot", true),
			)

			DescribeTable("should reject vm start if restore in progress", func(deleteFunc string) {
				vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskCirros, snapshotStorageClass))

				By(stoppingVM)
				vm = libvmops.StopVirtualMachine(vm)

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
									Name:      "nonexistent",
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
					updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
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

				err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Cannot update VM runStrategy until restore %q completes", restore.Name)))

				switch deleteFunc {
				case "deleteWebhook":
					deleteWebhook(webhook)
					webhook = nil

					restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				case "deleteRestore":
					deleteRestore(restore)
				default:
					Fail("Delete function not valid")
				}

				Eventually(func() bool {
					updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVM.Status.RestoreInProgress == nil
				}, 30*time.Second, 3*time.Second).Should(BeTrue())

				vm = libvmops.StartVirtualMachine(vm)
				deleteRestore(restore)
			},
				Entry("and allow it to start after completion", "deleteWebhook"),
				Entry("and allow it to start after vmrestore deletion", "deleteRestore"),
			)

			DescribeTable("should restore a vm from an online snapshot", decorators.StorageCritical, func(restoreToNewVM bool) {
				vm = createVMWithCloudInit(cd.ContainerDiskCirros, snapshotStorageClass)
				vm.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{}
				vm, vmi = createAndStartVM(vm)
				targetVMName := getTargetVMName(restoreToNewVM, newVmName)
				login := console.LoginToCirros

				if !restoreToNewVM {
					// Expect to get event in case we stop
					// the VM after we created the restore.
					// Once VM is stopped the VMRestore will
					// continue and complete successfully
					doRestoreStopVMAfterRestoreCreate("", login, onlineSnapshot, targetVMName)
				} else {
					doRestoreNoVMStart("", login, onlineSnapshot, false, targetVMName)
				}
				startVMAfterRestore(targetVMName, "", false, login)
				Expect(restore.Status.Restores).To(HaveLen(1))
				if restoreToNewVM {
					checkNewVMEquality()
				}

			},
				Entry("[test_id:6053] to the same VM, stop VM after create restore", false),
				Entry("to a new VM", true),
			)

			DescribeTable("should restore a vm from an online snapshot with guest agent", func(restoreToNewVM bool) {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
				vmi.Namespace = testsuite.GetTestNamespace(nil)
				vm = libvmi.NewVirtualMachine(vmi)
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
								Resources: corev1.VolumeResourceRequirements{
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

			DescribeTable("should restore an online vm snapshot that boots from a datavolumetemplate with guest agent", decorators.StorageCritical, func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(createVMWithCloudInit(cd.ContainerDiskFedoraTestTooling, snapshotStorageClass, libvmi.WithResourceMemory("512Mi")))
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
				vm, vmi = createAndStartVM(createVMWithCloudInit(cd.ContainerDiskFedoraTestTooling, snapshotStorageClass, libvmi.WithResourceMemory("512Mi")))
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Updating the VM template spec")
				initialMemory := vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]
				newMemory := resource.MustParse("2Gi")
				Expect(newMemory).ToNot(Equal(initialMemory))

				patchSet := patch.New(
					patch.WithReplace("/spec/template/spec/domain/resources/requests/"+string(corev1.ResourceMemory), newMemory),
				)
				patchData, err := patchSet.GeneratePayload()
				Expect(err).NotTo(HaveOccurred())

				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				newVM = libvmops.StopVirtualMachine(updatedVM)
				newVM = libvmops.StartVirtualMachine(newVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(newMemory))

				newVM = libvmops.StopVirtualMachine(newVM)

				By("Restoring VM")
				restore = createRestoreDefWithMacAddressPatch(vm, newVM.Name, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				restore = waitRestoreComplete(restore, newVM.Name, &newVM.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))

				libvmops.StartVirtualMachine(newVM)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]).To(Equal(initialMemory))
			})

			It("should restore an already cloned virtual machine", func() {
				vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskFedoraTestTooling, snapshotStorageClass))

				targetVMName := vm.Name + "-clone"
				cloneVM(vm.Name, targetVMName)

				By(fmt.Sprintf("Getting the cloned VM %s", targetVMName))
				targetVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), targetVMName, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				By(creatingSnapshot)
				targetVM = libvmops.StartVirtualMachine(targetVM)
				snapshot = createSnapshot(targetVM)
				newVM = libvmops.StopVirtualMachine(targetVM)

				By("Restoring cloned VM")
				restore = createRestoreDefWithMacAddressPatch(vm, newVM.Name, snapshot.Name)
				restore, err = virtClient.VirtualMachineRestore(targetVM.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				restore = waitRestoreComplete(restore, newVM.Name, &newVM.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))
			})

			DescribeTable("should restore vm with hot plug disks", func(restoreToNewVM bool) {
				vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskFedoraTestTooling, snapshotStorageClass))
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
				targetVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(context.Background(), targetVM.Name, metav1.GetOptions{})
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

			It("should override VM during restore", func() {
				// Create a VM and snapshot it
				vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskCirros, snapshotStorageClass))
				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				const newDiskName = "new-disk"

				// Create the restore definition of the VM, change the name of the restored volume
				restoreDef := createRestoreDef(vm.Name, snapshot.Name)
				restoreDef.Spec.TargetReadinessPolicy = ptr.To(snapshotv1.VirtualMachineRestoreStopTarget)
				restoreDef.Spec.VolumeRestoreOverrides = []snapshotv1.VolumeRestoreOverride{
					{
						VolumeName:  "disk0",
						RestoreName: newDiskName,
						Labels:      map[string]string{"new-label": "value"},
						Annotations: map[string]string{"new-annotation": "value"},
					},
				}

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restoreDef, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))

				// Check the VM post-restore has info we want
				restoreVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restoreVM.Spec.Template.Spec.Volumes).To(HaveLen(1))
				Expect(restoreVM.Spec.Template.Spec.Volumes[0].Name).To(Equal("disk0"))
				Expect(restoreVM.Spec.Template.Spec.Volumes[0].DataVolume.Name).To(Equal(newDiskName))

				// Check the restored PVC has the info we want
				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), newDiskName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc.Labels["new-label"]).To(Equal("value"))
				Expect(pvc.Annotations["new-annotation"]).To(Equal("value"))

				deleteRestore(restore)
				restore = nil
			})

			It("should restore with volume restore policy InPlace and DV template as disk", func() {
				// Create a VM and snapshot it
				vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskCirros, snapshotStorageClass))
				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				restoreDef := createRestoreDef(vm.Name, snapshot.Name)
				restoreDef.Spec.TargetReadinessPolicy = ptr.To(snapshotv1.VirtualMachineRestoreStopTarget)

				// We want to overwrite existing volumes during the restore, that means deleting the existing PVCs
				restoreDef.Spec.VolumeRestorePolicy = ptr.To(snapshotv1.VolumeRestorePolicyInPlace)

				// We're about to restore a VM in such a way that the restored volumes are identical to the source volumes.
				// We need to make sure they're indeed new, and not that nothing happened during the test. We add
				// a special annotation to the restored volume to ensure it has been restored from a VolumeSnapshot.
				restoreDef.Spec.VolumeRestoreOverrides = []snapshotv1.VolumeRestoreOverride{
					{
						VolumeName: "disk0",
						Annotations: map[string]string{
							"test": "value",
						},
					},
				}

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restoreDef, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))

				// Check the VM post-restore has info we want
				restoreVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restoreVM.Spec.Template.Spec.Volumes).To(HaveLen(1))
				Expect(restoreVM.Spec.Template.Spec.Volumes[0].Name).
					To(Equal(vm.Spec.Template.Spec.Volumes[0].Name)) // Volume name didn't change
				Expect(restoreVM.Spec.Template.Spec.Volumes[0].DataVolume.Name).
					To(Equal(vm.Spec.Template.Spec.Volumes[0].DataVolume.Name)) // Related DV didn't change

				originalPvcName := vm.Spec.Template.Spec.Volumes[0].DataVolume.Name // PVC has same name as original DV

				// The original VM should have information on the name of the PVC linked to the DV, check it's accurate
				Expect(vmi.Status.VolumeStatus[0].Name).To(Equal("disk0"))
				Expect(vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo.ClaimName).
					To(Equal(originalPvcName))

				// Check the restored PVC exists
				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), originalPvcName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc.Annotations["test"]).To(Equal("value")) // Ensure new annotation is present

				// PVC should have owner reference back to the DV (which has same name as itself)
				Expect(pvc.OwnerReferences).ToNot(BeNil())
				Expect(pvc.OwnerReferences[0].Name).To(Equal(originalPvcName))

				// Check the source DV for that PVC
				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), originalPvcName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(dv.Annotations["restore.kubevirt.io/name"]).To(Equal(restore.Name))
				Expect(dv.Annotations[cdiv1.AnnPrePopulated]).To(Equal(originalPvcName))

				// Start VM
				targetVM := libvmops.StartVirtualMachine(restoreVM)
				restoreVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(context.Background(), targetVM.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Check the restored VM is exactly the same, the PVC is the same
				Expect(restoreVMI.Status.VolumeStatus[0].Name).To(Equal("disk0"))
				Expect(restoreVMI.Status.VolumeStatus[0].
					PersistentVolumeClaimInfo.ClaimName).To(Equal(originalPvcName))

				deleteRestore(restore)
				restore = nil
			})

			It("should restore with volume restore policy InPlace and DV (not template) as disk", func() {
				// VM with normal DV mounted to it
				vm = createVMWithCloudInit(cd.ContainerDiskCirros, snapshotStorageClass)

				// Create standalone DV, not linked to a VM's template
				dv := orphanDataVolumeTemplate(vm, 0)
				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				originalPVCName := dv.Name

				// Create and start the VM, wait for the DV to bind
				vm, _ = createAndStartVM(vm)
				dv = waitDVReady(dv)

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				restoreDef := createRestoreDef(vm.Name, snapshot.Name)
				restoreDef.Spec.TargetReadinessPolicy = ptr.To(snapshotv1.VirtualMachineRestoreStopTarget)

				// We want to overwrite existing volumes during the restore, that means deleting the existing PVCs
				restoreDef.Spec.VolumeRestorePolicy = ptr.To(snapshotv1.VolumeRestorePolicyInPlace)

				// We're about to restore a VM in such a way that the restored volumes are identical to the source volumes.
				// We need to make sure they're indeed new, and not that nothing happened during the test. We add
				// a special annotation to the restored volume to ensure it has been restored from a VolumeSnapshot.
				restoreDef.Spec.VolumeRestoreOverrides = []snapshotv1.VolumeRestoreOverride{
					{
						VolumeName: "disk0",
						Annotations: map[string]string{
							"test": "value",
						},
					},
				}

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restoreDef, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))
				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty()) // This is handled only for DV templates, so we expect nothing

				// Check the VM post-restore has info we want
				restoreVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restoreVM.Spec.Template.Spec.Volumes).To(HaveLen(2))
				Expect(restoreVM.Spec.Template.Spec.Volumes[0].Name).
					To(Equal(vm.Spec.Template.Spec.Volumes[0].Name)) // Volume name didn't change
				Expect(restoreVM.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).
					To(Equal(vm.Spec.Template.Spec.Volumes[0].DataVolume.Name)) // DV got converted to a PVC with the same name

				// Check the restored PVC exists
				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), originalPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc.Annotations["test"]).To(Equal("value")) // Ensure new annotation is present

				// PVC should have owner reference back to the DV (which has same name as itself)
				Expect(pvc.OwnerReferences).ToNot(BeNil())
				Expect(pvc.OwnerReferences[0].Name).To(Equal(originalPVCName))

				// Check the source DV for that PVC
				restoredDV, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), originalPVCName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restoredDV.Annotations["restore.kubevirt.io/name"]).To(Equal(restore.Name))
				Expect(restoredDV.Annotations[cdiv1.AnnPrePopulated]).To(Equal(originalPVCName))
			})

			It("should restore with volume restore policy InPlace and PVC as disk", func() {
				// VM with normal DV mounted to it
				vm = createVMWithCloudInit(cd.ContainerDiskCirros, snapshotStorageClass)
				vm.Spec.DataVolumeTemplates = nil // Remove traces of DV, we want a raw PVC
				vm.Spec.Template.Spec.Volumes = vm.Spec.Template.Spec.Volumes[1:]

				// Create and mount PVC to VM
				pvcName := "standalone-pvc"
				pvc := &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pvcName,
						Namespace: vm.Namespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								"storage": resource.MustParse("2Gi"),
							},
						},
					},
				}

				vm.Spec.Template.Spec.Volumes = append([]v1.Volume{{
					Name: "disk0",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				}}, vm.Spec.Template.Spec.Volumes...)

				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm, _ = createAndStartVM(vm)

				By(creatingSnapshot)
				snapshot = createSnapshot(vm)

				restoreDef := createRestoreDef(vm.Name, snapshot.Name)
				restoreDef.Spec.TargetReadinessPolicy = ptr.To(snapshotv1.VirtualMachineRestoreStopTarget)

				// We want to overwrite existing volumes during the restore, that means deleting the existing PVCs
				restoreDef.Spec.VolumeRestorePolicy = ptr.To(snapshotv1.VolumeRestorePolicyInPlace)

				// We're about to restore a VM in such a way that the restored volumes are identical to the source volumes.
				// We need to make sure they're indeed new, and not that nothing happened during the test. We add
				// a special annotation to the restored volume to ensure it has been restored from a VolumeSnapshot.
				restoreDef.Spec.VolumeRestoreOverrides = []snapshotv1.VolumeRestoreOverride{
					{
						VolumeName: "disk0",
						Annotations: map[string]string{
							"test": "value",
						},
					},
				}

				restore, err = virtClient.VirtualMachineRestore(vm.Namespace).Create(context.Background(), restoreDef, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				restore = waitRestoreComplete(restore, vm.Name, &vm.UID)
				Expect(restore.Status.Restores).To(HaveLen(1))
				Expect(restore.Status.DeletedDataVolumes).To(BeEmpty()) // This is handled only for DV templates, so we expect nothing

				// Check the VM post-restore has info we want
				restoreVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restoreVM.Spec.Template.Spec.Volumes).To(HaveLen(2))
				Expect(restoreVM.Spec.Template.Spec.Volumes[0].Name).
					To(Equal(vm.Spec.Template.Spec.Volumes[0].Name)) // Volume name didn't change
				Expect(restoreVM.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).
					To(Equal(pvcName)) // PVC name didn't change

				// Check the restored PVC exists
				pvc, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc.Annotations["test"]).To(Equal("value")) // Ensure new annotation is present

				// PVC should have owner reference back to the VM
				Expect(pvc.OwnerReferences).To(HaveLen(1))
				Expect(pvc.OwnerReferences[0].Kind).To(Equal("VirtualMachine"))
				Expect(pvc.OwnerReferences[0].Name).To(Equal(restoreVM.Name))
      })

      It("with run strategy and snapshot should successfully restore", func() {
				vm = renderVMWithRegistryImportDataVolume(cd.ContainerDiskFedoraTestTooling, snapshotStorageClass)
				libvmi.WithRunStrategy(v1.RunStrategyRerunOnFailure)(vm)
				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
				vm = libvmops.StartVirtualMachine(vm)
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 360).Should(BeInPhase(v1.Running))
				Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyRerunOnFailure)))
				doRestoreNoVMStart("", console.LoginToFedora, onlineSnapshot, false, vm.Name)
				Expect(restore.Status.Restores).To(HaveLen(1))
				restoredVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(restoredVM.Spec.RunStrategy).To(Equal(pointer.P(v1.RunStrategyRerunOnFailure)))
			})

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
					Expect(err).ToNot(HaveOccurred())
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
						updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						if updatedVM.Status.MemoryDumpRequest == nil ||
							updatedVM.Status.MemoryDumpRequest.Phase != v1.MemoryDumpCompleted {
							return false
						}

						return true
					}, 60*time.Second, time.Second).Should(BeTrue())
				}

				DescribeTable("should not restore memory dump volume", func(restoreToNewVM bool) {
					vm, vmi = createAndStartVM(renderVMWithRegistryImportDataVolume(cd.ContainerDiskFedoraTestTooling, snapshotStorageClass))
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					By("Get VM memory dump")
					getMemoryDump(vm.Name, vm.Namespace, memoryDumpPVCName)
					waitMemoryDumpCompletion(vm)

					doRestoreNoVMStart("", console.LoginToFedora, onlineSnapshot, false, getTargetVMName(restoreToNewVM, newVmName))
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

					startVMAfterRestore(getTargetVMName(restoreToNewVM, newVmName), "", false, console.LoginToFedora)

					targetVM := getTargetVM(restoreToNewVM)
					targetVMI, err := virtClient.VirtualMachineInstance(targetVM.Namespace).Get(context.Background(), targetVM.Name, metav1.GetOptions{})
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
				var forcedHostAssistedScName string

				BeforeEach(func() {
					sc, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), snapshotStorageClass, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					hostAssistedSc := sc.DeepCopy()
					hostAssistedSc.ObjectMeta = metav1.ObjectMeta{
						GenerateName: fmt.Sprintf("%s-force-host-assisted", snapshotStorageClass),
						Labels: map[string]string{
							cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(nil)): "",
						},
						Annotations: map[string]string{
							"cdi.kubevirt.io/clone-strategy": string(cdiv1.CloneStrategyHostAssisted),
						},
					}
					sc, err = virtClient.StorageV1().StorageClasses().Create(context.Background(), hostAssistedSc, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					forcedHostAssistedScName = sc.Name

					source := libdv.NewDataVolume(
						libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), cdiv1.RegistryPullNode),
						libdv.WithStorage(libdv.StorageWithStorageClass(forcedHostAssistedScName)),
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
					// Make sure we recreate the alternative namespace when completing the tests
					_, err := virtClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testsuite.NamespaceTestAlternative}}, metav1.CreateOptions{})
					if err != nil && !errors.IsAlreadyExists(err) {
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
					if forcedHostAssistedScName != "" {
						err := virtClient.StorageV1().StorageClasses().Delete(context.Background(), forcedHostAssistedScName, metav1.DeleteOptions{})
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
						libdv.WithStorage(libdv.StorageWithStorageClass(forcedHostAssistedScName), libdv.StorageWithVolumeSize("1Gi")),
					)

					return libvmi.NewVirtualMachine(
						libstorage.RenderVMIWithDataVolume(dataVolume.Name, testsuite.GetTestNamespace(nil), libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot())),
						libvmi.WithDataVolumeTemplate(dataVolume),
					)
				}

				DescribeTable("should restore a vm that boots from a network cloned datavolumetemplate", func(restoreToNewVM, deleteSourcePVC, deleteSourceNamespace bool) {
					vm, vmi = createAndStartVM(createNetworkCloneVMFromSource())

					checkCloneAnnotations(vm, true)
					if deleteSourceNamespace {
						err = virtClient.CoreV1().Namespaces().Delete(context.Background(), testsuite.NamespaceTestAlternative, metav1.DeleteOptions{})
						Expect(err).ToNot(HaveOccurred())
						Eventually(func() error {
							_, err := virtClient.CoreV1().Namespaces().Get(context.Background(), testsuite.NamespaceTestAlternative, metav1.GetOptions{})
							return err
						}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

						sourceDV = nil
						cloneRole = nil
						cloneRoleBinding = nil
					} else if deleteSourcePVC {
						err := virtClient.CdiClient().CdiV1beta1().DataVolumes(sourceDV.Namespace).Delete(context.Background(), sourceDV.Name, metav1.DeleteOptions{})
						Expect(err).ToNot(HaveOccurred())
					}

					doRestore("", console.LoginToCirros, offlineSnaphot, getTargetVMName(restoreToNewVM, newVmName))
					checkCloneAnnotations(getTargetVM(restoreToNewVM), false)
				},
					Entry("to the same VM", false, false, false),
					Entry("to a new VM", true, false, false),
					Entry("to the same VM, no source pvc", false, true, false),
					Entry("to a new VM, no source pvc", true, true, false),
					Entry("to the same VM, no source namespace", false, false, true),
					Entry("to a new VM, no source namespace", true, false, true),
				)

				DescribeTable("should restore a vm that boots from a network cloned datavolume (not template)", func(restoreToNewVM, deleteSourcePVC bool) {
					vm = createNetworkCloneVMFromSource()
					dv := orphanDataVolumeTemplate(vm, 0)

					dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vm, vmi = createAndStartVM(vm)
					waitDVReady(dv)

					checkCloneAnnotations(vm, true)
					if deleteSourcePVC {
						err := virtClient.CdiClient().CdiV1beta1().DataVolumes(sourceDV.Namespace).Delete(context.Background(), sourceDV.Name, metav1.DeleteOptions{})
						Expect(err).ToNot(HaveOccurred())
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
}))
