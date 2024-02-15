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
	"fmt"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

const PasstNetAttDef = "netbindingpasst"

func CreatePasstNetworkAttachmentDefinition(namespace string) error {
	const passtType = "kubevirt-passt-binding" // #nosec G101
	const netAttDefFmt = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":%q,"namespace":%q},` +
		`"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"%s\", \"plugins\": [{\"type\": \"%s\"}]}"}}`
	return CreateNetworkAttachmentDefinition(
		PasstNetAttDef,
		namespace,
		fmt.Sprintf(netAttDefFmt, PasstNetAttDef, namespace, PasstNetAttDef, passtType),
	)
}

func CreateMacvtapNetworkAttachmentDefinition(namespace, networkName, macvtapLowerDevice string) error {
	const macvtapNADFmt = `{
		"apiVersion":"k8s.cni.cncf.io/v1",
		"kind":"NetworkAttachmentDefinition",
		"metadata":{
			"name":"%s",
			"namespace":"%s", 
			"annotations": {
				"k8s.v1.cni.cncf.io/resourceName": "macvtap.network.kubevirt.io/%s"
			}
		},
		"spec":{
			"config":"{\"cniVersion\": \"0.3.1\",\"name\": \"%s\",\"type\": \"macvtap\"}"
		}
	}`
	macvtapNad := fmt.Sprintf(macvtapNADFmt, networkName, namespace, macvtapLowerDevice, networkName)
	return CreateNetworkAttachmentDefinition(networkName, namespace, macvtapNad)
}

func CreateNetworkAttachmentDefinition(name, namespace, netConf string) error {
	const postURL = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"
	return kubevirt.Client().RestClient().
		Post().
		RequestURI(fmt.Sprintf(postURL, namespace, name)).
		Body([]byte(netConf)).
		Do(context.Background()).
		Error()
}
