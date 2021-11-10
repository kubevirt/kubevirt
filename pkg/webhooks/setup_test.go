package webhooks

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("Hyperconverged API: Webhook", func() {
	Context("Test GetWebhookCertDir", func() {
		It("should return default value, if the env var is not set", func() {
			Expect(GetWebhookCertDir()).Should(Equal(hcoutil.DefaultWebhookCertDir))
		})

		It("should return the value of the env var, if set", func() {
			env := os.Getenv(webHookCertDirEnv)
			defer os.Setenv(webHookCertDirEnv, env)

			const somethingElse = "/something/else"
			os.Setenv(webHookCertDirEnv, somethingElse)
			Expect(GetWebhookCertDir()).Should(Equal(somethingElse))
		})
	})
})
