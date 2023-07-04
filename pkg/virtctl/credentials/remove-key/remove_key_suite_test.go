package remove_key_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRemoveKey(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RemoveKey Suite")
}
