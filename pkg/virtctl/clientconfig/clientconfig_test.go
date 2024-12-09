package clientconfig_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

var _ = Describe("client", func() {
	It("NewContext should store a clientConfig and make it retrievable with FromContext", func() {
		By("creating a clientConfig")
		clientConfig := kubecli.DefaultClientConfig(&pflag.FlagSet{})

		By("creating a client from context")
		clientConfigFromContext, err := clientconfig.FromContext(
			clientconfig.NewContext(context.Background(), clientConfig),
		)
		Expect(err).ToNot(HaveOccurred())

		By("verifying the created config and config from context are the same")
		Expect(clientConfig).To(Equal(clientConfigFromContext))
	})

	It("FromContext should fail when clientConfig is missing from context", func() {
		clientConfig, err := clientconfig.FromContext(context.Background())
		Expect(err).To(MatchError("unable to get client config from context"))
		Expect(clientConfig).To(BeNil())
	})
})
