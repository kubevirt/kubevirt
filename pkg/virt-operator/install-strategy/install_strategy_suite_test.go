package installstrategy

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInstallStrategy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "InstallStrategy Suite")
}
