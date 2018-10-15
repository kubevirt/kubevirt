package imageupload_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestImageUpload(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ImageUpload Suite")
}
