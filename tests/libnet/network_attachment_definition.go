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

const postUrl = "/apis/k8s.cni.cncf.io/v1/namespaces/%s/network-attachment-definitions/%s"

func CreateNetworkAttachmentDefinition(name, namespace, netConf string) error {
	return kubevirt.Client().RestClient().
		Post().
		RequestURI(fmt.Sprintf(postUrl, namespace, name)).
		Body([]byte(netConf)).
		Do(context.Background()).
		Error()
}
