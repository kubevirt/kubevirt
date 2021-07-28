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

package profiler

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"k8s.io/client-go/rest"
)

type counter struct {
	counter     map[string]int
	lock        sync.Mutex
	isProfiling bool
}

var globalCounter counter

func startHTTPProfiler() {
	globalCounter.lock.Lock()
	defer globalCounter.lock.Unlock()
	globalCounter.counter = make(map[string]int)
	globalCounter.isProfiling = true
}

func stopHTTPProfiler(clearResults bool) {
	globalCounter.lock.Lock()
	defer globalCounter.lock.Unlock()
	globalCounter.isProfiling = false

	if clearResults {
		globalCounter.counter = make(map[string]int)
	}
}

func dumpHTTPProfilerResults() map[string]int {
	globalCounter.lock.Lock()
	defer globalCounter.lock.Unlock()

	newCopy := make(map[string]int)
	for k, v := range globalCounter.counter {
		newCopy[k] = v
	}

	return newCopy
}

type rtWrapper struct {
	origRoundTripper http.RoundTripper
}

func isProfiling() bool {
	globalCounter.lock.Lock()
	isProfiling := globalCounter.isProfiling
	globalCounter.lock.Unlock()

	return isProfiling
}

func (r *rtWrapper) RoundTrip(request *http.Request) (*http.Response, error) {

	if !isProfiling() {
		return r.origRoundTripper.RoundTrip(request)
	}

	pathSplit := strings.Split(request.URL.Path, "/")
	var resource string
	var operation string

	isWatch := false
	individualResource := false

	if pathSplit[1] == "api" {
		// native k8s api
		if len(pathSplit) < 4 {
			return r.origRoundTripper.RoundTrip(request)
		}

		if len(pathSplit) >= 5 && pathSplit[3] == "watch" {
			// example URL   -  /api/v1/watch/namespaces/kubevirt/pods
			// example URL   -  /api/v1/watch/pods
			isWatch = true
			resource = pathSplit[4]
		} else if len(pathSplit) >= 6 && pathSplit[3] == "namespaces" {
			// example URL   -  /api/v1/namespaces/kubevirt/endpoints/virt-controller
			// example split - 0|1  |2 |3         |4       |5        |6
			resource = pathSplit[5]
			if len(pathSplit) >= 7 {
				individualResource = true
			}
		} else {
			// example URL   -  /api/v1/endpoints
			resource = pathSplit[3]
			if len(pathSplit) >= 5 {
				individualResource = true
			}
		}

	} else if pathSplit[1] == "apis" {
		// Custom Resource api

		// example URL   -  /apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/vmi-ephemeral
		// example split - 0|1   |2          |3       |4         |5      |6

		if len(pathSplit) < 5 {
			return r.origRoundTripper.RoundTrip(request)
		}

		if len(pathSplit) >= 6 && pathSplit[4] == "watch" {
			// example URL   -  /apis/kubevirt.io/v1/watch/namespaces/kubevirt/virtualmachineinstances
			// example URL   -  /apis/kubevirt.io/v1/watch/virtualmachineinstances
			isWatch = true
			resource = pathSplit[5]
		} else if len(pathSplit) >= 7 && pathSplit[4] == "namespaces" {
			// example URL   -  /apis/kubevirt.io/v1/namespaces/kubevirt/virtualmachineinstances
			// example split - 0|1   |2          |3 |4         |5       |6
			resource = pathSplit[6]
			if len(pathSplit) >= 8 {
				// example URL   -  /apis/kubevirt.io/v1/namespaces/kubevirt/virtualmachineinstances/myvm
				// example split - 0|1   |2          |3 |4         |5       |6                      |7
				individualResource = true
			}
		} else {
			// example URL   -  /apis/kubevirt.io/v1alpha3/kubevirts
			// example split - 0|1   |2          |3       |4
			resource = pathSplit[4]
			if len(pathSplit) >= 6 {
				// example URL   -  /apis/kubevirt.io/v1alpha3/kubevirts/my-kubevirt
				individualResource = true
			}
		}

	} else {
		// unknown
		return r.origRoundTripper.RoundTrip(request)
	}

	operation = request.Method
	switch request.Method {
	case "GET":
		operation = "GET"
		if isWatch {
			operation = "WATCH"
		} else if !individualResource {
			operation = "LIST"
		}
	case "PUT":
		operation = "UPDATE"
	case "PATCH":
		operation = "PATCH"
	case "POST":
		operation = "CREATE"
	case "DELETE":
		operation = "DELETE"
	}

	key := fmt.Sprintf("%s %s", operation, resource)

	// Don't use a defer for the Unlock because we don't want to hold the
	// lock during the round tripper execution during the return
	globalCounter.lock.Lock()
	val, _ := globalCounter.counter[key]
	globalCounter.counter[key] = val + 1
	globalCounter.lock.Unlock()

	return r.origRoundTripper.RoundTrip(request)
}

func AddHTTPRoundTripProfiler(config *rest.Config) {
	fn := func(rt http.RoundTripper) http.RoundTripper {
		return &rtWrapper{
			origRoundTripper: rt,
		}
	}
	config.Wrap(fn)
}
