/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2024 Red Hat, Inc.
 *
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
	bridgeBindingOnPodNetEnabled bool
	macvtapFeatureGateEnabled    bool
	passtFeatureGateEnabled      bool
	bindingPluginFGEnabled       bool
}

func (s stubClusterConfigChecker) IsBridgeInterfaceOnPodNetworkEnabled() bool {
	return s.bridgeBindingOnPodNetEnabled
}

func (s stubClusterConfigChecker) MacvtapEnabled() bool {
	return s.macvtapFeatureGateEnabled
}

func (s stubClusterConfigChecker) PasstEnabled() bool {
	return s.passtFeatureGateEnabled
}
