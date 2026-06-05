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
 * Copyright The KubeVirt Authors.
 *
 */

package main

import (
	"testing"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

func TestFeatureGateReport(t *testing.T) {
	gates := featuregate.GetRegisteredFeatureGates()
	if len(gates) == 0 {
		t.Fatal("expected registered feature gates, got none")
	}

	for name, fg := range gates {
		if fg.State == featuregate.GA || fg.State == featuregate.Discontinued {
			continue
		}
		if name == "" {
			t.Error("found feature gate with empty name")
		}
		if fg.State != featuregate.Alpha && fg.State != featuregate.Beta && fg.State != featuregate.Deprecated {
			t.Errorf("feature gate %q has unexpected state %q", name, fg.State)
		}
	}
}
