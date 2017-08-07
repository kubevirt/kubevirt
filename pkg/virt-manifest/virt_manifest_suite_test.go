package virt_manifest_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVirtManifest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtManifest Suite")
}
