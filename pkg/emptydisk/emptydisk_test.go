package emptydisk

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"path"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("EmptyDisk", func() {

	AppendEmptyDisk := func(vm *v1.VirtualMachine, diskName string, volumeName string) {
		vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
			Name:       diskName,
			VolumeName: volumeName,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{},
			},
		})
		vm.Spec.Volumes = append(vm.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDisk: &v1.EmptyDiskSource{
					Capacity: resource.MustParse("3Gi"),
				},
			},
		})
	}

	BeforeEach(func() {
		var err error
		EmptyDiskBaseDir, err = ioutil.TempDir("", "emptydisk-dir")
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		os.RemoveAll(EmptyDiskBaseDir)
	})

	Describe("a vm with emptyDisks attached", func() {
		It("should get a new qcow2 image if not already present", func() {
			vm := v1.NewMinimalVM("testvm")
			AppendEmptyDisk(vm, "testdisk", "testvolume")
			err := CreateTemporaryDisks(vm)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(FilePathForVolumeName("testvolume"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(path.Join(EmptyDiskBaseDir, "testvolume.qcow2"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not override ", func() {
			vm := v1.NewMinimalVM("testvm")
			AppendEmptyDisk(vm, "testdisk", "testvolume")
			err := CreateTemporaryDisks(vm)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(FilePathForVolumeName("testvolume"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(path.Join(EmptyDiskBaseDir, "testvolume.qcow2"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should generate non-conflicting volume paths per disk", func() {
			Expect(FilePathForVolumeName("volume1")).ToNot(Equal(FilePathForVolumeName("volume2")))
		})
		It("should leave pre-existing disks alone", func() {
			vm := v1.NewMinimalVM("testvm")
			AppendEmptyDisk(vm, "testdisk", "testvolume")
			ioutil.WriteFile(FilePathForVolumeName("testvolume"), []byte("test"), 0777)
			err := CreateTemporaryDisks(vm)
			Expect(err).ToNot(HaveOccurred())
			data, err := ioutil.ReadFile(FilePathForVolumeName("testvolume"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("test"))
		})
	})

})
