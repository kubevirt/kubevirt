package operands

import (
	"os"
	"path"
	"strings"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/gomega"
)

const (
	pkgDirectory = "controllers/operands"
	testFilesLoc = "testFiles"
)

var (
	qsCrd = &extv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind: "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: consoleQuickStartCrdName,
		},
	}
)

func getTestFilesLocation() string {
	wd, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	if strings.HasSuffix(wd, pkgDirectory) {
		return testFilesLoc
	}
	return path.Join(pkgDirectory, testFilesLoc)
}
