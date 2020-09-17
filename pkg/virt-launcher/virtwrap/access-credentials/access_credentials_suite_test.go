package accesscredentials_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAccessCredentials(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AccessCredentials Suite")
}
