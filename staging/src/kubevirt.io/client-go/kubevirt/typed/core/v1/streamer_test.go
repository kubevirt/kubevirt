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
 * Copyright The KubeVirt Authors
 *
 */

package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"
)

var _ = Describe("wsStreamer", func() {
	var server *httptest.Server
	var config *rest.Config

	BeforeEach(func() {
		server = newEchoWebsocketServer()
		config = &rest.Config{Host: server.URL}
	})

	AfterEach(func() {
		server.Close()
	})

	It("should not leak a goroutine when AsConn() is closed instead of Stream()", func() {
		quiesce()
		before := runtime.NumGoroutine()

		stream, err := AsyncSubresourceHelper(config, "virtualmachineinstances", "default", "testvmi", "vsock", nil)
		Expect(err).ToNot(HaveOccurred())

		conn := stream.AsConn()
		Expect(conn.Close()).To(Succeed())

		quiesce()
		after := runtime.NumGoroutine()
		Expect(after).To(BeNumerically("<=", before), "leaked goroutines after AsConn().Close(): before=%d after=%d\n%s", before, after, goroutineDump())
		Expect(webSocketCallbackGoroutines()).To(BeZero(), "leaked a goroutine parked in WebsocketCallback after AsConn().Close():\n%s", goroutineDump())
	})

	It("should not leak a goroutine when Stream() is used (control case)", func() {
		quiesce()
		before := runtime.NumGoroutine()

		stream, err := AsyncSubresourceHelper(config, "virtualmachineinstances", "default", "testvmi", "vsock", nil)
		Expect(err).ToNot(HaveOccurred())

		in := strings.NewReader("bye")
		var out bytes.Buffer
		_ = stream.Stream(StreamOptions{In: in, Out: &out})

		quiesce()
		after := runtime.NumGoroutine()
		Expect(after).To(BeNumerically("<=", before), "did not expect a leak when using Stream(): before=%d after=%d\n%s", before, after, goroutineDump())
		Expect(webSocketCallbackGoroutines()).To(BeZero(), "did not expect a goroutine parked in WebsocketCallback when using Stream():\n%s", goroutineDump())
	})

	It("should not accumulate goroutines over repeated AsConn() connect/close cycles", func() {
		const iterations = 20

		quiesce()
		before := runtime.NumGoroutine()

		for range iterations {
			stream, err := AsyncSubresourceHelper(config, "virtualmachineinstances", "default", "testvmi", "vsock", nil)
			Expect(err).ToNot(HaveOccurred())

			conn := stream.AsConn()
			Expect(conn.Close()).To(Succeed())
		}

		quiesce()
		after := runtime.NumGoroutine()
		Expect(after).To(BeNumerically("<=", before), "expected no accumulation over %d iterations, got before=%d after=%d\n%s", iterations, before, after, goroutineDump())
		Expect(webSocketCallbackGoroutines()).To(BeZero(), "expected no goroutines parked in WebsocketCallback over %d iterations:\n%s", iterations, goroutineDump())
	})

	It("should not panic when the AsConn() connection is closed twice", func() {
		stream, err := AsyncSubresourceHelper(config, "virtualmachineinstances", "default", "testvmi", "vsock", nil)
		Expect(err).ToNot(HaveOccurred())

		conn := stream.AsConn()
		Expect(conn.Close()).To(Succeed())
		// A second Close() may return an error (the underlying connection is
		// already closed), but it must not panic.
		Expect(func() { _ = conn.Close() }).ToNot(Panic())
	})
})

// newEchoWebsocketServer upgrades every request to a websocket and holds
// it open until the client hangs up.
func newEchoWebsocketServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		upgrader := NewUpgrader()
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
}

// quiesce lets background goroutines settle before sampling NumGoroutine().
func quiesce() {
	for range 3 {
		runtime.Gosched()
	}
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
}

// goroutineDump captures a full goroutine profile for debugging leaks.
func goroutineDump() string {
	var buf bytes.Buffer
	Expect(pprof.Lookup("goroutine").WriteTo(&buf, 2)).To(Succeed())
	return buf.String()
}

// webSocketCallbackGoroutines counts goroutines currently parked in AsyncWSRoundTripper.WebsocketCallback.
func webSocketCallbackGoroutines() int {
	return strings.Count(goroutineDump(), "WebsocketCallback")
}
