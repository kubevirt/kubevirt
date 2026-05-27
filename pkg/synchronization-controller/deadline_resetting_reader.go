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
 *
 */

package synchronization

import (
	"net"
	"time"
)

// deadlineResettingReader wraps a connection and resets the idle timeout
// on both src and dst connections after each successful read. This allows
// io.Copy to be used while still preventing idle connection hangs.
//
// The reader automatically extends the deadline on both connections whenever
// data flows, ensuring active connections never timeout while idle connections
// fail after the specified timeout period.
type deadlineResettingReader struct {
	src     net.Conn
	dst     net.Conn
	timeout time.Duration
}

// NewDeadlineResettingReader creates a reader that resets idle timeouts on both
// src and dst connections after each successful read.
//
// This is designed to wrap one side of a bidirectional TCP proxy to prevent
// connections from hanging when one side stops sending data without closing
// the connection.
//
// Parameters:
//   - src: The connection to read from
//   - dst: The connection being written to (deadline will be reset on both)
//   - timeout: How long connections can be idle before timing out (must be positive)
//
// Panics if timeout is zero or negative.
func NewDeadlineResettingReader(src, dst net.Conn, timeout time.Duration) *deadlineResettingReader {
	if timeout <= 0 {
		panic("deadline resetting reader requires positive timeout")
	}
	return &deadlineResettingReader{
		src:     src,
		dst:     dst,
		timeout: timeout,
	}
}

// Read implements io.Reader by reading from the source connection and
// automatically resetting the idle timeout on both connections after
// each successful read.
func (r *deadlineResettingReader) Read(p []byte) (n int, err error) {
	r.src.SetReadDeadline(time.Now().Add(r.timeout))
	n, err = r.src.Read(p)
	if n > 0 {
		// Data was successfully read from src, extend deadline on both connections
		// Both deadlines must be extended because:
		// 1. src deadline ensures we detect when the sender stops sending
		// 2. dst deadline ensures the writer goroutine also gets extended timeout
		// Without updating dst, the writing goroutine could timeout even while data flows
		r.src.SetDeadline(time.Now().Add(r.timeout))
		r.dst.SetDeadline(time.Now().Add(r.timeout))
	}
	return
}
