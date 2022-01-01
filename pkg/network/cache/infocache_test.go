package cache

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Infocache", func() {

	var tmpDir string
	var cacheFactory *interfaceCacheFactory

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "cache")
		Expect(err).ToNot(HaveOccurred())
		cacheFactory = NewInterfaceCacheFactoryWithBasePath(tmpDir)
		dutils.MockDefaultOwnershipManager()
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("DomainInfoCache", func() {
		obj := &api.Interface{
			Model: &api.Model{Type: "a nice model"},
		}
		It("should return os.ErrNotExist if no cache entry exists", func() {
			_, err := cacheFactory.CacheDomainInterfaceForPID("123").Read("abc")
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("should save and restore pod interface information", func() {
			Expect(cacheFactory.CacheDomainInterfaceForPID("123").Write("abc", obj)).To(Succeed())
			newObj, err := cacheFactory.CacheDomainInterfaceForPID("123").Read("abc")
			Expect(err).ToNot(HaveOccurred())
			Expect(newObj).To(Equal(obj))
		})
	})
})
