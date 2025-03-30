package emptydisk

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("EmptyDisk", func() {
	var emptyDiskBaseDir string
	var creator *emptyDiskCreator

	BeforeEach(func() {
		var err error
		emptyDiskBaseDir, err = os.MkdirTemp("", "emptydisk-dir")
		Expect(err).ToNot(HaveOccurred())
		creator = &emptyDiskCreator{
			emptyDiskBaseDir: emptyDiskBaseDir,
			discCreateFunc:   fakeCreatorFunc,
		}
	})
	AfterEach(func() {
		Expect(os.RemoveAll(emptyDiskBaseDir)).To(Succeed())
	})

	Describe("a vmi with emptyDisks attached", func() {
		It("should get a new qcow2 image if not already present", func() {
			vmi := libvmi.New(
				libvmi.WithEmptyDisk("testdisk", "", resource.MustParse("3Gi")),
			)

			err := creator.CreateTemporaryDisks(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(filePathForVolumeName(emptyDiskBaseDir, "testdisk"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(path.Join(emptyDiskBaseDir, "testdisk.qcow2"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not override ", func() {
			vmi := libvmi.New(
				libvmi.WithEmptyDisk("testdisk", "", resource.MustParse("3Gi")),
			)

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
			vmi := libvmi.New(
				libvmi.WithEmptyDisk("testdisk", "", resource.MustParse("3Gi")),
			)

			err := os.WriteFile(filePathForVolumeName(emptyDiskBaseDir, "testdisk"), []byte("test"), 0o777)
			Expect(err).ToNot(HaveOccurred())
			err = creator.CreateTemporaryDisks(vmi)
			Expect(err).ToNot(HaveOccurred())
			data, err := os.ReadFile(filePathForVolumeName(emptyDiskBaseDir, "testdisk"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("test"))
		})
	})
})

func fakeCreatorFunc(filePath string, _ string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	return f.Close()
}
