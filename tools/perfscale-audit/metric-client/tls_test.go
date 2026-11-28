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
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	auditapi "kubevirt.io/kubevirt/tools/perfscale-audit/api"
)

func TestPrometheusTLSVerification(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"status":"success","data":{}}`)
	}))
	defer server.Close()

	for _, tt := range []struct {
		name       string
		config     string
		wantReject bool
	}{
		{"omitted", `{}`, true},
		{"null", `{"prometheusVerifyTLS":null}`, true},
		{"enabled", `{"prometheusVerifyTLS":true}`, true},
		{"disabled", `{"prometheusVerifyTLS":false}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := auditapi.InputConfig{PrometheusURL: server.URL}
			if err := json.Unmarshal([]byte(tt.config), &cfg); err != nil {
				t.Fatal(err)
			}

			_, err := NewMetricClient(&cfg)
			if tt.wantReject {
				var unknownCA x509.UnknownAuthorityError
				if !errors.As(err, &unknownCA) {
					t.Fatalf("expected an untrusted certificate error, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("expected connection to succeed with verification disabled: %v", err)
			}
		})
	}
}
