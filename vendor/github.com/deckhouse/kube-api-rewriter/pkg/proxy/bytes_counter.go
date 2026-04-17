/*
Copyright 2024 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxy

import (
	"io"
	"sync/atomic"
)

func BytesCounterReaderWrap(r io.Reader) io.ReadCloser {
	return &bytesCounter{origReader: r}
}

func BytesCounterWriterWrap(w io.Writer) io.Writer {
	return &bytesCounter{origWriter: w}
}

var _ io.ReadCloser = &bytesCounter{}
var _ io.Writer = &bytesCounter{}

type bytesCounter struct {
	origReader io.Reader
	origWriter io.Writer
	counter    atomic.Int64
}

func (r *bytesCounter) Read(p []byte) (n int, err error) {
	l, err := r.origReader.Read(p)
	r.counter.Add(int64(l))
	return l, err
}

func (r *bytesCounter) Write(p []byte) (n int, err error) {
	l, err := r.origWriter.Write(p)
	r.counter.Add(int64(l))
	return l, err
}

func (r *bytesCounter) Close() error {
	return nil
}

func (r *bytesCounter) Reset() {
	r.counter.Store(0)
}

func (r *bytesCounter) Count() int {
	return int(r.counter.Load())
}

func CounterReset(wrapped interface{}) {
	if bytesCounter, ok := wrapped.(*bytesCounter); ok {
		bytesCounter.Reset()
	}
}

func CounterValue(wrapped interface{}) int {
	if bytesCounter, ok := wrapped.(*bytesCounter); ok {
		return bytesCounter.Count()
	}
	return 0
}
