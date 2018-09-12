package mutating_webhook_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMutatingWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MutatingWebhook Suite")
}
