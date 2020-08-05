package openapi_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestOpenapi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Openapi Suite")
}
