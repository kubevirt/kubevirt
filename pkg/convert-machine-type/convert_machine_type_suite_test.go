package convertmachinetype_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConvertMachineType(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MassMachineTypeTransition Suite")
}
