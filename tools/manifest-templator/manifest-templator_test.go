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

import "testing"

func TestInfraReplicasToUint8(t *testing.T) {
	for _, tc := range []struct {
		replicas uint
		want     uint8
		wantErr  bool
	}{
		{replicas: 1, want: 1},
		{replicas: 255, want: 255},
		{replicas: 256, wantErr: true},
		{replicas: 1000, wantErr: true},
	} {
		got, err := infraReplicasToUint8(tc.replicas)
		if (err != nil) != tc.wantErr {
			t.Errorf("infraReplicasToUint8(%d) error = %v, wantErr %v", tc.replicas, err, tc.wantErr)
		}
		if got != tc.want {
			t.Errorf("infraReplicasToUint8(%d) = %d, want %d", tc.replicas, got, tc.want)
		}
	}
}
