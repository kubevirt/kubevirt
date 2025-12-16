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

			expectedKubeletRootFlag := "--kubelet-root\n      - " + tt.kubeletRoot
			if !strings.Contains(rendered, expectedKubeletRootFlag) {
				t.Errorf("expected rendered manifest to contain %q, got:\n%s", expectedKubeletRootFlag, rendered)
			}

			expectedKubeletPodsDirFlag := "--kubelet-pods-dir\n      - " + tt.kubeletRoot + "/pods"
			if !strings.Contains(rendered, expectedKubeletPodsDirFlag) {
				t.Errorf("expected rendered manifest to contain %q, got:\n%s", expectedKubeletPodsDirFlag, rendered)
			}
		})
	}
}
