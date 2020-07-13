package tests_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VirtualMachineSnapshot Tests", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	groupName := "kubevirt.io/v1alpha3"

	Context("With simple VM", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			var err error
			vmiImage := tests.ContainerDiskFor(tests.ContainerDiskCirros)
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")
			vm = tests.NewRandomVirtualMachine(vmi, false)
			vm, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should successfully create a snapshot", func() {
			snapshotName := "snapshot-" + vm.Name
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      snapshotName,
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

			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(snapshot)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(snapshotName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse
			}, 180*time.Second, time.Second).Should(BeTrue())

			contentName := *snapshot.Status.VirtualMachineSnapshotContentName
			content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(contentName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshotName))
			Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
			Expect(content.Spec.VolumeBackups).To(BeEmpty())
		})

		It("should not create a snapshot when VM is running", func() {
			patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
			vm, err := virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch)
			Expect(err).ToNot(HaveOccurred())

			snapshotName := "snapshot-" + vm.Name
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      snapshotName,
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

			_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(snapshot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(fmt.Sprintf("VirtualMachine \"%s\" is running", vm.Name)))
		})
	})

	Context("With more complicated VM", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			sc, err := getSnapshotStorageClass(virtClient)
			Expect(err).ToNot(HaveOccurred())

			if sc == "" {
				Skip("Skiping test, no VolumeSnapshot support")
			}

			running := false
			vm = tests.NewRandomVMWithDataVolumeInStorageClass(
				tests.GetUrl(tests.AlpineHttpUrl),
				tests.NamespaceTestDefault,
				sc,
			)
			vm.Spec.Running = &running
		})

		It("should successfully create a snapshot", func() {
			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			for _, dvt := range vm.Spec.DataVolumeTemplates {
				Eventually(func() bool {
					dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(dvt.Name, metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					Expect(dv.Status.Phase).ShouldNot(Equal(cdiv1.Failed))
					return dv.Status.Phase == cdiv1.Succeeded
				}, 180*time.Second, time.Second).Should(BeTrue())
			}

			snapshotName := "snapshot-" + vm.Name
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      snapshotName,
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

			_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(snapshot)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(snapshotName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse
			}, 180*time.Second, time.Second).Should(BeTrue())

			Expect(snapshot.Status.CreationTime).ToNot(BeNil())
			contentName := *snapshot.Status.VirtualMachineSnapshotContentName
			content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(contentName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshotName))
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
						Expect(vol.Name).To(Equal(vb.DiskName))

						pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(vol.DataVolume.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						Expect(pvc.Spec).To(Equal(vb.PersistentVolumeClaim.Spec))

						Expect(vb.VolumeSnapshotName).ToNot(BeNil())
						vs, err := virtClient.
							KubernetesSnapshotClient().
							SnapshotV1beta1().
							VolumeSnapshots(vm.Namespace).
							Get(*vb.VolumeSnapshotName, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						Expect(*vs.Spec.Source.PersistentVolumeClaimName).Should(Equal(vol.DataVolume.Name))
						Expect(vs.Status.Error).To(BeNil())
						Expect(*vs.Status.ReadyToUse).To(BeTrue())
					}
				}
				Expect(found).To(BeTrue())
			}
		})
	})
})

func getSnapshotStorageClass(client kubecli.KubevirtClient) (string, error) {
	crd, err := client.
		ExtensionsClient().
		ApiextensionsV1beta1().
		CustomResourceDefinitions().
		Get("volumesnapshotclasses.snapshot.storage.k8s.io", metav1.GetOptions{})
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

	volumeSnapshotClasses, err := client.KubernetesSnapshotClient().SnapshotV1beta1().VolumeSnapshotClasses().List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	if len(volumeSnapshotClasses.Items) > 0 {
		storageClasses, err := client.StorageV1().StorageClasses().List(metav1.ListOptions{})
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
