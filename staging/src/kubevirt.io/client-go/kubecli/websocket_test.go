package kubecli

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("Websocket", func() {

	Context("data proxied through our websocket proxy", func() {
		var proxy *httptest.Server
		var target *httptest.Server
		var receivedDataHash hash.Hash
		var done chan error
		BeforeEach(func() {
			done = make(chan error)
			receivedDataHash = sha256.New()
			target = newTargetServer(receivedDataHash, done)
			proxy = newProxyServer(target)
		})
		AfterEach(func() {
			proxy.Close()
			target.Close()
		})
		It("should transfer arbitrary sized packets which are bigger and smaller than the websocket buffer", func() {
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
				Expect(err).ToNot(HaveOccurred())
			}
			err := proxyCon.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			Expect(err).ToNot(HaveOccurred(), "failed to write close message")
			err = <-done
			Expect(err).ToNot(HaveOccurred(), "target server did not receive a propler close message")
			Expect(fmt.Sprintf("%x", expectedDataHash.Sum(nil))).To(Equal(fmt.Sprintf("%x", receivedDataHash.Sum(nil))))
		})
	})

})

func newTargetServer(writer io.Writer, done chan error) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		upgrader := NewUpgrader()
		targetCon, err := upgrader.Upgrade(w, r, nil)
		Expect(err).ToNot(HaveOccurred())
		_, err = CopyFrom(writer, targetCon)
		done <- err
	}))
}

func newProxyServer(target *httptest.Server) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		upgrader := NewUpgrader()
		src, err := upgrader.Upgrade(w, r, nil)
		Expect(err).ToNot(HaveOccurred())
		targetURL := "ws" + strings.TrimPrefix(target.URL, "http")
		dst, _, err := Dial(targetURL, nil)
		Expect(err).ToNot(HaveOccurred())
		defer dst.Close()
		_, _ = Copy(dst, src)
	}))
}

func dial(proxy *httptest.Server) *websocket.Conn {
	u := "ws" + strings.TrimPrefix(proxy.URL, "http")
	proxyCon, _, err := Dial(u, nil)
	Expect(err).ToNot(HaveOccurred())
	return proxyCon
}
