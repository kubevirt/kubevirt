package password_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSetPassword(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SetPassword Suite")
}
