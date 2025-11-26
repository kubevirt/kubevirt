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
	"encoding/json"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func TestMarshallObject(t *testing.T) {
	var imagePullSecret []v1.LocalObjectReference
	imagePullSecretsJSON, _ := json.Marshal(imagePullSecret)

	config := &operatorutil.KubeVirtDeploymentConfig{
		Namespace:       "{{.Namespace}}",
		Registry:        "",
		ImagePrefix:     "{{.DockerPrefix}}",
		KubeVirtVersion: "{{.DockerTag}}",
		AdditionalProperties: map[string]string{
			operatorutil.AdditionalPropertiesNamePullPolicy: string(v1.PullIfNotPresent),
			operatorutil.AdditionalPropertiesPullSecrets:    string(imagePullSecretsJSON),
		},
	}

	handler := components.NewHandlerDaemonSet(config, "", "", "")
	writer := strings.Builder{}

	MarshallObject(handler, &writer)

	result := writer.String()

	if !strings.Contains(result, "namespace: {{.Namespace}}") {
		t.Fail()
	}

}
