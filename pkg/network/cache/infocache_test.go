package cache

import (
	"io/ioutil"
	"os"

	"kubevirt.io/kubevirt/pkg/os/fs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Infocache", func() {

	var tmpDir string
	var cacheFactory *interfaceCacheFactory
	var cacheCreator tempCacheCreator

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "cache")
		Expect(err).ToNot(HaveOccurred())
		cacheFactory = NewInterfaceCacheFactoryWithBasePath(tmpDir)
		dutils.MockDefaultOwnershipManager()
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
		cacheCreator.New("").Delete()
	})

	Context("PodInfoCache", func() {
		const UID = "123"
		var podIfaceCache PodInterfaceCacheStore
		var cacheData PodCacheInterface

		BeforeEach(func() {
			podCache := NewPodInterfaceCache(cacheCreator, UID)

			var err error
			podIfaceCache, err = podCache.IfaceEntry("net0")
			Expect(err).NotTo(HaveOccurred())

			cacheData = PodCacheInterface{
				Iface: &v1.Interface{
					Model: "nice model",
				},
				PodIP: "random ip",
				PodIPs: []string{
					"ip1", "ip2",
				},
			}
		})

		It("should return os.ErrNotExist if no cache entry exists", func() {
			_, err := podIfaceCache.Read()
			Expect(err).To(MatchError(os.ErrNotExist))
		})
		It("should save and restore pod interface information", func() {
			Expect(podIfaceCache.Write(&cacheData)).To(Succeed())
			Expect(podIfaceCache.Read()).To(Equal(&cacheData))
		})
		It("should remove the cache file", func() {
			Expect(podIfaceCache.Write(&cacheData)).To(Succeed())
			Expect(podIfaceCache.Remove()).To(Succeed())

			_, err := podIfaceCache.Read()
			Expect(err).To(MatchError(os.ErrNotExist))
		})
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

type tempCacheCreator struct{}

func (_ tempCacheCreator) New(filePath string) *Cache {
	tmpDir, err := ioutil.TempDir("", "temp-cache")
	if err != nil {
		panic("Unable to create temp cache directory")
	}
	return NewCustomCache(filePath, fs.NewWithRootPath(tmpDir))
}
