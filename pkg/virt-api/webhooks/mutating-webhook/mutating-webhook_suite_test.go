package mutating_webhook_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestValidatingWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MutatingWebhook Suite")
}
