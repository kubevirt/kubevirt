package cache_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("DomainInterfaceCache", func() {
	var cacheCreator tempCacheCreator

	obj := &api.Interface{
		Model: &api.Model{Type: "a nice model"},
	}

	BeforeEach(dutils.MockDefaultOwnershipManager)

	AfterEach(func() {
		Expect(cacheCreator.New("").Delete()).To(Succeed())
	})

	It("should return os.ErrNotExist if no cache entry exists", func() {
		domainIfaceCache, err := cache.NewDomainInterfaceCache(&cacheCreator, "123").IfaceEntry("abc")
		Expect(err).NotTo(HaveOccurred())
		_, err = domainIfaceCache.Read()
		Expect(err).To(MatchError(os.ErrNotExist))
	})

	It("should save and restore pod interface information", func() {
		domainIfaceCache, err := cache.NewDomainInterfaceCache(&cacheCreator, "123").IfaceEntry("abc")
		Expect(err).NotTo(HaveOccurred())
		Expect(domainIfaceCache.Write(obj)).To(Succeed())
		newObj, err := domainIfaceCache.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(newObj).To(Equal(obj))
	})

	It("should delete pod interface from the cache", func() {
		domainIfaceCache, err := cache.NewDomainInterfaceCache(&cacheCreator, "123").IfaceEntry("abc")
		Expect(err).NotTo(HaveOccurred())
		Expect(domainIfaceCache.Write(obj)).To(Succeed())
		newObj, err := domainIfaceCache.Read()
		Expect(err).NotTo(HaveOccurred())
		Expect(newObj).To(Equal(obj))

		Expect(domainIfaceCache.Delete()).To(Succeed())
		_, err = domainIfaceCache.Read()
		Expect(err).To(MatchError(os.ErrNotExist))
	})
})
