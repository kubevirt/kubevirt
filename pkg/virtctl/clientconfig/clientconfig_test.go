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

		By("creating a client from the clientConfig")
		client, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
		Expect(err).ToNot(HaveOccurred())

		By("creating a client from context")
		contextClient, _, _, _, err := clientconfig.ClientAndNamespaceFromContext(
			clientconfig.NewContext(context.Background(), clientConfig),
		)
		Expect(err).ToNot(HaveOccurred())

		By("verifying the clients have the same config")
		Expect(contextClient.Config()).To(Equal(client.Config()))
	})

	It("ClientAndNamespaceFromContext should fail when clientConfig is missing from context", func() {
		_, _, _, _, err := clientconfig.ClientAndNamespaceFromContext(context.Background())
		Expect(err).To(MatchError("unable to get client config from context"))
	})
})
