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
		contextClient, _, _, err := clientconfig.ClientAndNamespaceFromContext(
			clientconfig.NewContext(context.Background(), clientConfig),
		)
		Expect(err).ToNot(HaveOccurred())

		By("verifying the clients have the same config")
		Expect(contextClient.Config()).To(Equal(client.Config()))
	})

	It("ClientAndNamespaceFromContext should fail when clientConfig is missing from context", func() {
		_, _, _, err := clientconfig.ClientAndNamespaceFromContext(context.Background())
		Expect(err).To(MatchError("unable to get client config from context"))
	})

	It("WithOverriddenNamespace should override the namespace and preserve the original client config", func() {
		By("setting up a base clientConfig")
		clientConfig := kubecli.DefaultClientConfig(&pflag.FlagSet{})

		By("creating a client from the base clientConfig")
		client, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
		Expect(err).ToNot(HaveOccurred())

		By("storing the clientConfig in a context")
		ctx := clientconfig.NewContext(context.Background(), clientConfig)

		By("creating a new context with a namespace override")
		newCtx, err := clientconfig.WithOverriddenNamespace(ctx, "test")
		Expect(err).ToNot(HaveOccurred())

		By("retrieving the client and namespace from the overridden context")
		contextClient, namespace, overridden, err := clientconfig.ClientAndNamespaceFromContext(newCtx)
		Expect(err).ToNot(HaveOccurred())

		By("verifying that the overridden namespace is returned")
		Expect(namespace).To(Equal("test"))
		Expect(overridden).To(BeTrue())

		By("verifying that the underlying client config remains unchanged")
		Expect(contextClient.Config()).To(Equal(client.Config()))
	})
})
