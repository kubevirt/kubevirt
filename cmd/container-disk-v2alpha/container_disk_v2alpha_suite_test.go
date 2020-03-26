package container_disk_v2alpha_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestContainerDiskV2alpha(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ContainerDiskV2alpha Suite")
}
