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
 * Copyright The KubeVirt Authors.
 */

package trace

import (
	"sync"
	"time"

	"k8s.io/utils/trace"
)

type Tracer struct {
	traceMap map[string]*trace.Trace
	mux      sync.Mutex

	Threshold time.Duration
}

func (t *Tracer) StartTrace(key string, name string, field ...trace.Field) {
	t.mux.Lock()
	defer t.mux.Unlock()
	if t.traceMap == nil {
		t.traceMap = make(map[string]*trace.Trace)
	}
	t.traceMap[key] = trace.New(name, field...)
	return
}

func (t *Tracer) StopTrace(key string) {
	if key == "" {
		return
	}
	t.mux.Lock()
	defer t.mux.Unlock()
	if _, ok := t.traceMap[key]; !ok {
		return
	}
	t.traceMap[key].LogIfLong(t.Threshold)
	delete(t.traceMap, key)
	return
}

// A trace Step adds a new step with a specific message.
// Call StepTrace after an execution step to record how long it took.
func (t *Tracer) StepTrace(key string, name string, field ...trace.Field) {
	// Trace shouldn't be making noise unless the Trace is slow.
	// Fail silently on errors like empty or incorrect keys.
	if key == "" {
		return
	}
	t.mux.Lock()
	defer t.mux.Unlock()
	if _, ok := t.traceMap[key]; !ok {
		return
	}
	t.traceMap[key].Step(name, field...)
	return
}
