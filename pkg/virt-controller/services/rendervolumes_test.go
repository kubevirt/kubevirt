package services

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Container spec renderer", func() {
	var vsr *VolumeRenderer

	const (
		containerDisk = "cdisk1"
		ephemeralDisk = "disk1"
		namespace     = "ns1"
		virtShareDir  = "dir1"
	)

	Context("without any options", func() {
		BeforeEach(func() {
			var err error
			vsr, err = NewVolumeRenderer(namespace, ephemeralDisk, containerDisk, virtShareDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("does *not* have any volume devices", func() {
			Expect(vsr.VolumeDevices()).To(BeEmpty())
		})

		It("to feature the private, public, ephemeral-disks, libvirt-runtime, and sockets mount points", func() {
			Expect(vsr.Mounts()).To(ConsistOf(defaultVolumeMounts()))
		})

		It("to feature the private, public, sockets, virt-bin-share-dir, libvirt-runtime, ephemeral, and container disk volumes", func() {
			Expect(vsr.Volumes()).To(ConsistOf(defaultVolumes()))
		})
	})

	Context("with ephemeral volume option", func() {
		const ephemeralVolumeName = "evn"
		BeforeEach(func() {
			ephemeralVolumeOption := v1.Volume{
				Name: ephemeralVolumeName,
				VolumeSource: v1.VolumeSource{
					Ephemeral: &v1.EphemeralVolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
					},
				},
			}

			pvcStore := &cache.FakeCustomStore{
				GetByKeyFunc: func(key string) (item interface{}, exists bool, err error) {
					return &k8sv1.PersistentVolumeClaim{
						Spec: k8sv1.PersistentVolumeClaimSpec{},
					}, true, nil
				},
			}

			var err error
			vsr, err = NewVolumeRenderer(namespace, ephemeralDisk, containerDisk, virtShareDir, withVMIVolumes(pvcStore, []v1.Volume{ephemeralVolumeOption}, nil))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should feature the default mount points plus the ephemeral disk volume mount", func() {
			Expect(vsr.Mounts()).To(ConsistOf(
				append(
					defaultVolumeMounts(),
					k8sv1.VolumeMount{
						Name:      ephemeralVolumeName,
						MountPath: vmiDiskPath(ephemeralVolumeName)})))
		})

		It("should feature the default volumes plus the ephemeral disk volume", func() {
			Expect(vsr.Volumes()).To(ConsistOf(
				append(
					defaultVolumes(),
					k8sv1.Volume{
						Name: ephemeralVolumeName,
						VolumeSource: k8sv1.VolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
						},
					})))
		})

		It("does *not* have any volume devices", func() {
			Expect(vsr.VolumeDevices()).To(BeEmpty())
		})
	})

	Context("with host disk volume option", func() {
		const (
			hostDiskName = "tiny-winy-disk"
			hostDiskPath = "/little-bit/to/the/left"
		)

		var (
			expectedHostDiskType = v1.HostDiskExistsOrCreate
			expectedHostPathType = k8sv1.HostPathDirectoryOrCreate
		)

		BeforeEach(func() {
			hostDisk := v1.Volume{
				Name: hostDiskName,
				VolumeSource: v1.VolumeSource{
					HostDisk: &v1.HostDisk{
						Path: hostDiskPath,
						Type: expectedHostDiskType,
					},
				},
			}

			pvcStore := &cache.FakeCustomStore{
				GetByKeyFunc: func(key string) (item interface{}, exists bool, err error) {
					return &hostDisk, true, nil
				},
			}

			var err error
			vsr, err = NewVolumeRenderer(namespace, ephemeralDisk, containerDisk, virtShareDir, withVMIVolumes(pvcStore, []v1.Volume{hostDisk}, nil))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should feature the default mount points plus the host disk volume mount", func() {
			Expect(vsr.Mounts()).To(ConsistOf(
				append(
					defaultVolumeMounts(),
					k8sv1.VolumeMount{
						Name:      hostDiskName,
						MountPath: vmiDiskPath(hostDiskName)})))
		})

		It("should feature the default volumes plus the host disk volume", func() {
			Expect(vsr.Volumes()).To(ConsistOf(
				append(
					defaultVolumes(),
					k8sv1.Volume{
						Name: hostDiskName,
						VolumeSource: k8sv1.VolumeSource{
							HostPath: &k8sv1.HostPathVolumeSource{
								Type: &expectedHostPathType,
								Path: hostDiskPath[:strings.LastIndex(hostDiskPath, "/")],
							}},
					})))
		})

		It("does *not* have any volume devices", func() {
			Expect(vsr.VolumeDevices()).To(BeEmpty())
		})
	})

	Context("with CloudInitConfigDrive option", func() {
		const (
			cloudInitDriveName = "pepitos-drive"
			userData           = "break-dancing-flamingo"
			networkData        = "hoooonk.hooooonk"
		)

		BeforeEach(func() {
			cloudInitConfig := v1.Volume{
				Name: cloudInitDriveName,
				VolumeSource: v1.VolumeSource{
					CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{
						UserDataSecretRef:    &k8sv1.LocalObjectReference{Name: userData},
						NetworkDataSecretRef: &k8sv1.LocalObjectReference{Name: networkData},
					},
				},
			}

			pvcStore := &cache.FakeCustomStore{
				GetByKeyFunc: func(key string) (item interface{}, exists bool, err error) {
					return &cloudInitConfig, true, nil
				},
			}

			var err error
			vsr, err = NewVolumeRenderer(namespace, ephemeralDisk, containerDisk, virtShareDir, withVMIVolumes(pvcStore, []v1.Volume{cloudInitConfig}, nil))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should feature the default mount points plus the cloud init config drive volume mount", func() {
			Expect(vsr.Mounts()).To(ConsistOf(
				append(
					defaultVolumeMounts(),
					k8sv1.VolumeMount{
						Name:      "pepitos-drive-udata",
						ReadOnly:  true,
						MountPath: "/var/run/kubevirt-private/secret/pepitos-drive/userdata",
						SubPath:   "userdata",
					}, k8sv1.VolumeMount{
						Name:      "pepitos-drive-udata",
						ReadOnly:  true,
						MountPath: "/var/run/kubevirt-private/secret/pepitos-drive/userData",
						SubPath:   "userData",
					}, k8sv1.VolumeMount{
						Name:      "pepitos-drive-ndata",
						ReadOnly:  true,
						MountPath: "/var/run/kubevirt-private/secret/pepitos-drive/networkdata",
						SubPath:   "networkdata",
					}, k8sv1.VolumeMount{
						Name:      "pepitos-drive-ndata",
						ReadOnly:  true,
						MountPath: "/var/run/kubevirt-private/secret/pepitos-drive/networkData",
						SubPath:   "networkData",
					})))
		})

		It("should feature the default volumes plus the  cloud init config drive disk volume", func() {
			Expect(vsr.Volumes()).To(ConsistOf(
				append(
					defaultVolumes(),
					k8sv1.Volume{
						Name: "pepitos-drive-udata",
						VolumeSource: k8sv1.VolumeSource{
							Secret: &k8sv1.SecretVolumeSource{
								SecretName: "break-dancing-flamingo",
							},
						}}, k8sv1.Volume{
						Name: "pepitos-drive-ndata",
						VolumeSource: k8sv1.VolumeSource{
							Secret: &k8sv1.SecretVolumeSource{
								SecretName: "hoooonk.hooooonk",
							},
						}})))
		})

		It("does *not* have any volume devices", func() {
			Expect(vsr.VolumeDevices()).To(BeEmpty())
		})
	})

	Context("with DataVolume option", func() {
		const (
			dataVolumeName = "dv1"
		)

		BeforeEach(func() {
			dataVolume := v1.Volume{
				Name: dataVolumeName,
				VolumeSource: v1.VolumeSource{DataVolume: &v1.DataVolumeSource{
					Name: dataVolumeName,
				}},
			}

			pvcStore := &cache.FakeCustomStore{
				GetByKeyFunc: func(key string) (item interface{}, exists bool, err error) {
					return &k8sv1.PersistentVolumeClaim{}, true, nil
				},
			}

			var err error
			vsr, err = NewVolumeRenderer(namespace, ephemeralDisk, containerDisk, virtShareDir, withVMIVolumes(pvcStore, []v1.Volume{dataVolume}, nil))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should feature the default mount points plus the downward API volume mount", func() {
			Expect(vsr.Mounts()).To(ConsistOf(
				append(
					defaultVolumeMounts(),
					k8sv1.VolumeMount{
						Name:      "dv1",
						MountPath: "/var/run/kubevirt-private/vmi-disks/dv1",
					})))
		})

		It("should feature the default volumes plus the downward API volume", func() {
			Expect(vsr.Volumes()).To(ConsistOf(
				append(
					defaultVolumes(),
					k8sv1.Volume{
						Name: dataVolumeName,
						VolumeSource: k8sv1.VolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: dataVolumeName,
							},
						},
					})))
		})

		It("does *not* have any volume devices", func() {
			Expect(vsr.VolumeDevices()).To(BeEmpty())
		})
	})

	Context("with Downward API option", func() {
		const (
			downwardAPIVolumeName = "downward-then-upward"
		)

		BeforeEach(func() {
			downwardAPIVolume := v1.Volume{
				Name: downwardAPIVolumeName,
				VolumeSource: v1.VolumeSource{
					DownwardAPI: &v1.DownwardAPIVolumeSource{},
				}}

			disk := v1.Disk{Name: downwardAPIVolumeName}

			var err error
			vsr, err = NewVolumeRenderer(namespace, ephemeralDisk, containerDisk, virtShareDir, withVMIConfigVolumes([]v1.Disk{disk}, []v1.Volume{downwardAPIVolume}))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should feature the default mount points plus the downward API volume mount", func() {
			Expect(vsr.Mounts()).To(ConsistOf(
				append(
					defaultVolumeMounts(),
					k8sv1.VolumeMount{
						Name:      downwardAPIVolumeName,
						ReadOnly:  true,
						MountPath: "/var/run/kubevirt-private/downwardapi/downward-then-upward",
					})))
		})

		It("should feature the default volumes plus the downward API volume", func() {
			Expect(vsr.Volumes()).To(ConsistOf(
				append(
					defaultVolumes(),
					k8sv1.Volume{
						Name: downwardAPIVolumeName,
						VolumeSource: k8sv1.VolumeSource{
							DownwardAPI: &k8sv1.DownwardAPIVolumeSource{},
						},
					})))
		})

		It("does *not* have any volume devices", func() {
			Expect(vsr.VolumeDevices()).To(BeEmpty())
		})
	})
})

