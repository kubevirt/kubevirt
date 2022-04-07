package common

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestControllerCommon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Common Suite")
}
