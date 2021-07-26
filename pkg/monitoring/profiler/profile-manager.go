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
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	v1 "kubevirt.io/client-go/api/v1"

	restful "github.com/emicklei/go-restful"
)

type profileManager struct {
	lock sync.Mutex
}

type ProfilerResults struct {
	ProcessProfilerResults map[string][]byte `json:"processProfilerResults,omitempty"`
	HTTPProfilerResults    map[string]int    `json:"httpProfilerResults,omitempty"`
}

var globalManager profileManager

func startProfiler(options *v1.ClusterProfilerOptions) error {

	// make sure all profilers are stopped before
	// we attempt to start again
	err := stopProfiler(true)
	if err != nil {
		return err
	}

	if options.ProfileProcess {
		err = startProcessProfiler()
		if err != nil {
			stopProfiler(true)
			return err
		}
	}

	if options.ProfileHTTP {
		startHTTPProfiler()
	}

	return nil
}

func stopProfiler(clearResults bool) error {
	stopProcessProfiler(clearResults)
	stopHTTPProfiler(clearResults)

	return nil
}

func dumpProfilerResultString() (string, error) {
	pprofResults, err := dumpProcessProfilerResults()
	if err != nil {
		return "", err
	}
	httpResults := dumpHTTPProfilerResults()

	profilerResult := &v1.ProfilerResult{
		PprofData:         pprofResults,
		HTTPRequestCounts: httpResults,
	}

	b, err := json.MarshalIndent(profilerResult, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil

}

func dumpProfilerResult() (*v1.ProfilerResult, error) {
	pprofResults, err := dumpProcessProfilerResults()
	if err != nil {
		return nil, err
	}
	httpResults := dumpHTTPProfilerResults()

	profilerResults := &v1.ProfilerResult{
		PprofData:         pprofResults,
		HTTPRequestCounts: httpResults,
	}

	return profilerResults, nil

}

func HandleStartProfiler(_ *restful.Request, response *restful.Response) {

	options := &v1.ClusterProfilerOptions{
		ProfileProcess: true,
		ProfileHTTP:    true,
	}

	err := startProfiler(options)
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("could not start internal profiling: %v", err))
		return
	}
	response.WriteHeader(http.StatusOK)
}

func HandleStopProfiler(_ *restful.Request, response *restful.Response) {
	err := stopProfiler(false)
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("could not stop internal profiling: %v", err))
		return
	}

	response.WriteHeader(http.StatusOK)
}

func HandleDumpProfiler(_ *restful.Request, response *restful.Response) {

	res, err := dumpProfilerResult()
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, fmt.Sprintf("could not dump internal profiling: %v", err))
		return
	}

	response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
}