var _ = Describe("Backend storage render", func() {
	const (
		containerDisk = "cdisk1"
		ephemeralDisk = "disk1"
		namespace     = "ns1"
		virtShareDir  = "dir1"

		vmiName           = "vm1"
		vmStateVolumeName = "vm-state"
	)

	var (
		filesystemVolumeMode = k8sv1.PersistentVolumeFilesystem
		blockVolumeMode      = k8sv1.PersistentVolumeBlock

		baseVMI = v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: vmiName,
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{},
				},
			},
		}
		vmStateVolumes = []k8sv1.Volume{
			{
				Name: vmStateVolumeName,
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: "persistent-state-for-" + vmiName,
					},
				},
			},
		}
		vmStateVolumeDevices = []k8sv1.VolumeDevice{
			{
				Name:       vmStateVolumeName,
				DevicePath: "/dev/vm-state",
			},
		}
		tpmStateVolumeMounts = []k8sv1.VolumeMount{
			{Name: vmStateVolumeName, MountPath: "/var/lib/libvirt/swtpm", SubPath: "swtpm"},
			{Name: vmStateVolumeName, MountPath: "/var/lib/swtpm-localca", SubPath: "swtpm-localca"},
		}
		nvramStateVolumeMounts = []k8sv1.VolumeMount{
			{Name: vmStateVolumeName, MountPath: "/var/lib/libvirt/qemu/nvram", SubPath: "nvram"},
		}
	)

	DescribeTable("Should generate correct volumes",
		func(persistTPM, persistEFI *bool, volumeMode *k8sv1.PersistentVolumeMode, expectedVolumes []k8sv1.Volume, expectedVolumeMounts []k8sv1.VolumeMount, expectedVolumeDevices []k8sv1.VolumeDevice) {
			vmi := baseVMI.DeepCopy()
			if persistTPM != nil && *persistTPM {
				vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{Persistent: pointer.Bool(true)}
			}
			if persistEFI != nil && *persistEFI {
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						EFI: &v1.EFI{Persistent: pointer.Bool(true)},
					},
				}
			}

			var err error
			r, err := NewVolumeRenderer(namespace, ephemeralDisk, containerDisk, virtShareDir, withBackendStorage(vmi, volumeMode))
			Expect(err).NotTo(HaveOccurred())
			Expect(r.Volumes()).To(ConsistOf(append(defaultVolumes(), expectedVolumes...)))
			Expect(r.Mounts()).To(ConsistOf(append(defaultVolumeMounts(), expectedVolumeMounts...)))
			Expect(r.VolumeDevices()).To(ConsistOf(expectedVolumeDevices))
		},

		Entry("No TPM or EFI devices", nil, nil, &filesystemVolumeMode, []k8sv1.Volume{}, []k8sv1.VolumeMount{}, []k8sv1.VolumeDevice{}),
		Entry("Non-persistent TPM and EFI", pointer.Bool(false), pointer.Bool(false), &blockVolumeMode, []k8sv1.Volume{}, []k8sv1.VolumeMount{}, []k8sv1.VolumeDevice{}),
		Entry("Persistent TPM only, filesystem storage", pointer.Bool(true), pointer.Bool(false), &filesystemVolumeMode, vmStateVolumes, tpmStateVolumeMounts, []k8sv1.VolumeDevice{}),
		Entry("Persistent EFI only, default volume mode", pointer.Bool(false), pointer.Bool(true), nil, vmStateVolumes, nvramStateVolumeMounts, []k8sv1.VolumeDevice{}),
		Entry("Persistent EFI only, block storage", pointer.Bool(false), pointer.Bool(true), &blockVolumeMode, vmStateVolumes, []k8sv1.VolumeMount{}, vmStateVolumeDevices),
		Entry("Persistent TPM and EFI, block storage", pointer.Bool(true), pointer.Bool(true), &blockVolumeMode, vmStateVolumes, []k8sv1.VolumeMount{}, vmStateVolumeDevices),
	)
})

