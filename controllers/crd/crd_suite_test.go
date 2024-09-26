package crd_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCrd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRD Suite")
}
