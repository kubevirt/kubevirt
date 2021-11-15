package cache

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
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

	Context("PodInfoCache", func() {

		obj := &PodCacheInterface{
			Iface: &v1.Interface{
				Model: "nice model",
			},
			PodIP: "random ip",
			PodIPs: []string{
				"ip1", "ip2",
			},
		}

		It("should return os.ErrNotExist if no cache entry exists", func() {
			vmi := &v1.VirtualMachineInstance{ObjectMeta: v12.ObjectMeta{UID: "123"}}
			_, err := cacheFactory.CacheForVMI(vmi).Read("abc")
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("should save and restore pod interface information", func() {
			vmi := &v1.VirtualMachineInstance{ObjectMeta: v12.ObjectMeta{UID: "123"}}

			Expect(cacheFactory.CacheForVMI(vmi).Write("abc", obj)).To(Succeed())
			newObj, err := cacheFactory.CacheForVMI(vmi).Read("abc")
			Expect(err).ToNot(HaveOccurred())
			Expect(newObj).To(Equal(obj))
		})
		It("should remove the cache file", func() {
			vmi := &v1.VirtualMachineInstance{ObjectMeta: v12.ObjectMeta{UID: "123"}}
			Expect(cacheFactory.CacheForVMI(vmi).Write("abc", obj)).To(Succeed())
			_, err := cacheFactory.CacheForVMI(vmi).Read("abc")
			Expect(err).ToNot(HaveOccurred())
			Expect(cacheFactory.CacheForVMI(vmi).Remove()).To(Succeed())
			_, err = cacheFactory.CacheForVMI(vmi).Read("abc")
			Expect(err).To(HaveOccurred())
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
