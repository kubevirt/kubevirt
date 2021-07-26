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
	if len(pathSplit) < 7 {
		return r.origRoundTripper.RoundTrip(request)
	}

	group := pathSplit[2]
	version := pathSplit[3]
	//namespace := pathSplit[5]
	resource := pathSplit[6]

	method := request.Method
	key := fmt.Sprintf("%s/%s/%s/%s", group, version, resource, method)

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
