package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVirtChroot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virt Chroot Suite")
}
