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

package evacuation

import (
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
)

func FuzzGenerateNewMigration(f *testing.F) {
	for _, vmi := range []*v1.VirtualMachineInstance{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "fuzz-vmi",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "descheduled-vmi",
				Annotations: map[string]string{
					v1.EvictionSourceAnnotation: "descheduler",
				},
			},
		},
	} {
		raw, err := json.Marshal(vmi)
		if err != nil {
			f.Fatalf("failed to marshal the seed VMI: %v", err)
		}
		f.Add(raw, "node01")
	}

	f.Fuzz(func(t *testing.T, raw []byte, key string) {
		vmi := &v1.VirtualMachineInstance{}
		if err := json.Unmarshal(raw, vmi); err != nil {
			return
		}

		migration := GenerateNewMigration(vmi, key)
		if migration == nil {
			t.Fatal("migration generation returned nil")
		}
		if migration.Spec.VMIName != vmi.Name {
			t.Fatalf("migration VMI name %q does not match %q", migration.Spec.VMIName, vmi.Name)
		}
		if migration.GenerateName != "kubevirt-evacuation-" {
			t.Fatalf("unexpected migration generate name %q", migration.GenerateName)
		}
		if migration.Annotations[v1.EvacuationMigrationAnnotation] != key {
			t.Fatalf("migration evacuation key %q does not match %q", migration.Annotations[v1.EvacuationMigrationAnnotation], key)
		}

		expectedPriority := v1.PrioritySystemCritical
		if vmi.Annotations[v1.EvictionSourceAnnotation] == "descheduler" {
			expectedPriority = v1.PrioritySystemMaintenance
		}
		if migration.Spec.Priority == nil || *migration.Spec.Priority != expectedPriority {
			t.Fatalf("unexpected migration priority %v, expected %q", migration.Spec.Priority, expectedPriority)
		}
	})
}