func vmiDiskPath(volumeName string) string {
	return fmt.Sprintf("/var/run/kubevirt-private/vmi-disks/%s", volumeName)
}

func defaultVolumes() []k8sv1.Volume {
	return []k8sv1.Volume{
		{
			Name:         "private",
			VolumeSource: k8sv1.VolumeSource{EmptyDir: &k8sv1.EmptyDirVolumeSource{}},
		}, {
			Name:         "public",
			VolumeSource: k8sv1.VolumeSource{EmptyDir: &k8sv1.EmptyDirVolumeSource{}},
		}, {
			Name:         "sockets",
			VolumeSource: k8sv1.VolumeSource{EmptyDir: &k8sv1.EmptyDirVolumeSource{}},
		}, {
			Name:         "virt-bin-share-dir",
			VolumeSource: k8sv1.VolumeSource{EmptyDir: &k8sv1.EmptyDirVolumeSource{}},
		}, {
			Name:         "libvirt-runtime",
			VolumeSource: k8sv1.VolumeSource{EmptyDir: &k8sv1.EmptyDirVolumeSource{}},
		}, {
			Name:         "ephemeral-disks",
			VolumeSource: k8sv1.VolumeSource{EmptyDir: &k8sv1.EmptyDirVolumeSource{}},
		}, {
			Name:         "container-disks",
			VolumeSource: k8sv1.VolumeSource{EmptyDir: &k8sv1.EmptyDirVolumeSource{}},
		},
	}
}

func defaultVolumeMounts() []k8sv1.VolumeMount {
	hostToContainerPropagation := k8sv1.MountPropagationHostToContainer

	return []k8sv1.VolumeMount{
		{Name: "private", MountPath: "/var/run/kubevirt-private"},
		{Name: "public", MountPath: "/var/run/kubevirt"},
		{Name: "ephemeral-disks", MountPath: "disk1"},
		{Name: "container-disks", MountPath: "cdisk1", MountPropagation: &hostToContainerPropagation},
		{Name: "libvirt-runtime", MountPath: "/var/run/libvirt"},
		{Name: "sockets", MountPath: "dir1/sockets"},
	}
}
