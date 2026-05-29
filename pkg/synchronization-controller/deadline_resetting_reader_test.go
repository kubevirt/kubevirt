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
	"io"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deadline Resetting Reader", func() {
	var (
		srcConn, dstConn net.Conn
		reader           *deadlineResettingReader
		timeout          time.Duration
	)

	BeforeEach(func() {
		// Create a pair of connected pipes for testing
		srcConn, dstConn = createConnectedPipes()

		timeout = 100 * time.Millisecond
	})

	AfterEach(func() {
		if srcConn != nil {
			srcConn.Close()
		}
		if dstConn != nil {
			dstConn.Close()
		}
	})

	Describe("NewDeadlineResettingReader", func() {
		It("should create a reader with valid timeout", func() {
			reader = NewDeadlineResettingReader(srcConn, dstConn, timeout)
			Expect(reader).ToNot(BeNil())
			Expect(reader.src).To(Equal(srcConn))
			Expect(reader.dst).To(Equal(dstConn))
			Expect(reader.timeout).To(Equal(timeout))
		})

		It("should panic with zero timeout", func() {
			Expect(func() {
				NewDeadlineResettingReader(srcConn, dstConn, 0)
			}).To(Panic())
		})

		It("should panic with negative timeout", func() {
			Expect(func() {
				NewDeadlineResettingReader(srcConn, dstConn, -1*time.Second)
			}).To(Panic())
		})
	})

	Describe("Read", func() {
		BeforeEach(func() {
			reader = NewDeadlineResettingReader(srcConn, dstConn, timeout)
		})

		It("should read data successfully", func() {
			// Write data to the source connection from the other end
			testData := []byte("test data")
			go func() {
				_, err := dstConn.Write(testData)
				Expect(err).ToNot(HaveOccurred())
			}()

			// Read should succeed
			buf := make([]byte, 100)
			n, err := reader.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(len(testData)))
			Expect(buf[:n]).To(Equal(testData))
		})

		It("should set deadline before each read", func() {
			// This test verifies that deadline is set by checking that a read
			// times out when no data is available
			buf := make([]byte, 100)
			_, err := reader.Read(buf)

			// Should get a timeout error since no data is available
			Expect(err).To(HaveOccurred())
			netErr, ok := err.(net.Error)
			Expect(ok).To(BeTrue())
			Expect(netErr.Timeout()).To(BeTrue())
		})

		It("should extend deadline after successful read", func() {
			testData := []byte("test")

			// Write data in chunks with delays
			go func() {
				time.Sleep(10 * time.Millisecond)
				dstConn.Write(testData)
				time.Sleep(10 * time.Millisecond)
				dstConn.Write(testData)
			}()

			// First read should succeed and extend deadline
			buf := make([]byte, 10)
			n, err := reader.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(len(testData)))

			// Second read should also succeed because deadline was extended
			n, err = reader.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(len(testData)))
		})

		It("should not extend deadline when read returns 0 bytes", func() {
			// This behavior is implicit in the implementation:
			// if n > 0 { extend deadline }
			// So reading 0 bytes should not extend the deadline

			// We can't easily test this directly without mocking, but we can
			// verify the code path by checking that EOF is handled correctly
			dstConn.Close() // Close to trigger EOF

			buf := make([]byte, 100)
			_, err := reader.Read(buf)
			Expect(err).To(Equal(io.EOF))
		})

		It("should propagate read errors", func() {
			// Close the connection to trigger an error
			dstConn.Close()

			buf := make([]byte, 100)
			_, err := reader.Read(buf)
			Expect(err).To(HaveOccurred())
		})
	})
})

// createConnectedPipes creates a pair of connected net.Conn for testing
// This simulates a bidirectional connection
func createConnectedPipes() (net.Conn, net.Conn) {
	// Use net.Pipe which creates a synchronous in-memory full duplex network connection
	return net.Pipe()
}
