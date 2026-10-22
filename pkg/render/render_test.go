package render_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	netbinding "kubevirt.io/kubevirt/pkg/network/netbinding"
	netannotations "kubevirt.io/kubevirt/pkg/network/pod/annotations"
	netresources "kubevirt.io/kubevirt/pkg/network/resources"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/render"
	storageannotations "kubevirt.io/kubevirt/pkg/storage/pod/annotations"
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

		It("does not attach an input to a standalone VMI", func() {
			vmi := libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("inputvm"))
			vmi.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI.Spec.Domain.Devices.Inputs).To(BeEmpty())
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

		It("renders a binding-plugin sidecar from Options.NetworkBindings", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("plugvm"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding())),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.NewInterface("net1", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "custom"}))),
				libvmi.WithNetwork(&v1.Network{
					Name: "net1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "other-net"},
					},
				}),
			)

			result, err := render.PodFromVMI(vmi, render.Options{
				LauncherImage: launcherImage,
				NetworkBindings: map[string]v1.InterfaceBindingPlugin{
					"custom": {SidecarImage: "kubevirt/binding-sidecar:test"},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Containers).To(ContainElement(HaveField("Image", "kubevirt/binding-sidecar:test")))
		})

		It("rejects a binding plugin that was not registered", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("plugvm"),
				libvmi.WithInterface(libvmi.NewInterface("net1", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "custom"}))),
				libvmi.WithNetwork(&v1.Network{
					Name: "net1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "other-net"},
					},
				}),
			)

			_, err := render.PodFromVMI(vmi, opts())
			Expect(err).To(MatchError(ContainSubstring("couldn't find configuration for network binding: custom")))
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

		It("applies RunAsUser to the VMI and pod security context", func() {
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
			Expect(result.VMI.Status.RuntimeUser).To(Equal(uint64(1000)))
			Expect(result.Pod.Spec.SecurityContext).NotTo(BeNil())
			Expect(result.Pod.Spec.SecurityContext.RunAsUser).To(HaveValue(Equal(int64(1000))))
		})
	})

	Describe("PodFromVM", func() {
		It("returns an error when the VM is nil", func() {
			_, err := render.PodFromVM(nil, opts())
			Expect(err).To(MatchError(ContainSubstring("VM is nil")))
		})

		It("returns an error when the VM has no template", func() {
			vm := &v1.VirtualMachine{ObjectMeta: metav1.ObjectMeta{Name: "empty"}}

			_, err := render.PodFromVM(vm, opts())
			Expect(err).To(MatchError(ContainSubstring("no template spec")))
		})

		It("assigns a random firmware UUID and serial when the template has none", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("fromvm"),
			))

			first, err := render.VMIFromVM(vm, opts())
			Expect(err).NotTo(HaveOccurred())
			second, err := render.VMIFromVM(vm, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(first.Spec.Domain.Firmware.UUID).NotTo(BeEmpty())
			Expect(first.Spec.Domain.Firmware.Serial).NotTo(BeEmpty())
			Expect(second.Spec.Domain.Firmware.UUID).NotTo(Equal(first.Spec.Domain.Firmware.UUID))
		})

		It("keeps a firmware UUID already set on the template", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("fromvm"),
				libvmi.WithFirmwareUUID("keep-me"),
			))

			vmi, err := render.VMIFromVM(vm, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(types.UID("keep-me")))
		})

		It("defaults the auto-attached input type for a VM", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("inputvm"),
			))
			vm.Spec.Template.Spec.Domain.Devices.AutoattachInputDevice = pointer.P(true)

			result, err := render.PodFromVM(vm, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI.Spec.Domain.Devices.Inputs).To(HaveLen(1))
			Expect(result.VMI.Spec.Domain.Devices.Inputs[0].Type).NotTo(BeEmpty())
		})

		It("uses Options.Architecture for the machine type", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("archvm"),
			))

			vmi, err := render.VMIFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Architecture:  "s390x",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Spec.Architecture).To(Equal("s390x"))
			Expect(vmi.Spec.Domain.Machine.Type).To(Equal(virtconfig.DefaultS390XMachineType))
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

		It("rejects an instancetype inferred from a volume", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("itvm")),
				libvmi.WithInstancetypeInferredFromVolume("disk0"),
			)

			_, err := render.PodFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Instancetype:  &instancetypev1beta1.VirtualMachineInstancetypeSpec{},
			})
			Expect(err).To(MatchError(ContainSubstring("infer instancetype from a volume")))
		})

		It("rejects a preference inferred from a volume", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("prefvm")),
				libvmi.WithPreferenceInferredFromVolume("disk0"),
			)

			_, err := render.PodFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Preference:    &instancetypev1beta1.VirtualMachinePreferenceSpec{},
			})
			Expect(err).To(MatchError(ContainSubstring("infer preference from a volume")))
		})

		It("rejects a VM with an instancetype matcher", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("itvm")),
				libvmi.WithInstancetype("u1.small"),
			)

			_, err := render.PodFromVM(vm, opts())
			Expect(err).To(MatchError(ContainSubstring("instancetype matchers")))
		})

		It("drops a completed memory-dump volume before pairing disks", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("dumpvm"),
			))
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "dump-pvc",
				VolumeSource: v1.VolumeSource{
					MemoryDump: &v1.MemoryDumpVolumeSource{},
				},
			})
			vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "dump-pvc",
				Phase:     v1.MemoryDumpCompleted,
			}

			vmi, err := render.VMIFromVM(vm, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Spec.Volumes).NotTo(ContainElement(HaveField("Name", "dump-pvc")))
		})

		It("keeps an in-progress memory-dump volume", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("dumpvm"),
			))
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
				Name: "dump-pvc",
				VolumeSource: v1.VolumeSource{
					MemoryDump: &v1.MemoryDumpVolumeSource{},
				},
			})
			vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "dump-pvc",
				Phase:     v1.MemoryDumpInProgress,
			}

			vmi, err := render.VMIFromVM(vm, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Spec.Volumes).To(ContainElement(HaveField("Name", "dump-pvc")))
		})

		It("rejects an instancetype spec when the VM has no matcher", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("itvm"),
			))

			_, err := render.PodFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Instancetype:  &instancetypev1beta1.VirtualMachineInstancetypeSpec{},
			})
			Expect(err).To(MatchError(ContainSubstring("no instancetype matcher")))
		})

		It("rejects a preference spec when the VM has no matcher", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("prefvm"),
			))

			_, err := render.PodFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Preference:    &instancetypev1beta1.VirtualMachinePreferenceSpec{},
			})
			Expect(err).To(MatchError(ContainSubstring("no preference matcher")))
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

		It("applies a preferred interface model to the default pod interface", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("prefvm")),
				libvmi.WithPreference("fedora"),
			)

			result, err := render.PodFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Preference: &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Devices: &instancetypev1beta1.DevicePreferences{
						PreferredInterfaceModel: "virtio",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.VMI.Spec.Domain.Devices.Interfaces).NotTo(BeEmpty())
			Expect(result.VMI.Spec.Domain.Devices.Interfaces[0].Model).To(Equal("virtio"))
		})

		It("applies architecture and machine preferences before cluster defaults", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("archvm")),
				libvmi.WithPreference("arm"),
			)

			vmi, err := render.VMIFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Architecture:  "amd64",
				Preference: &instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferredArchitecture: pointer.P("arm64"),
					Machine: &instancetypev1beta1.MachinePreferences{
						PreferredMachineType: "virt",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Spec.Architecture).To(Equal("arm64"))
			Expect(vmi.Spec.Domain.Machine.Type).To(Equal("virt"))
		})

		It("keeps a preferred disk bus when the volume has no disk entry", func() {
			vm := libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("diskvm")),
				libvmi.WithPreference("linux"),
			)
			vm.Spec.Template.Spec.Volumes = []v1.Volume{{
				Name: "disk0",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "some/image"},
				},
			}}
			vm.Spec.Template.Spec.Domain.Devices.Disks = nil

			vmi, err := render.VMIFromVM(vm, render.Options{
				LauncherImage: launcherImage,
				Architecture:  "amd64",
				Preference: &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Devices: &instancetypev1beta1.DevicePreferences{
						PreferredDiskBus: v1.DiskBusVirtio,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Spec.Domain.Devices.Disks).To(ContainElement(And(
				HaveField("Name", "disk0"),
				HaveField("Disk.Bus", v1.DiskBusVirtio),
			)))
		})
	})

	Describe("backend storage", func() {
		It("stubs persistent-state-for-<vm> when persistent TPM has no claim", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("tpmvm"),
				libvmi.WithTPM(true),
			)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Volumes).To(ContainElement(And(
				HaveField("Name", "vm-state"),
				HaveField("PersistentVolumeClaim.ClaimName", "persistent-state-for-tpmvm"),
			)))
		})

		It("uses a caller-supplied backend-storage claim", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("tpmvm"),
				libvmi.WithTPM(true),
			)

			result, err := render.PodFromVMI(vmi, render.Options{
				LauncherImage: launcherImage,
				PVCs: []*k8sv1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "state-claim",
						Namespace: "default",
						Labels:    map[string]string{"persistent-state-for": "tpmvm"},
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Volumes).To(ContainElement(And(
				HaveField("Name", "vm-state"),
				HaveField("PersistentVolumeClaim.ClaimName", "state-claim"),
			)))
		})

		It("stubs backend storage for PodFromVM", func() {
			vm := libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("tpmvm"),
				libvmi.WithTPM(true),
			))

			result, err := render.PodFromVM(vm, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Volumes).To(ContainElement(And(
				HaveField("Name", "vm-state"),
				HaveField("PersistentVolumeClaim.ClaimName", "persistent-state-for-tpmvm"),
			)))
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

		It("stubs a missing DataVolume claim as a filesystem mount", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("dvvm"),
				libvmi.WithDataVolume("disk0", "dv-claim"),
			)

			result, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Pod.Spec.Volumes).To(ContainElement(HaveField("Name", "disk0")))
			Expect(computeContainer(result.Pod).VolumeMounts).To(ContainElement(HaveField("Name", "disk0")))
			Expect(computeContainer(result.Pod).VolumeDevices).NotTo(ContainElement(HaveField("Name", "disk0")))
		})

		It("uses a caller-supplied Block PVC for a DataVolume", func() {
			block := k8sv1.PersistentVolumeBlock
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("dvvm"),
				libvmi.WithDataVolume("disk0", "dv-claim"),
			)

			result, err := render.PodFromVMI(vmi, render.Options{
				LauncherImage: launcherImage,
				PVCs: []*k8sv1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{Name: "dv-claim", Namespace: "default"},
					Spec: k8sv1.PersistentVolumeClaimSpec{
						AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
						VolumeMode:  &block,
					},
				}},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(computeContainer(result.Pod).VolumeDevices).To(ContainElement(HaveField("Name", "disk0")))
		})
	})

	Describe("offline vs online equivalence", func() {
		It("matches prepared VMI defaults and Pod output after normalizing QEMU timeout jitter", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace("default"),
				libvmi.WithName("equiv"),
				libvmi.WithContainerDisk("disk0", "some/image"),
			)

			offline, err := render.PodFromVMI(vmi, opts())
			Expect(err).NotTo(HaveOccurred())
			Expect(offline.VMI.Spec.EvictionStrategy).To(HaveValue(Equal(v1.EvictionStrategyNone)))

			config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
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
				"",
				cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
				cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
				services.WithSidecarCreator(func(vmi *v1.VirtualMachineInstance, _ *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
					return hooks.UnmarshalHookSidecarList(vmi)
				}),
				services.WithSidecarCreator(netbinding.NetBindingPluginSidecarList),
				services.WithMemoryOverheadCalculators(netresources.NewMemoryCalculator(config)),
				services.WithAnnotationsGenerators(netannotations.NewGenerator(config), storageannotations.Generator{}),
			)

			onlineVMI := vmi.DeepCopy()
			Expect(mutators.ApplyNewVMIMutations(onlineVMI, config)).To(Succeed())
			onlinePod, err := svc.RenderLaunchManifest(onlineVMI)
			Expect(err).NotTo(HaveOccurred())

			Expect(offline.VMI.Spec.EvictionStrategy).To(Equal(onlineVMI.Spec.EvictionStrategy))
			Expect(offline.Pod.Spec.Containers).To(HaveLen(len(onlinePod.Spec.Containers)))
			for i := range offline.Pod.Spec.Containers {
				Expect(offline.Pod.Spec.Containers[i].Name).To(Equal(onlinePod.Spec.Containers[i].Name))
				Expect(offline.Pod.Spec.Containers[i].Image).To(Equal(onlinePod.Spec.Containers[i].Image))
				Expect(offline.Pod.Spec.Containers[i].Resources).To(Equal(onlinePod.Spec.Containers[i].Resources))
				Expect(withoutQemuTimeout(offline.Pod.Spec.Containers[i].Args)).To(Equal(withoutQemuTimeout(onlinePod.Spec.Containers[i].Args)))
			}
			Expect(volumeNames(offline.Pod)).To(ConsistOf(volumeNames(onlinePod)))
		})
	})
})

func withoutQemuTimeout(args []string) []string {
	out := make([]string, 0, len(args))
	skip := false
	for _, arg := range args {
		if skip {
			skip = false
			continue
		}
		if arg == "--qemu-timeout" {
			skip = true
			continue
		}
		out = append(out, arg)
	}
	return out
}

func computeContainer(pod *k8sv1.Pod) k8sv1.Container {
	for _, c := range pod.Spec.Containers {
		if c.Name == "compute" {
			return c
		}
	}
	return k8sv1.Container{}
}

func volumeNames(pod *k8sv1.Pod) []string {
	names := make([]string, 0, len(pod.Spec.Volumes))
	for _, vol := range pod.Spec.Volumes {
		names = append(names, vol.Name)
	}
	return names
}
