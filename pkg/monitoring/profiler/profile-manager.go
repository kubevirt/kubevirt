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
	"sync"
)

type profileManager struct {
	lock sync.Mutex
}

type ProfilerOptions struct {
	ProfileProcess bool
	ProfileHTTP    bool
}

type ProfilerResults struct {
	ProcessProfilerResults map[string][]byte `json:"processProfilerResults,omitempty"`
	HTTPProfilerResults    map[string]int    `json:"httpProfilerResults,omitempty"`
}

var globalManager profileManager

func StartProfiler(options *ProfilerOptions) error {

	// make sure all profilers are stopped before
	// we attempt to start again
	err := StopProfiler(true)
	if err != nil {
		return err
	}

	if options.ProfileProcess {
		err = startProcessProfiler()
		if err != nil {
			StopProfiler(true)
			return err
		}
	}

	if options.ProfileHTTP {
		startHTTPProfiler()
	}

	return nil
}

func StopProfiler(clearResults bool) error {
	stopProcessProfiler(clearResults)
	stopHTTPProfiler(clearResults)

	return nil
}

func DumpProfilerResults() (string, error) {
	pprofResults, err := dumpProcessProfilerResults()
	if err != nil {
		return "", err
	}
	httpResults := dumpHTTPProfilerResults()

	profilerResults := &ProfilerResults{
		ProcessProfilerResults: pprofResults,
		HTTPProfilerResults:    httpResults,
	}

	b, err := json.MarshalIndent(profilerResults, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil

}
