package inotifyinformer

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

var _ = Describe("Inotify", func() {

	Context("When watching files in a directory", func() {

		var tmpDir string
		var informer cache.SharedIndexInformer
		var stopInformer chan struct{}

		BeforeEach(func() {
			var err error
			stopInformer = make(chan struct{})
			tmpDir, err = ioutil.TempDir("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())

			// create two files
			Expect(os.Create(tmpDir + "/" + "default_testvm.some-extension")).ToNot(BeNil())
			Expect(os.Create(tmpDir + "/" + "default1_testvm1.some-extension")).ToNot(BeNil())

			informer = cache.NewSharedIndexInformer(
				NewFileListWatchFromClient(tmpDir),
				&api.Domain{},
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

			go informer.Run(stopInformer)
			Expect(cache.WaitForCacheSync(stopInformer, informer.HasSynced)).To(BeTrue())

		})

		It("should update the cache with all files in the directory", func() {
			Expect(informer.GetStore().ListKeys()).To(HaveLen(2))
			_, exists, _ := informer.GetStore().GetByKey("default/testvm")
			Expect(exists).To(BeTrue())
			_, exists, _ = informer.GetStore().GetByKey("default1/testvm1")
			Expect(exists).To(BeTrue())
		})

		It("should detect a file creation", func() {
			Expect(os.Create(tmpDir + "/" + "default2_testvm2.some-extension")).ToNot(BeNil())
			Eventually(func() bool {
				_, exists, _ := informer.GetStore().GetByKey("default2/testvm2")
				return exists
			}).Should(BeTrue())
		})

		It("should detect a file deletion", func() {
			Expect(os.Remove(tmpDir + "/" + "default1_testvm1.some-extension")).To(Succeed())
			Eventually(func() bool {
				_, exists, _ := informer.GetStore().GetByKey("default1/testvm1")
				return exists
			}).Should(BeFalse())
		})
		Context("and something goes wrong", func() {
			It("should notify and abort when listing files", func() {
				lw := NewFileListWatchFromClient(tmpDir)
				// Deleting the watch directory should have some impact
				Expect(os.RemoveAll(tmpDir)).To(Succeed())
				_, err := lw.List(v1.ListOptions{})
				Expect(err).To(HaveOccurred())
			})
			It("should ignore invalid file content", func() {
				lw := NewFileListWatchFromClient(tmpDir)
				_, err := lw.List(v1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				i, err := lw.Watch(v1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				defer i.Stop()

				// Adding files in wrong formats should have an impact
				// TODO should we just ignore them?
				Expect(os.Create(tmpDir + "/" + "test.some-extension")).ToNot(BeNil())

				// No event should be received
				Consistently(i.ResultChan()).ShouldNot(Receive())
			})
		})

		AfterEach(func() {
			close(stopInformer)
			os.RemoveAll(tmpDir)
		})

	})
})
