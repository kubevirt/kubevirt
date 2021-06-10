package hostdisk

import (
	"testing"

	"kubevirt.io/client-go/testutils"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

func TestHostDisk(t *testing.T) {
	ephemeraldiskutils.MockDefaultOwnershipManager()
	testutils.KubeVirtTestSuiteSetup(t)
}
