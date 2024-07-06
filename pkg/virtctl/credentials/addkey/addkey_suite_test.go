package addkey_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAddKey(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AddKey Suite")
}
