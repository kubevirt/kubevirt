package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVirtLauncher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtLauncher Suite")
}
