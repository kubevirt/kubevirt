package fuzz_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFuzz(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fuzz Suite")
}
