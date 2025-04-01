package filewatcher_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-handler/filewatcher"
)

var _ = Describe("Filewatcher", func() {
	var (
		path         string
		testfilePath string
		watcher      *filewatcher.FileWatcher
	)

	createFile := func() {
		_, err := os.Create(testfilePath)
		Expect(err).ToNot(HaveOccurred())
	}

	removeFile := func() {
		Expect(os.Remove(testfilePath)).To(Succeed())
	}

	BeforeEach(func() {
		path = GinkgoT().TempDir()
		testfilePath = filepath.Join(path, "testfile")
		watcher = filewatcher.New(testfilePath, 10*time.Millisecond)
	})

	AfterEach(func() {
		watcher.Close()
		Eventually(watcher.Events).Should(BeClosed())
		Eventually(watcher.Errors).Should(BeClosed())
	})

	It("Should detect a file being created", func() {
		watcher.Run()
		createFile()
		Eventually(watcher.Events).Should(Receive(Equal(filewatcher.Create)))
		removeFile()
	})

	It("Should detect a file being removed", func() {
		createFile()
		watcher.Run()
		removeFile()
		Eventually(watcher.Events).Should(Receive(Equal(filewatcher.Remove)))
	})

	It("Should detect the ino of a file changing", func() {
		createFile()
		watcher.Run()
		removeFile()
		createFile()
		Eventually(watcher.Events).Should(Receive(Equal(filewatcher.InoChange)))
	})

	It("Should detect nothing if file exists", func() {
		createFile()
		watcher.Run()
		Consistently(watcher.Events).ShouldNot(Receive())
	})

	It("Should not detect if other files are created or removed", func() {
		watcher.Run()
		otherfile := filepath.Join(path, "otherfile")
		_, err := os.Create(otherfile)
		Expect(err).ToNot(HaveOccurred())
		Consistently(watcher.Events).ShouldNot(Receive())
		Expect(os.Remove(otherfile)).To(Succeed())
		Consistently(watcher.Events).ShouldNot(Receive())
	})
})
