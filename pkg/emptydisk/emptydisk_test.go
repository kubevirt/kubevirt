package emptydisk

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("EmptyDisk", func() {

	var emptyDiskBaseDir string
	var creator *emptyDiskCreator

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
		emptyDiskBaseDir, err = ioutil.TempDir("", "emptydisk-dir")
		Expect(err).ToNot(HaveOccurred())
		creator = &emptyDiskCreator{
			emptyDiskBaseDir: emptyDiskBaseDir,
			discCreateFunc:   fakeCreatorFunc,
		}
	})
	AfterEach(func() {
		os.RemoveAll(emptyDiskBaseDir)
	})

	Describe("a vmi with emptyDisks attached", func() {
		It("should get a new qcow2 image if not already present", func() {
			vmi := api.NewMinimalVMI("testvmi")
			AppendEmptyDisk(vmi, "testdisk")
			err := creator.CreateTemporaryDisks(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(filePathForVolumeName(emptyDiskBaseDir, "testdisk"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(path.Join(emptyDiskBaseDir, "testdisk.qcow2"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not override ", func() {
			vmi := api.NewMinimalVMI("testvmi")
			AppendEmptyDisk(vmi, "testdisk")
			err := creator.CreateTemporaryDisks(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(filePathForVolumeName(emptyDiskBaseDir, "testdisk"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(path.Join(emptyDiskBaseDir, "testdisk.qcow2"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should generate non-conflicting volume paths per disk", func() {
			Expect(NewEmptyDiskCreator().FilePathForVolumeName("volume1")).ToNot(Equal(NewEmptyDiskCreator().FilePathForVolumeName("volume2")))
		})
		It("should leave pre-existing disks alone", func() {
			vmi := api.NewMinimalVMI("testvmi")
			AppendEmptyDisk(vmi, "testdisk")
			ioutil.WriteFile(filePathForVolumeName(emptyDiskBaseDir, "testdisk"), []byte("test"), 0777)
			err := creator.CreateTemporaryDisks(vmi)
			Expect(err).ToNot(HaveOccurred())
			data, err := ioutil.ReadFile(filePathForVolumeName(emptyDiskBaseDir, "testdisk"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("test"))
		})
	})

})

func fakeCreatorFunc(filePath string, _ string) error {
	fmt.Println(filePath)
	f, err := os.Create(filePath)
	if err == nil {
		f.Close()
	}
	return err
}
