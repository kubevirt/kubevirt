package validating_webhook

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kunalkushwaha/go-sdk/log"
)

func TestValidatingWebhook(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "ValidatingWebhook Suite")
}
