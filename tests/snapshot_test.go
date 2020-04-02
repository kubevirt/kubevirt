package tests_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	vmsnapshotv1alpha1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VirtualMachineSnapshot Tests", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

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
			snapshot := &vmsnapshotv1alpha1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      snapshotName,
					Namespace: vm.Namespace,
				},
				Spec: vmsnapshotv1alpha1.VirtualMachineSnapshotSpec{
					Source: vmsnapshotv1alpha1.VirtualMachineSnapshotSource{
						VirtualMachineName: &vm.Name,
					},
				},
			}

			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(snapshot)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(snapshotName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse
			}, 60*time.Second, time.Second).Should(BeTrue())

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
			snapshot := &vmsnapshotv1alpha1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      snapshotName,
					Namespace: vm.Namespace,
				},
				Spec: vmsnapshotv1alpha1.VirtualMachineSnapshotSpec{
					Source: vmsnapshotv1alpha1.VirtualMachineSnapshotSource{
						VirtualMachineName: &vm.Name,
					},
				},
			}

			_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(snapshot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(fmt.Sprintf("VirtualMachine \"%s\" is running", vm.Name)))
		})
	})
})
