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
}

var globalManager profileManager

func startProfiler() error {

	// make sure all profilers are stopped before
	// we attempt to start again
	err := stopProfiler(true)
	if err != nil {
		return err
	}

	err = startProcessProfiler()
	if err != nil {
		stopProfiler(true)
		return err
	}

	return nil
}

func stopProfiler(clearResults bool) error {
	stopProcessProfiler(clearResults)

	return nil
}

func dumpProfilerResultString() (string, error) {
	pprofResults, err := dumpProcessProfilerResults()
	if err != nil {
		return "", err
	}
	profilerResult := &v1.ProfilerResult{
		PprofData: pprofResults,
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
	profilerResults := &v1.ProfilerResult{
		PprofData: pprofResults,
	}

	return profilerResults, nil

}

func HandleStartProfiler(_ *restful.Request, response *restful.Response) {

	err := startProfiler()
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
