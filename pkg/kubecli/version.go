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

package kubecli

import (
	"encoding/json"
	"net/url"

	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/version"
)

func (k *kubevirt) ServerVersion() *ServerVersion {
	return &ServerVersion{
		restClient: k.restClient,
		resource:   "version",
	}
}

type ServerVersion struct {
	restClient *rest.RESTClient
	resource   string
}

func (v *ServerVersion) Get() (*version.Info, error) {
	result := v.restClient.Get().RequestURI("/apis/subresources.kubevirt.io/v1alpha2/version").Do()
	data, err := result.Raw()
	if err != nil {
		connErr, isConnectionErr := err.(*url.Error)

		if isConnectionErr {
			return nil, connErr.Err
		}

		return nil, err
	}

	var serverInfo version.Info
	json.Unmarshal(data, &serverInfo)
	if err != nil {
		return nil, err
	}

	return &serverInfo, nil
}
