/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package emptydisk

import (
	"testing"

	"kubevirt.io/client-go/testutils"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

func TestEmptydisk(t *testing.T) {
	ephemeraldiskutils.MockDefaultOwnershipManager()
	testutils.KubeVirtTestSuiteSetup(t)
}
