package validating_webhook_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestValidatingWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ValidatingWebhook Suite")
}
