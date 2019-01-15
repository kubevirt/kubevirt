package emptydisk

import (
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("EmptyDisk", func() {

	AppendEmptyDisk := func(vmi *v1.VirtualMachineInstance, diskName string) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: diskName,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{},
			},
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: diskName,
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

	Describe("a vmi with emptyDisks attached", func() {
		It("should get a new qcow2 image if not already present", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			AppendEmptyDisk(vmi, "testdisk")
			err := CreateTemporaryDisks(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(FilePathForVolumeName("testdisk"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(path.Join(EmptyDiskBaseDir, "testdisk.qcow2"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not override ", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			AppendEmptyDisk(vmi, "testdisk")
			err := CreateTemporaryDisks(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(FilePathForVolumeName("testdisk"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(path.Join(EmptyDiskBaseDir, "testdisk.qcow2"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should generate non-conflicting volume paths per disk", func() {
			Expect(FilePathForVolumeName("volume1")).ToNot(Equal(FilePathForVolumeName("volume2")))
		})
		It("should leave pre-existing disks alone", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			AppendEmptyDisk(vmi, "testdisk")
			ioutil.WriteFile(FilePathForVolumeName("testdisk"), []byte("test"), 0777)
			err := CreateTemporaryDisks(vmi)
			Expect(err).ToNot(HaveOccurred())
			data, err := ioutil.ReadFile(FilePathForVolumeName("testdisk"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("test"))
		})
	})

})
