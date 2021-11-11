package trace

import (
	"time"

	"k8s.io/utils/trace"
)

type Tracer struct {
	Trace     *trace.Trace
	Threshold time.Duration
}

func NewTrace(threshold time.Duration, name string, field ...trace.Field) *Tracer {
	return &Tracer{
		Trace:     trace.New(name, field...),
		Threshold: threshold,
	}
}
