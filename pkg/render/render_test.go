package render_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/render"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("render", func() {
	const launcherImage = "kubevirt/virt-launcher:test"

	opts := func() render.Options {
		return render.Options{LauncherImage: launcherImage}
	}

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

		It("requires LauncherImage", func() {
			vmi := libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("noimage"))

			_, err := render.PodFromVMI(vmi, render.Options{})
			Expect(err).To(MatchError(ContainSubstring("LauncherImage is required")))
		})

		It("rejects ConfigMap-backed hook sidecars", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("cmsidecar"),
				libvmi.WithAnnotation(hooks.HookSidecarListAnnotationName,
					`[{"configMap":{"name":"hook-cm","key":"hook.sh","hookPath":"/usr/bin/onDefineDomain"}}]`),
			)

			_, err := render.PodFromVMI(vmi, opts())
			Expect(err).To(MatchError(ContainSubstring("ConfigMap-backed hook sidecars")))
		})

		It("renders an image-backed hook sidecar", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("imgsidecar"),
				libvmi.WithAnnotation(hooks.HookSidecarListAnnotationName,
					`[{"image":"kubevirt/sidecar:test","imagePullPolicy":"IfNotPresent"}]`),
			)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Containers).To(ContainElement(HaveField("Image", "kubevirt/sidecar:test")))
		})

		It("defaults the auto-attached input type", func() {
			vmi := libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("inputvm"))
			vmi.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI.Spec.Domain.Devices.Inputs).To(HaveLen(1))
			Expect(result.VMI.Spec.Domain.Devices.Inputs[0].Type).NotTo(BeEmpty())
		})

		It("renders a HostDisk volume", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("hostdisk"),
				libvmi.WithHostDisk("hd0", "/mnt/data/disk.img", v1.HostDiskExistsOrCreate),
			)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Volumes).To(ContainElement(HaveField("Name", "hd0")))
		})

		It("renders a CloudInit NoCloud volume", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("cloudinit"),
				libvmi.WithCloudInitNoCloud(cloudinit.WithNoCloudUserData("#cloud-config\n")),
			)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI.Spec.Volumes).To(ContainElement(HaveField("CloudInitNoCloud", Not(BeNil()))))
		})

		It("annotates a Multus secondary network", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("multinet"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding())),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.NewInterface("net1", libvmi.WithBridgeBinding())),
				libvmi.WithNetwork(&v1.Network{
					Name: "net1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "other-net"},
					},
				}),
			)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Annotations).To(HaveKey("k8s.v1.cni.cncf.io/networks"))
		})

		It("applies RunAsUser to the pod security context", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("uidvm"),
				libvmi.WithContainerDisk("disk0", "some/image"),
			)

			result, err := render.PodFromVMI(vmi, render.Options{
				LauncherImage: launcherImage,
				RunAsUser:     1000,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.SecurityContext).NotTo(BeNil())
			Expect(result.Pod.Spec.SecurityContext.RunAsUser).To(HaveValue(Equal(int64(1000))))
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

		It("rejects a VM with an instancetype matcher", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("itvm")),
				libvmi.WithInstancetype("u1.small"),
			)

			_, err := render.PodFromVM(vm, opts())
			Expect(err).To(MatchError(ContainSubstring("instancetype matchers")))
		})

		It("rejects a VM with a preference matcher", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("prefvm")),
				libvmi.WithPreference("fedora"),
			)

			_, err := render.PodFromVM(vm, opts())
			Expect(err).To(MatchError(ContainSubstring("preference matchers")))
		})

		It("applies a supplied instancetype spec", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("itvm")),
				libvmi.WithInstancetype("u1.small"),
			)

			result, err := render.PodFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Instancetype: &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU:    instancetypev1beta1.CPUInstancetype{Guest: 2},
					Memory: instancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("2Gi")},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI.Spec.Domain.CPU).NotTo(BeNil())
			Expect(result.VMI.Spec.Domain.CPU.Sockets).To(Equal(uint32(2)))
			Expect(result.VMI.Spec.Domain.Memory).NotTo(BeNil())
			Expect(result.VMI.Spec.Domain.Memory.Guest.String()).To(Equal("2Gi"))
			Expect(result.VMI.Annotations).To(HaveKey(v1.InstancetypeAnnotation))
		})

		It("applies a supplied preference auto-attach input", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("prefvm")),
				libvmi.WithPreference("fedora"),
			)

			result, err := render.PodFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Preference: &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Devices: &instancetypev1beta1.DevicePreferences{
						PreferredAutoattachInputDevice: pointer.P(true),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI.Spec.Domain.Devices.Inputs).To(HaveLen(1))
			Expect(result.VMI.Spec.Domain.Devices.Inputs[0].Type).NotTo(BeEmpty())
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

		It("does not let an other-namespace claim suppress the VMI claim stub", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("nsclaim"),
				libvmi.WithPersistentVolumeClaim("disk0", "claim"),
			)

			result, err := render.PodFromVMI(vmi, render.Options{
				LauncherImage: launcherImage,
				PVCs: []*k8sv1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{Name: "claim", Namespace: "other"},
					Spec: k8sv1.PersistentVolumeClaimSpec{
						AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Volumes).To(ContainElement(HaveField("Name", "disk0")))
		})

		It("stubs a DataVolume-backed claim", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("dvvm"),
				libvmi.WithDataVolume("disk0", "dv-claim"),
			)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Volumes).To(ContainElement(HaveField("Name", "disk0")))
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
