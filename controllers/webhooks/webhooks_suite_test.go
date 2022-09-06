package webhooks

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOperators(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhooks Controller Suite")
}
