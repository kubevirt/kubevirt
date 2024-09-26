package descheduler_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescheduler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Descheduler Suite")
}
