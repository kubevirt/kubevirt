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

package libnet

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

func NewNetAttachDef(name, config string) *nadv1.NetworkAttachmentDefinition {
	return &nadv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       nadv1.NetworkAttachmentDefinitionSpec{Config: config},
	}
}

func NewNetConfig(name, pluginConfig string) string {
	const cniVersion = "0.3.1"
	netConfig := fmt.Sprintf(`{
      "cniVersion": %q,
      "name": %q,
      "plugins": [%s]
	}`,
		cniVersion, name, pluginConfig,
	)
	var nc map[string]interface{}
	if err := json.Unmarshal([]byte(netConfig), &nc); err != nil {
		panic(fmt.Sprintf("failed to unmarshal:\n%s\nerror: %v", netConfig, err))
	}
	return netConfig
}

func NewNetPluginConfig(cniType string, conf map[string]interface{}) string {
	if conf == nil {
		conf = map[string]interface{}{}
	}
	conf["type"] = cniType

	pluginConfig, err := json.Marshal(conf)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal:\n%s\nerror: %v", conf, err))
	}
	return string(pluginConfig)
}

func CreateNetAttachDef(
	ctx context.Context, namespace string, netAttachDef *nadv1.NetworkAttachmentDefinition,
) (*nadv1.NetworkAttachmentDefinition, error) {
	kvclient := kubevirt.Client()
	return kvclient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Create(
		ctx, netAttachDef, metav1.CreateOptions{},
	)
}
