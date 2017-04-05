package kubecli_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestKubecli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubecli Suite")
}
