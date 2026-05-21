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

package admitters

import (
	"context"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"

	"kubevirt.io/api/plugin"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type PluginAdmitter struct {
	Config *virtconfig.ClusterConfig
}

func NewPluginAdmitter(config *virtconfig.ClusterConfig) *PluginAdmitter {
	return &PluginAdmitter{Config: config}
}

func (admitter *PluginAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != plugin.GroupName {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected group: %s, expected: %s", ar.Request.Resource.Group, plugin.GroupName))
	}
	if ar.Request.Resource.Resource != plugin.ResourcePluginPlural {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource: %s, expected: %s", ar.Request.Resource.Resource, plugin.ResourcePluginPlural))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.PluginsEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("Plugins feature gate is not enabled"))
	}

	return validating_webhooks.NewPassingAdmissionResponse()
}
