package vgpuhook_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVGPUHook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "vGPUHook Suite")
}
