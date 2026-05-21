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
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
)

func (k *kubevirt) ClusterProfiler() *ClusterProfiler {
	return &ClusterProfiler{
		restClient: k.restClient,
		resource:   "cluster-profiler",
	}
}

type ClusterProfiler struct {
	restClient *rest.RESTClient
	resource   string
}

func (v *ClusterProfiler) preferredVersion() (string, error) {
	var group metav1.APIGroup
	// First, find out which version to query
	uri := ApiGroupName
	result := v.restClient.Get().AbsPath(uri).Do(context.Background())
	if data, err := result.Raw(); err != nil {
		connErr, isConnectionErr := err.(*url.Error)

		if isConnectionErr {
			return "", connErr.Err
		}

		return "", err
	} else if err = json.Unmarshal(data, &group); err != nil {
		return "", err
	}

	return group.PreferredVersion.GroupVersion, nil

}

func (v *ClusterProfiler) Start() error {
	preferredVersion, err := v.preferredVersion()
	if err != nil {
		return err
	}

	// Now, query the preferred version
	uri := fmt.Sprintf("/apis/%s/start-cluster-profiler", preferredVersion)

	return v.restClient.Get().AbsPath(uri).Do(context.Background()).Error()
}

func (v *ClusterProfiler) Stop() error {
	preferredVersion, err := v.preferredVersion()
	if err != nil {

		return fmt.Errorf("error encountered while detecting preferred version: %v", err)
	}

	// Now, query the preferred version
	uri := fmt.Sprintf("/apis/%s/stop-cluster-profiler", preferredVersion)

	return v.restClient.Get().AbsPath(uri).Do(context.Background()).Error()
}

// Dump returns at most cpRequest.PageSize profiler results. To fetch results from all kubevirt pods
// Dump should be called with Continue fields set to Continue field value from the response to a previous request.
// This should be repeated until Continue or ComponentsResult field in ClusterProfilerResponse is empty.
func (v *ClusterProfiler) Dump(cpRequest *v1.ClusterProfilerRequest) (*v1.ClusterProfilerResults, error) {
	preferredVersion, err := v.preferredVersion()
	if err != nil {
		return nil, err
	}

	if cpRequest == nil {
		return nil, fmt.Errorf("request body can't be nil")
	}

	bytes, err := json.Marshal(cpRequest)
	if err != nil {
		return nil, err
	}

	// Now, query the preferred version
	uri := fmt.Sprintf("/apis/%s/dump-cluster-profiler", preferredVersion)

	var profileResults v1.ClusterProfilerResults

	result := v.restClient.Get().AbsPath(uri).Body(bytes).Do(context.Background())
	if data, err := result.Raw(); err != nil {
		connErr, isConnectionErr := err.(*url.Error)

		if isConnectionErr {
			return nil, connErr.Err
		}

		return nil, err
	} else if len(data) == 0 {
		return &profileResults, nil
	} else if err = json.Unmarshal(data, &profileResults); err != nil {
		return nil, err
	}

	return &profileResults, nil
}
