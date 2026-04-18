/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package admitter_test

import (
	"testing"

	"kubevirt.io/client-go/testutils"
)

func TestAdmitter(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

type stubClusterConfigChecker struct {
	bridgeBindingOnPodNetEnabled   bool
	passtBindingFeatureGateEnabled bool
}

func (s stubClusterConfigChecker) PasstBindingEnabled() bool { return s.passtBindingFeatureGateEnabled }

func (s stubClusterConfigChecker) IsBridgeInterfaceOnPodNetworkEnabled() bool {
	return s.bridgeBindingOnPodNetEnabled
}
