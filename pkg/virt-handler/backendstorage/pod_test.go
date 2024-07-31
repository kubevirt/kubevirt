package backendstorage

import (
	"log"
	"os"
	"path"

	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backend storage block volume tests", func() {
	filesystemVolumeMode := k8sv1.PersistentVolumeFilesystem
	blockVolumeMode := k8sv1.PersistentVolumeBlock
	vmiWithBlockBackendStorage := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vm1",
		},
		Status: v1.VirtualMachineInstanceStatus{
			VolumeStatus: []v1.VolumeStatus{
				{
					Name: "vm-state",
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						VolumeMode: &blockVolumeMode,
					},
				},
			},
		},
	}
	vmiWithoutBlockBackendStorage := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vm1",
		},
		Status: v1.VirtualMachineInstanceStatus{
			VolumeStatus: []v1.VolumeStatus{
				{
					Name: "vm1-main-disk",
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						VolumeMode: &filesystemVolumeMode,
					},
				},
			},
		},
	}
	vmiWithFilesystemVolumeStatus := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vm1",
		},
		Status: v1.VirtualMachineInstanceStatus{
			VolumeStatus: []v1.VolumeStatus{
				{
					Name: "vm-state",
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						VolumeMode: &filesystemVolumeMode,
					},
				},
			},
		},
	}

	It("should properly detect if VM is using block backend storage", func() {
		u, err := usingBlockStorage(vmiWithBlockBackendStorage)
		Expect(err).ToNot(HaveOccurred())
		Expect(u).To(BeTrue())

		u, err = usingBlockStorage(vmiWithoutBlockBackendStorage)
		Expect(err).ToNot(HaveOccurred())
		Expect(u).To(BeFalse())

		u, err = usingBlockStorage(vmiWithFilesystemVolumeStatus)
		Expect(err).ToNot(HaveOccurred())
		Expect(u).To(BeFalse())
	})
})

var _ = Describe("Create directories for persistent devices", func() {
	BeforeEach(func() {
		var err error
		testTempDir, err = prepareFilesystemTestEnv("create-backend-directories-")
		log.Printf("Temporary directory created at %s", testTempDir)
		Expect(err).ToNot(HaveOccurred())
		configureNonRootOwnerAndSelinuxContext = func(vmi *v1.VirtualMachineInstance, f *safepath.Path) error {
			return nil
		}
	})
	AfterEach(func() {
		err := cleanupFilesystemTestEnv(testTempDir)
		Expect(err).ToNot(HaveOccurred())
	})
	It("should properly create directories for VMs with persistent TPM device", func() {
		log.Printf("Using temporary directory: %s", testTempDir)
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "vm1",
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						TPM: &v1.TPMDevice{Persistent: pointer.Bool(true)},
					},
				},
			},
		}
		vmStateDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, "/var/lib/libvirt/vm-state")
		Expect(err).ToNot(HaveOccurred())
		err = prepareVMStateDirectories(vmi, vmStateDir)
		Expect(err).ToNot(HaveOccurred())

		tpmSubDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, unsafepath.UnsafeRelative(vmStateDir.Raw()), "swtpm")
		Expect(err).ToNot(HaveOccurred())
		localcaSubDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, unsafepath.UnsafeRelative(vmStateDir.Raw()), "swtpm-localca")
		Expect(err).ToNot(HaveOccurred())
		tpmStateDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, "/var/lib/libvirt/swtpm")
		Expect(err).ToNot(HaveOccurred())
		localcaStateDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, "/var/lib/swtpm-localca")
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeRelative(tpmStateDir.Raw())).To(Equal(unsafepath.UnsafeRelative(tpmSubDir.Raw())))
		Expect(unsafepath.UnsafeAbsolute(localcaStateDir.Raw())).To(Equal(unsafepath.UnsafeAbsolute(localcaSubDir.Raw())))

		nvramDirFile, err := os.Stat(path.Join(testTempDir, "/var/lib/libvirt/qemu/nvram"))
		Expect(err).ToNot(HaveOccurred())
		Expect(nvramDirFile.IsDir()).To(BeTrue())
	})

	It("should successfully prepare the directories even if device subdirectory already exists", func() {
		log.Printf("Using temporary directory: %s", testTempDir)
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "vm1",
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Firmware: &v1.Firmware{
						Bootloader: &v1.Bootloader{EFI: &v1.EFI{Persistent: pointer.Bool(true)}},
					},
				},
			},
		}
		vmStateDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, "/var/lib/libvirt/vm-state")
		Expect(err).ToNot(HaveOccurred())

		// Create the "nvram" directory under "vm-state".
		err = safepath.MkdirAtNoFollow(vmStateDir, "nvram", os.ModePerm)
		Expect(err).ToNot(HaveOccurred())

		err = prepareVMStateDirectories(vmi, vmStateDir)
		Expect(err).ToNot(HaveOccurred())

		nvramSubDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, unsafepath.UnsafeRelative(vmStateDir.Raw()), "nvram")
		Expect(err).ToNot(HaveOccurred())
		nvramStateDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, "/var/lib/libvirt/qemu/nvram")
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeRelative(nvramStateDir.Raw())).To(Equal(unsafepath.UnsafeRelative(nvramSubDir.Raw())))
		Expect(unsafepath.UnsafeAbsolute(nvramStateDir.Raw())).To(Equal(unsafepath.UnsafeAbsolute(nvramSubDir.Raw())))
	})

	It("should not remove the target VM state directories if the VMI is reconciled multiple times", func() {
		log.Printf("Using temporary directory: %s", testTempDir)
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "vm1",
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Firmware: &v1.Firmware{
						Bootloader: &v1.Bootloader{EFI: &v1.EFI{Persistent: pointer.Bool(true)}},
					},
				},
			},
		}
		vmStateDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, "/var/lib/libvirt/vm-state")
		Expect(err).ToNot(HaveOccurred())

		var testFilePath string
		// Reconcile the VM state directories.
		err = prepareVMStateDirectories(vmi, vmStateDir)
		Expect(err).ToNot(HaveOccurred())
		// Create a file in NVRAM subdirectory.
		nvramSubDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, unsafepath.UnsafeRelative(vmStateDir.Raw()), "nvram")
		Expect(err).ToNot(HaveOccurred())
		testFilePath = path.Join(unsafepath.UnsafeAbsolute(nvramSubDir.Raw()), "default_vm1")
		_, err = os.Create(testFilePath)
		Expect(err).ToNot(HaveOccurred())
		// Reconcile the VM state directories again.
		err = prepareVMStateDirectories(vmi, vmStateDir)
		Expect(err).ToNot(HaveOccurred())
		// Reconcile the VM state directories for the third time.
		err = prepareVMStateDirectories(vmi, vmStateDir)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat(testFilePath)
		Expect(err).ToNot(HaveOccurred())
	})
})
