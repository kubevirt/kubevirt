package storage

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = SIGDescribe("[Serial]VirtualMachineSnapshot Tests", func() {

	var (
		err        error
		virtClient kubecli.KubevirtClient
		vm         *v1.VirtualMachine
		snapshot   *snapshotv1.VirtualMachineSnapshot
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

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	AfterEach(func() {
		Eventually(func() bool {
			if vm != nil {
				err := virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
				if errors.IsNotFound(err) {
					vm = nil
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			}
			if snapshot != nil {
				err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Delete(context.Background(), snapshot.Name, metav1.DeleteOptions{})
				if errors.IsNotFound(err) {
					snapshot = nil
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			}
			return vm == nil && snapshot == nil
		}, 90*time.Second, time.Second).Should(BeTrue())
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

			if *vm.Spec.Running {
				expectedIndications := []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotNoGuestAgentIndication}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))
			} else {
				Expect(snapshot.Status.Indications).To(BeEmpty())
			}

			contentName := *snapshot.Status.VirtualMachineSnapshotContentName
			content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
			Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
			Expect(content.Spec.VolumeBackups).To(BeEmpty())
		}

		It("[test_id:4609]should successfully create a snapshot", func() {
			createAndVerifyVMSnapshot(vm)
		})

		It("[test_id:4610]create a snapshot when VM is running should succeed", func() {
			patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch)
			Expect(err).ToNot(HaveOccurred())
			Expect(*vm.Spec.Running).Should(BeTrue())

			createAndVerifyVMSnapshot(vm)
		})

		It("VM should contain snapshot status for all volumes", func() {
			patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
			vm, err := virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch)
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

			It("with volumes and guest agent available", func() {
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

				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				expectedIndications := []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotGuestAgentIndication}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))
			})

			It("with volumes and no guest agent available", func() {
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

				expectedIndications := []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotNoGuestAgentIndication}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))
			})

			It("without volumes with guest agent available", func() {
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

				expectedIndications := []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotNoGuestAgentIndication}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.VolumeBackups).Should(BeEmpty())
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
					tests.GetUrl(tests.AlpineHttpUrl),
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

	if len(volumeSnapshotClasses.Items) > 0 {
		storageClasses, err := client.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return "", err
		}

		for _, sc := range storageClasses.Items {
			if sc.Provisioner == volumeSnapshotClasses.Items[0].Driver {
				return sc.Name, nil
			}
		}
	}

	return "", nil
}
