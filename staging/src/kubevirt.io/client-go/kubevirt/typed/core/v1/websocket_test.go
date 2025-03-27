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
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = ginkgo.Describe("Websocket", func() {
	ginkgo.Context("data proxied through our websocket proxy", func() {
		var proxy *httptest.Server
		var target *httptest.Server
		var receivedDataHash hash.Hash
		var done chan error
		ginkgo.BeforeEach(func() {
			done = make(chan error)
			receivedDataHash = sha256.New()
			target = newTargetServer(receivedDataHash, done)
			proxy = newProxyServer(target)
		})
		ginkgo.AfterEach(func() {
			proxy.Close()
			target.Close()
		})
		ginkgo.It("should transfer arbitrary sized packets which are bigger and smaller than the websocket buffer", func() {
			proxyCon := dial(proxy)
			defer proxyCon.Close()
			messages := [][]byte{
				[]byte(rand.String(WebsocketMessageBufferSize - 10)),
				[]byte(rand.String(WebsocketMessageBufferSize + 10)),
				[]byte(rand.String(10)),
				[]byte(rand.String(WebsocketMessageBufferSize*3 + 10)),
			}

			expectedDataHash := sha256.New()
			writer := binaryWriter{conn: proxyCon}
			for _, msg := range messages {
				expectedDataHash.Write(msg)
				_, err := writer.Write(msg)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
			err := proxyCon.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			gomega.Expect(err).ToNot(gomega.HaveOccurred(), "failed to write close message")
			err = <-done
			gomega.Expect(err).ToNot(gomega.HaveOccurred(), "target server did not receive a propler close message")
			gomega.Expect(fmt.Sprintf("%x", expectedDataHash.Sum(nil))).To(gomega.Equal(fmt.Sprintf("%x", receivedDataHash.Sum(nil))))
		})
	})
})

func newTargetServer(writer io.Writer, done chan error) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer ginkgo.GinkgoRecover()
		upgrader := NewUpgrader()
		targetCon, err := upgrader.Upgrade(w, r, nil)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		_, err = CopyFrom(writer, targetCon)
		done <- err
	}))
}

func newProxyServer(target *httptest.Server) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer ginkgo.GinkgoRecover()
		upgrader := NewUpgrader()
		src, err := upgrader.Upgrade(w, r, nil)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		targetURL := "ws" + strings.TrimPrefix(target.URL, "http")
		dst, _, err := Dial(targetURL, nil)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		defer dst.Close()
		_, _ = Copy(dst, src)
	}))
}

func dial(proxy *httptest.Server) *websocket.Conn {
	u := "ws" + strings.TrimPrefix(proxy.URL, "http")
	proxyCon, _, err := Dial(u, nil)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return proxyCon
}
