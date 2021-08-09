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
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
)

type pprofData struct {
	cpuf        *os.File
	isProfiling bool
	hasResults  bool
	lock        sync.Mutex
}

var ProcessProfileBaseDir = "/profile-data"
var cpuProfileFilePath = filepath.Join(ProcessProfileBaseDir, "cpu.pprof")

func (p *pprofData) startProcessProfiler() error {
	var err error

	p.lock.Lock()
	defer p.lock.Unlock()

	p.hasResults = false
	p.isProfiling = true

	p.cpuf, err = os.Create(cpuProfileFilePath)
	if err != nil {
		return err
	}

	if err := pprof.StartCPUProfile(p.cpuf); err != nil {
		return err
	}

	return nil
}

func (p *pprofData) stopProcessProfiler(clearResults bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	pprof.StopCPUProfile()

	p.cpuf.Close()
	p.hasResults = true
	p.isProfiling = false
	if clearResults {
		p.hasResults = false
	}

}

func (p *pprofData) dumpProcessProfilerResults() (map[string][]byte, error) {
	var err error
	res := make(map[string][]byte)

	dumpTypes := []string{
		"heap",
		"goroutine",
		"allocs",
		"threadcreate",
		"block",
		"mutex",
		"cpu",
	}

	// Run garbage collector in order to clean up the "heap" dump so it's more useful
	runtime.GC()
	for _, dump := range dumpTypes {
		var b []byte
		if dump == "cpu" {
			p.lock.Lock()
			defer p.lock.Unlock()
			if !p.hasResults {
				continue
			}
			b, err = ioutil.ReadFile(cpuProfileFilePath)
			if err != nil {
				return res, err
			}
		} else {
			var buf bytes.Buffer
			if err := pprof.Lookup(dump).WriteTo(&buf, 2); err != nil {
				return res, err
			}
			b = buf.Bytes()
		}
		res[dump+".pprof"] = b
	}

	return res, nil
}
