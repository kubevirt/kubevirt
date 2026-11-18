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

func TestShouldContinueDumpPagination(t *testing.T) {
	tests := []struct {
		name            string
		requestContinue string
		resultContinue  string
		want            bool
	}{
		{name: "no continue token", want: false},
		{name: "first page has more", resultContinue: "page-2", want: true},
		{name: "token advanced", requestContinue: "page-2", resultContinue: "page-3", want: true},
		{name: "stuck token", requestContinue: "page-2", resultContinue: "page-2", want: false},
		{name: "last page", requestContinue: "page-2", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldContinueDumpPagination(tt.requestContinue, tt.resultContinue)
			if got != tt.want {
				t.Fatalf("shouldContinueDumpPagination(%q, %q) = %v, want %v", tt.requestContinue, tt.resultContinue, got, tt.want)
			}
		})
	}
}
