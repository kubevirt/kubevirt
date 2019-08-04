package selinux_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSelinux(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Selinux Suite")
}
