package vm_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOfflinevm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "vm Suite")
}
