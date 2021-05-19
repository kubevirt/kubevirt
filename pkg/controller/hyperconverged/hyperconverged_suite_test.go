package hyperconverged

import (
	"os"
	"path"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	pkgDirectory = "pkg/controller/hyperconverged"
	testFilesLoc = "test-files"
)

func TestHyperconverged(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hyperconverged Suite")
}

func getTestFilesLocation() string {
	wd, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	if strings.HasSuffix(wd, pkgDirectory) {
		return testFilesLoc
	}
	return path.Join(pkgDirectory, testFilesLoc)
}
