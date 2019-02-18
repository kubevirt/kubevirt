package approver_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApprover(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Approver Suite")
}
