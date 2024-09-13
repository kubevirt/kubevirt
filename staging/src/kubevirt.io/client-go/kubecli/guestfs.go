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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package kubecli

import (
	"context"

	"encoding/json"
	"fmt"
	"net/url"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type GuestfsInfo struct {
	Registry    string `json:"registry"`
	Tag         string `json:"tag"`
	Digest      string `json:"digest"`
	ImagePrefix string `json:"imagePrefix"`
	GsImage     string `json:"gsImage"`
}

func (k *kubevirtClient) GuestfsVersion() *GuestfsVersion {
	return &GuestfsVersion{
		restClient: k.restClient,
		resource:   "guestfs",
	}
}

type GuestfsVersion struct {
	restClient *rest.RESTClient
	resource   string
}

func (v *GuestfsVersion) Get() (*GuestfsInfo, error) {
	var group metav1.APIGroup
	// First, find out which version to query
	uri := ApiGroupName
	result := v.restClient.Get().AbsPath(uri).Do(context.Background())
	if data, err := result.Raw(); err != nil {
		connErr, isConnectionErr := err.(*url.Error)

		if isConnectionErr {
			return nil, connErr.Err
		}

		return nil, err
	} else if err = json.Unmarshal(data, &group); err != nil {
		return nil, err
	}

	// Now, query the preferred version
	uri = fmt.Sprintf("/apis/%s/guestfs", group.PreferredVersion.GroupVersion)
	var info GuestfsInfo

	result = v.restClient.Get().AbsPath(uri).Do(context.Background())
	if data, err := result.Raw(); err != nil {
		connErr, isConnectionErr := err.(*url.Error)

		if isConnectionErr {
			return nil, connErr.Err
		}

		return nil, err
	} else if err = json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
