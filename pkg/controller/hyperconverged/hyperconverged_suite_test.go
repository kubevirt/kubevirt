package hyperconverged_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHyperconverged(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hyperconverged Suite")
}
