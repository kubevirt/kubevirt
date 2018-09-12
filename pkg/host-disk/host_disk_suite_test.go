package hostdisk_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHostDisk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HostDisk Suite")
}
