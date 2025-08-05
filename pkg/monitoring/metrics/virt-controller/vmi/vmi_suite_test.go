package vmi_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVmi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VMI collectors Suite")
}
