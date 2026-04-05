/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package hotplug_volume

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestHotplugDisk(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
