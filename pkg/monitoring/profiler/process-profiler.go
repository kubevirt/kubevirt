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
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
)

type pprofData struct {
	cpuf        *os.File
	memf        *os.File
	isProfiling bool
	hasResults  bool
	lock        sync.Mutex
}

var globalProcessProfiler pprofData

var ProcessProfileBaseDir = "/profile-data"

var cpuProfileFilePath = filepath.Join(ProcessProfileBaseDir, "cpu-profile.pprof")
var memProfileFilePath = filepath.Join(ProcessProfileBaseDir, "mem-profile.pprof")

func startProcessProfiler() error {
	var err error

	globalProcessProfiler.lock.Lock()
	defer globalProcessProfiler.lock.Unlock()

	globalProcessProfiler.hasResults = false
	globalProcessProfiler.isProfiling = true

	globalProcessProfiler.cpuf, err = os.Create(cpuProfileFilePath)
	if err != nil {
		return err
	}

	if err := pprof.StartCPUProfile(globalProcessProfiler.cpuf); err != nil {
		return err
	}

	globalProcessProfiler.memf, err = os.Create(memProfileFilePath)
	if err != nil {
		return err
	}

	runtime.GC()
	if err = pprof.WriteHeapProfile(globalProcessProfiler.memf); err != nil {
		return err
	}

	return nil
}

func stopProcessProfiler(clearResults bool) {
	globalProcessProfiler.lock.Lock()
	defer globalProcessProfiler.lock.Unlock()

	pprof.StopCPUProfile()
	globalProcessProfiler.cpuf.Close()
	globalProcessProfiler.memf.Close()
	globalProcessProfiler.hasResults = true
	globalProcessProfiler.isProfiling = false
	if clearResults {
		globalProcessProfiler.hasResults = false
	}

}

func dumpProcessProfilerResults() (map[string][]byte, error) {
	var err error
	res := make(map[string][]byte)

	globalProcessProfiler.lock.Lock()
	defer globalProcessProfiler.lock.Unlock()
	if !globalProcessProfiler.hasResults {
		return res, nil
	}

	res["cpu-profile-dump"], err = ioutil.ReadFile(cpuProfileFilePath)
	if err != nil {
		return res, err
	}
	res["mem-profile-dump"], err = ioutil.ReadFile(memProfileFilePath)
	if err != nil {
		return res, err
	}

	return res, nil
}
