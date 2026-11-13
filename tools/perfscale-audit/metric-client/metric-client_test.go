/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package metricclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	auditapi "kubevirt.io/kubevirt/tools/perfscale-audit/api"
)

func TestPrometheusAuthorization(t *testing.T) {
	tests := []struct {
		name string
		cfg  auditapi.InputConfig
		want []string
	}{
		{
			name: "basic credentials",
			cfg:  auditapi.InputConfig{PrometheusUserName: "user", PrometheusPassword: "pass"},
			want: []string{"Basic dXNlcjpwYXNz"},
		},
		{
			name: "bearer token",
			cfg:  auditapi.InputConfig{PrometheusBearerToken: "test-token"},
			want: []string{"Bearer test-token"},
		},
		{
			name: "bearer token takes precedence over basic credentials",
			cfg:  auditapi.InputConfig{PrometheusUserName: "user", PrometheusPassword: "pass", PrometheusBearerToken: "test-token"},
			want: []string{"Bearer test-token"},
		},
		{name: "no credentials"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Values("Authorization"); !slices.Equal(got, tt.want) {
					t.Errorf("Authorization header = %q, want %q", got, tt.want)
				}
				_, _ = io.WriteString(w, `{"status":"success","data":{}}`)
			}))
			defer server.Close()

			tt.cfg.PrometheusURL = server.URL
			if _, err := NewMetricClient(&tt.cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}
