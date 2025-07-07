package infer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInfer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Infer Suite")
}
