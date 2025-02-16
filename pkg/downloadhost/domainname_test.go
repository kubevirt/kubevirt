package downloadhost_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/downloadhost"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "domainname")
}

var _ = Describe("domainname", func() {
	var valueBefore downloadhost.CLIDownloadHost
	BeforeEach(func() {
		valueBefore = downloadhost.Get()
	})

	AfterEach(func() {
		downloadhost.Set(valueBefore)
	})

	It("should set and get the domain name", func() {
		info := downloadhost.CLIDownloadHost{
			DefaultHost: "aaa",
			CurrentHost: "bbb",
			Cert:        "ccc",
			Key:         "ddd",
		}
		Expect(downloadhost.Set(info)).To(BeTrue())
		Expect(downloadhost.Get()).To(Equal(info))
	})

	It("should return true if the value was changed", func() {
		infoBefore := downloadhost.CLIDownloadHost{
			DefaultHost: "aaa",
			CurrentHost: "bbb",
			Cert:        "ccc",
			Key:         "ddd",
		}

		infoToSet := downloadhost.CLIDownloadHost{
			DefaultHost: "111",
			CurrentHost: "222",
			Cert:        "333",
			Key:         "444",
		}

		downloadhost.Set(infoBefore)
		Expect(downloadhost.Get()).To(Equal(infoBefore))
		Expect(downloadhost.Set(infoBefore)).To(BeFalse())

		Expect(downloadhost.Set(infoToSet)).To(BeTrue())
		Expect(downloadhost.Get()).To(Equal(infoToSet))
	})
})
