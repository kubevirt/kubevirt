package container_disk_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestContainerDisk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ContainerDisk Suite")
}
