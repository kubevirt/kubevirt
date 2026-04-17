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

package log

import (
	"bytes"
	"fmt"
	"io"
)

// ReaderLogger is ReadCloser implementation that catches content
// while underlying Reader is being read, e.g. with io.Copy.
// Content is copied into the buffer and may be used after copying
// for logging or other handling.
type ReaderLogger struct {
	wrappedReader io.ReadCloser
	buf           bytes.Buffer
}

func NewReaderLogger(r io.Reader) *ReaderLogger {
	rdr := &ReaderLogger{}
	rdr.wrappedReader = io.NopCloser(io.TeeReader(r, &rdr.buf))
	return rdr
}

func (r *ReaderLogger) Read(p []byte) (n int, err error) {
	return r.wrappedReader.Read(p)
}

func (r *ReaderLogger) Close() error {
	return r.wrappedReader.Close()
}

func HeadString(obj interface{}, limit int) string {
	readLog, ok := obj.(*ReaderLogger)
	if !ok {
		return ""
	}
	bufLen := readLog.buf.Len()
	bufStr := readLog.buf.String()
	if bufLen < limit {
		return bufStr
	}
	return bufStr[0:limit]
}

func HeadStringEx(obj interface{}, limit int) string {
	s := HeadString(obj, limit)
	if s == "" {
		return "<empty>"
	}
	return fmt.Sprintf("[%d] %s", len(s), s)
}

func HasData(obj interface{}) bool {
	readLog, ok := obj.(*ReaderLogger)
	if !ok {
		return false
	}
	return readLog.buf.Len() > 0
}

func Bytes(obj interface{}) []byte {
	readLog, ok := obj.(*ReaderLogger)
	if !ok {
		return nil
	}
	return readLog.buf.Bytes()
}
