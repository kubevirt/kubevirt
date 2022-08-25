package recorder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
)

var _ = Describe("Recorder", func() {
	var (
		m       MountRecorder
		vmi     *v1.VirtualMachineInstance
		tempDir string
	)

	Context("From cache", func() {
		var record *vmiMountTargetRecord

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			m = NewMountRecorder(tempDir)
			vmi = api.NewMinimalVMI("fake-vmi")
			vmi.UID = "1234"
			record = &vmiMountTargetRecord{
				MountTargetEntries: []MountTargetEntry{
					{
						TargetFile: "/test/target0",
						SocketFile: "/test/sock0",
					},
					{
						TargetFile: "/test/target1",
						SocketFile: "/test/sock1",
					},
				},
				UsesSafePaths: true,
			}
		})

		It("GetMountRecord from cache", func() {
			writeRecordFile(filepath.Join(tempDir, string(vmi.UID)), record.MountTargetEntries)
			res, err := m.GetMountRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(res, record.MountTargetEntries)).To(BeTrue())
		})

		DescribeTable("AddMountRecord", func(existingCachedValues bool) {
			recordFile := filepath.Join(tempDir, string(vmi.UID))
			expectedRecord := &vmiMountTargetRecord{UsesSafePaths: true}
			if existingCachedValues {
				writeRecordFile(recordFile, record.MountTargetEntries)
				expectedRecord = record
			}
			newRecord := []MountTargetEntry{
				{
					TargetFile: "/test/target2",
					SocketFile: "/test/sock2",
				},
			}
			err := m.AddMountRecord(vmi, newRecord)
			Expect(err).ToNot(HaveOccurred())
			if existingCachedValues {
				expectedRecord.MountTargetEntries = append(record.MountTargetEntries, newRecord...)
			} else {
				expectedRecord.MountTargetEntries = newRecord
			}
			res, err := readRecordFile(recordFile)
			Expect(err).ToNot(HaveOccurred())
			if !equality.Semantic.DeepEqual(expectedRecord, res) {
				ginkgo.Fail(fmt.Sprintf("expectedRecord %v not equal to %v", *expectedRecord, *res))
			}
		},
			Entry("no cached values", false),
			Entry("cached values", true),
		)

		DescribeTable("SetMountRecord", func(existingCachedValues bool) {
			recordFile := filepath.Join(tempDir, string(vmi.UID))
			expectedRecord := &vmiMountTargetRecord{UsesSafePaths: true}
			if existingCachedValues {
				writeRecordFile(recordFile, record.MountTargetEntries)
				expectedRecord = record
			}
			newRecord := []MountTargetEntry{
				{
					TargetFile: "/test/target2",
				},
			}
			err := m.SetMountRecord(vmi, newRecord)
			Expect(err).ToNot(HaveOccurred())
			expectedRecord.MountTargetEntries = newRecord
			res, err := readRecordFile(recordFile)
			Expect(err).ToNot(HaveOccurred())
			if !equality.Semantic.DeepEqual(expectedRecord, res) {
				ginkgo.Fail(fmt.Sprintf("expectedRecord %v not equal to %v", *expectedRecord, *res))
			}
		},
			Entry("no cached values", false),
			Entry("cached values", true),
		)

		It("SetMountRecord should fail if vmi.UID is empty", func() {
			vmi.UID = ""
			entry := []MountTargetEntry{
				{
					TargetFile: filepath.Join(tempDir, "test"),
				},
			}
			err := m.SetMountRecord(vmi, entry)
			Expect(err).To(HaveOccurred())
		})

		It("GetMountRecord should error if vmi UID is empty", func() {
			vmi.UID = ""
			_, err := m.GetMountRecord(vmi)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Clean-up", func() {
		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			m = NewMountRecorder(tempDir)
			vmi = api.NewMinimalVMI("fake-vmi")
			vmi.UID = "1234"
		})

		createTargetFiles := func(cd []MountTargetEntry) {
			for _, entry := range cd {
				file, err := os.Create(entry.TargetFile)
				Expect(err).ToNot(HaveOccurred())
				defer file.Close()
				file, err = os.Create(entry.SocketFile)
				Expect(err).ToNot(HaveOccurred())
				defer file.Close()
			}
		}

		areTargetFilesDeleted := func(cd []MountTargetEntry) bool {
			for _, entry := range cd {
				if _, err := os.Stat(entry.TargetFile); err == nil || !errors.Is(err, os.ErrNotExist) {
					return false
				}
				if _, err := os.Stat(entry.SocketFile); err == nil || !errors.Is(err, os.ErrNotExist) {
					return false
				}
			}
			return true
		}

		fileNotExist := func(file string) bool {
			_, err := os.Stat(file)
			return errors.Is(err, os.ErrNotExist)
		}

		It("Should delete target files and record entry", func() {
			cds := []MountTargetEntry{
				{
					TargetFile: filepath.Join(tempDir, "target0"),
					SocketFile: filepath.Join(tempDir, "socket0"),
				},
				{
					TargetFile: filepath.Join(tempDir, "target1"),
					SocketFile: filepath.Join(tempDir, "socket1"),
				},
			}
			createTargetFiles(cds)
			m.SetMountRecord(vmi, cds)

			err := m.DeleteMountRecord(vmi)
			Expect(err).ToNot(HaveOccurred())

			// Check clean-up
			res, err := m.GetMountRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(BeEmpty())
			Expect(areTargetFilesDeleted(cds)).To(BeTrue())
			// No futher record the record file should be deleted
			Expect(fileNotExist(filepath.Join(tempDir, string(vmi.UID)))).To(BeTrue())
		})
	})

	Context("For hotplug volumes", func() {
		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			m = NewMountRecorder(tempDir)
			vmi = api.NewMinimalVMI("fake-vmi")
			vmi.UID = "1234"
		})

		It("should be able to re-add entries after deleting them", func() {
			newRecord := []MountTargetEntry{
				{
					TargetFile: "/test/target1",
				},
			}
			expectedRecord := &vmiMountTargetRecord{UsesSafePaths: true,
				MountTargetEntries: newRecord,
			}

			// Add
			err := m.SetMountRecord(vmi, newRecord)
			Expect(err).ToNot(HaveOccurred())
			res, err := m.GetMountRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(res, expectedRecord.MountTargetEntries)).To(BeTrue())

			// Remove
			err = m.DeleteMountRecord(vmi)
			Expect(err).ToNot(HaveOccurred())

			// Re-add
			err = m.SetMountRecord(vmi, newRecord)
			Expect(err).ToNot(HaveOccurred())
			res, err = m.GetMountRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(res, expectedRecord.MountTargetEntries)).To(BeTrue())

		})
	})
})
