package virtio_serial_test

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestVirtioSerialServer(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
