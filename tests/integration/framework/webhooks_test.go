/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package framework

import (
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Webhook shutdown", func() {
	It("waits for an active handler to finish", func(ctx SpecContext) {
		entered := make(chan struct{})
		release := make(chan struct{})
		var releaseOnce sync.Once
		unblock := func() { releaseOnce.Do(func() { close(release) }) }
		DeferCleanup(unblock)
		f := &Framework{
			webhookServer: &http.Server{
				ReadHeaderTimeout: webhookReadHeaderTimeout,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					close(entered)
					<-release
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			webhookDone: make(chan struct{}),
		}
		var lc net.ListenConfig
		listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		go func() {
			defer GinkgoRecover()
			defer close(f.webhookDone)
			Expect(f.webhookServer.Serve(listener)).To(MatchError(http.ErrServerClosed))
		}()
		DeferCleanup(func() { Expect(f.webhookServer.Close()).To(Succeed()) })

		requestDone := make(chan error, 1)
		go func() {
			req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listener.Addr().String(), http.NoBody)
			if requestErr == nil {
				var response *http.Response
				response, requestErr = http.DefaultClient.Do(req)
				if requestErr == nil {
					_, requestErr = io.Copy(io.Discard, response.Body)
					_ = response.Body.Close()
				}
			}
			requestDone <- requestErr
		}()
		Eventually(ctx, entered).Should(BeClosed())
		stopped := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(stopped)
			f.Stop()
		}()
		Eventually(ctx, f.webhookDone).Should(BeClosed(), "Serve should exit while the handler is still active")
		Consistently(ctx, stopped, 100*time.Millisecond).ShouldNot(BeClosed())
		unblock()
		Eventually(ctx, requestDone).Should(Receive(Succeed()))
		Eventually(ctx, stopped).Should(BeClosed())
	}, SpecTimeout(5*time.Second))
})
