package config

import (
	"testing"

	"kubevirt.io/client-go/testutils"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

func TestConfig(t *testing.T) {
	ephemeraldiskutils.MockDefaultOwnershipManager()
	testutils.KubeVirtTestSuiteSetup(t)
}
