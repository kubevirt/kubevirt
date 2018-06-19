package device_manager_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDeviceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DeviceManager Suite")
}
