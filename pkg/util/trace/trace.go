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
}

// StepTrace A trace Step adds a new step with a specific message.
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
}
