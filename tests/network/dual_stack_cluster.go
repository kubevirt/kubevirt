package network

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
)

var _ = SIGDescribe("Dual stack cluster network configuration", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred(), "Should successfully initialize an API client")
	})

	Context("when dual stack cluster configuration is enabled", func() {
		Specify("the cluster must be dual stack", func() {
			if flags.SkipDualStackTests {
				Skip("user requested the dual stack check on the live cluster to be skipped")
			}

			isClusterDualStack, err := libnet.IsClusterDualStack(virtClient)
			Expect(err).NotTo(HaveOccurred(), "must be able to infer the dual stack configuration from the live cluster")
			Expect(isClusterDualStack).To(BeTrue(), "the live cluster should be in dual stack mode")
		})
	})
})
