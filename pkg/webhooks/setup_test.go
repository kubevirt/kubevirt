package webhooks

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	pkgDirectory = "pkg/webhooks"
	testFilesLoc = "testFiles"
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "webhooks Suite")
}

var _ = Describe("Hyperconverged API: Webhook", func() {
	Context("Test GetWebhookCertDir", func() {

		BeforeEach(func() {
			os.Unsetenv(webHookCertDirEnv)
		})

		AfterEach(func() {
			os.Unsetenv(webHookCertDirEnv)
		})

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

		It("should setup the webhooks with the manager", func() {
			const expectedNs = "mynamespace"
			_ = os.Setenv(hcoutil.OperatorNamespaceEnv, expectedNs)

			testFilesLocation := getTestFilesLocation()
			os.Setenv(webHookCertDirEnv, testFilesLocation)

			resources := []client.Object{}
			cl := commontestutils.InitClient(resources)
			s := scheme.Scheme

			ws := webhook.NewServer(webhook.Options{})
			mgr, err := commontestutils.NewManagerMock(&rest.Config{}, manager.Options{WebhookServer: ws, Scheme: s}, cl, logger)
			Expect(err).ToNot(HaveOccurred())

			Expect(SetupWebhookWithManager(context.TODO(), mgr, true, nil)).To(Succeed())
		})

	})
})

func getTestFilesLocation() string {
	wd, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	if strings.HasSuffix(wd, pkgDirectory) {
		return testFilesLoc
	}
	return path.Join(pkgDirectory, testFilesLoc)
}
