package bootstrap_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBootstrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bootstrap Suite")
}
