package render_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/render"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("render", func() {
	const launcherImage = "kubevirt/virt-launcher:test"

	opts := func() render.Options {
		return render.Options{LauncherImage: launcherImage}
	}

	Describe("NewVMI", func() {
		It("copies name, namespace, owner refs and a stable firmware UUID", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("ns"),
				libvmi.WithName("myvm"),
			))

			vmi := render.NewVMI(vm)

			Expect(vmi.Name).To(Equal("myvm"))
			Expect(vmi.Namespace).To(Equal("ns"))
			Expect(vmi.OwnerReferences).To(HaveLen(1))
			Expect(vmi.OwnerReferences[0].Name).To(Equal(vm.Name))
			Expect(vmi.Spec.Domain.Firmware).NotTo(BeNil())
			Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(render.FirmwareUUID("myvm")))
		})
	})

	Describe("AutoAttachInputDevice", func() {
		It("adds a default input when requested and none exist", func() {
			vmi := libvmi.New()
			vmi.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)

			render.AutoAttachInputDevice(vmi)

			Expect(vmi.Spec.Domain.Devices.Inputs).To(ConsistOf(v1.Input{Name: "default-0"}))
		})

		It("does not add an input when one is already present", func() {
			vmi := libvmi.New()
			vmi.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{{Name: "existing"}}

			render.AutoAttachInputDevice(vmi)

			Expect(vmi.Spec.Domain.Devices.Inputs).To(ConsistOf(v1.Input{Name: "existing"}))
		})
	})

	Describe("PodFromVMI", func() {
		It("does not mutate the caller-owned VMI", func() {
			vmi := libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("keep-me"))
			original := vmi.DeepCopy()

			_, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi).To(Equal(original))
		})

		It("returns the defaulted VMI that matches the Pod", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("testrender"),
				libvmi.WithContainerDisk("disk0", "some/image"),
			)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI).NotTo(BeNil())
			Expect(result.Pod).NotTo(BeNil())
			Expect(result.Pod.Spec.Containers).NotTo(BeEmpty())
			Expect(result.Pod.Spec.Containers[0].Name).To(Equal("compute"))
			Expect(result.Pod.Spec.Containers[0].Image).To(Equal(launcherImage))
		})

		It("keeps GenerateName instead of inventing a Pod Name", func() {
			vmi := libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("named"))

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.GenerateName).NotTo(BeEmpty())
		})
	})

	Describe("PodFromVM", func() {
		It("returns an error when the VM has no template", func() {
			vm := &v1.VirtualMachine{ObjectMeta: metav1.ObjectMeta{Name: "empty"}}

			_, err := render.PodFromVM(vm, opts())
			Expect(err).To(MatchError(ContainSubstring("no template spec")))
		})

		It("renders a compute container from a VM definition", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("fromvm"),
				libvmi.WithContainerDisk("disk0", "some/image"),
			))

			result, err := render.PodFromVM(vm, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI.Name).To(Equal("fromvm"))
			Expect(result.Pod.Spec.Containers[0].Name).To(Equal("compute"))
		})
	})

	Describe("PVC volume mode", func() {
		It("uses a caller-supplied Block PVC", func() {
			block := k8sv1.PersistentVolumeBlock
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("blockvm"),
				libvmi.WithPersistentVolumeClaim("disk0", "my-pvc"),
			)

			result, err := render.PodFromVMI(vmi, render.Options{
				LauncherImage: launcherImage,
				PVCs: []*k8sv1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{Name: "my-pvc", Namespace: "default"},
					Spec: k8sv1.PersistentVolumeClaimSpec{
						AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
						VolumeMode:  &block,
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred())

			var foundBlock bool
			for _, c := range result.Pod.Spec.Containers {
				if c.Name != "compute" {
					continue
				}
				for _, vd := range c.VolumeDevices {
					if vd.Name == "disk0" {
						foundBlock = true
					}
				}
			}
			Expect(foundBlock).To(BeTrue(), "expected a volumeDevice for the block PVC")
		})
	})

	Describe("offline vs online equivalence", func() {
		It("produces the same compute container image and volume count", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("equiv"),
				libvmi.WithContainerDisk("disk0", "some/image"),
			)

			offline, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())

			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{Name: "kubevirt", Namespace: "kubevirt"},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{},
					},
				},
			}
			config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)
			pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			svc := services.NewTemplateService(
				launcherImage, 240,
				"/var/run/kubevirt",
				"/var/run/kubevirt-ephemeral-disks",
				"/var/run/kubevirt/container-disks",
				v1.HotplugDiskDir,
				"",
				pvcCache,
				nil,
				config,
				107,
				"quay.io/kubevirt/vm-export:latest",
				cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
				cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
			)

			onlineVMI := vmi.DeepCopy()
			Expect(mutators.ApplyNewVMIMutations(onlineVMI, config)).To(Succeed())
			onlinePod, err := svc.RenderLaunchManifest(onlineVMI)
			Expect(err).NotTo(HaveOccurred())

			Expect(offline.Pod.Spec.Containers).To(HaveLen(len(onlinePod.Spec.Containers)))
			Expect(offline.Pod.Spec.Containers[0].Image).To(Equal(onlinePod.Spec.Containers[0].Image))
			Expect(offline.Pod.Spec.Volumes).To(HaveLen(len(onlinePod.Spec.Volumes)))
		})
	})
})

// compile-time check that the real ClusterConfig still satisfies the
// mutator interface used by PrepareVMI.
var _ mutators.ClusterConfigProvider = (*virtconfig.ClusterConfig)(nil)
