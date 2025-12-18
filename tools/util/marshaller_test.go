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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package util

import (
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

func TestMarshallObject(t *testing.T) {
	imagePullSecret := []v1.LocalObjectReference{}

	tests := []struct {
		name        string
		kubeletRoot string
	}{
		{
			name:        "default kubelet root",
			kubeletRoot: "/var/lib/kubelet",
		},
		{
			name:        "non-default kubelet root",
			kubeletRoot: "/var/lib/rancher/k3s/agent/kubelet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := components.NewHandlerDaemonSet(
				"{{.Namespace}}",
				"",
				"{{.DockerPrefix}}",
				"{{.DockerTag}}",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				"",
				v1.PullIfNotPresent,
				imagePullSecret,
				nil,
				"2",
				nil,
				false,
				tt.kubeletRoot,
			)

			var writer strings.Builder
			MarshallObject(handler, &writer)
			rendered := writer.String()

			if !strings.Contains(rendered, "namespace: {{.Namespace}}") {
				t.Errorf("expected rendered manifest to contain namespace placeholder")
			}

			// Check for kubelet-root flag and value as separate list items
			if !strings.Contains(rendered, "- --kubelet-root") {
				t.Errorf("expected rendered manifest to contain '- --kubelet-root', got:\n%s", rendered)
			}
			if !strings.Contains(rendered, "- "+tt.kubeletRoot) {
				t.Errorf("expected rendered manifest to contain '- %s', got:\n%s", tt.kubeletRoot, rendered)
			}

			// Check for kubelet-pods-dir flag and value as separate list items
			if !strings.Contains(rendered, "- --kubelet-pods-dir") {
				t.Errorf("expected rendered manifest to contain '- --kubelet-pods-dir', got:\n%s", rendered)
			}
			expectedPodsPath := tt.kubeletRoot + "/pods"
			if !strings.Contains(rendered, "- "+expectedPodsPath) {
				t.Errorf("expected rendered manifest to contain '- %s', got:\n%s", expectedPodsPath, rendered)
			}
		})
	}
}
