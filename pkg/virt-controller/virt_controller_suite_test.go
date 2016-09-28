package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestVirtController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtController Suite")
}
