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

package rbac

import (
	"slices"
	"testing"
)

func TestHandlerCanReadMigrationTargetPod(t *testing.T) {
	found := false
	for _, rule := range newHandlerClusterRole().Rules {
		if !slices.Contains(rule.APIGroups, "") || !slices.Contains(rule.Resources, "pods") {
			continue
		}
		found = true
		if len(rule.Resources) != 1 || len(rule.Verbs) != 1 || rule.Verbs[0] != "get" {
			t.Fatalf("migration readiness requires only get access to pods, got %+v", rule)
		}
	}
	if !found {
		t.Fatal("virt-handler cannot read target Pod readiness")
	}
}
