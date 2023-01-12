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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package kubevirt

import "kubevirt.io/client-go/kubecli"

// Client returns a KubeVirt client.
//
// The backend client is a singleton, therefore,
// an error could have occurred only once,
// any repeating call would just repeat the same results.
//
// In the context of tests, it is enough to declare panic,
// declaring that there is a problem in the test suite.
// This wrapper is in place to avoid the need to repeat the
// handling of the error each time.
func Client() kubecli.KubevirtClient {
	client, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	return client
}
