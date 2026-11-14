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

package rest

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/gorilla/websocket"
	"github.com/mdlayher/vsock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	virtvsock "kubevirt.io/kubevirt/pkg/virt-handler/vsock"
)

var _ = Describe("Console handler", func() {
	var (
		ctrl                  *gomock.Controller
		mockIsolationDetector *isolation.MockPodIsolationDetector
		mockIsolationResult   *isolation.MockIsolationResult
		handler               *ConsoleHandler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockIsolationDetector = isolation.NewMockPodIsolationDetector(ctrl)
		mockIsolationResult = isolation.NewMockIsolationResult(ctrl)
		handler = NewConsoleHandler(mockIsolationDetector, nil, nil)
	})

	Describe("getUnixSocketPath", func() {
		It("should resolve the socket path via safepath", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")
			vmi.UID = "test-uid-1234"

			socketDir := filepath.Join(tmpDir, "run", "kubevirt-private", string(vmi.UID))
			Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())

			socketFile := filepath.Join(socketDir, "virt-serial0")
			l, err := net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			defer l.Close()

			root, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().MountRoot().Return(root, nil)

			p, err := handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).ToNot(HaveOccurred())
			Expect(unsafepath.UnsafeAbsolute(p.Raw())).To(Equal(filepath.Join(tmpDir, "run", "kubevirt-private", string(vmi.UID), "virt-serial0")))
		})

		It("should fail when Detect returns an error", func() {
			vmi := api.NewMinimalVMI("testvmi")
			mockIsolationDetector.EXPECT().Detect(vmi).Return(nil, fmt.Errorf("detect failed"))

			_, err := handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).To(MatchError("detect failed"))
		})

		It("should fail when MountRoot returns an error", func() {
			vmi := api.NewMinimalVMI("testvmi")
			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().MountRoot().Return(nil, fmt.Errorf("mountroot failed"))

			_, err := handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).To(MatchError("mountroot failed"))
		})

		It("should fail when the socket does not exist", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")
			vmi.UID = "test-uid-1234"

			root, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().MountRoot().Return(root, nil)

			_, err = handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).To(HaveOccurred())
		})

		It("should reject a path containing a symlink", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")
			vmi.UID = "test-uid-1234"

			realDir := filepath.Join(tmpDir, "real-run", "kubevirt-private", string(vmi.UID))
			Expect(os.MkdirAll(realDir, 0755)).To(Succeed())

			socketFile := filepath.Join(realDir, "virt-serial0")
			l, err := net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			defer l.Close()

			// Create "run" as a symlink to "real-run" to simulate an attack
			Expect(os.Symlink(filepath.Join(tmpDir, "real-run"), filepath.Join(tmpDir, "run"))).To(Succeed())

			root, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
			Expect(err).ToNot(HaveOccurred())
			mockIsolationDetector.EXPECT().Detect(vmi).Return(mockIsolationResult, nil)
			mockIsolationResult.EXPECT().MountRoot().Return(root, nil)

			_, err = handler.getUnixSocketPath(vmi, "virt-serial0")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("unixSocketDialer", func() {
		It("should connect to a unix socket via safepath", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")

			socketFile := filepath.Join(tmpDir, "test.sock")
			l, err := net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			defer l.Close()

			socketPath, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir, "test.sock")
			Expect(err).ToNot(HaveOccurred())

			dial := unixSocketDialer(vmi, socketPath)
			conn, err := dial()
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			conn.Close()
		})

		It("should fail when the socket path is invalid", func() {
			tmpDir := GinkgoT().TempDir()
			vmi := api.NewMinimalVMI("testvmi")

			// Create the file so safepath resolves, then close the listener
			// (Go's UnixListener unlinks the socket on Close)
			socketFile := filepath.Join(tmpDir, "gone.sock")
			l, err := net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())

			socketPath, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir, "gone.sock")
			Expect(err).ToNot(HaveOccurred())

			l.Close()

			dial := unixSocketDialer(vmi, socketPath)
			_, err = dial()
			Expect(err).To(HaveOccurred())
		})
	})
})

// fakeTLSConn satisfies vsock.TLSConn by wrapping an already connected
// net.Conn, so the VSOCK dialer can be exercised without a real AF_VSOCK
// socket.
type fakeTLSConn struct {
	net.Conn
}

func (fakeTLSConn) Handshake() error { return nil }

var _ = Describe("Console handler streaming", func() {
	const (
		testUID    = types.UID("console-uid-1234")
		streamWait = 10 * time.Second
	)

	var (
		mockIsolationDetector *isolation.MockPodIsolationDetector
		mockIsolationResult   *isolation.MockIsolationResult
		handler               *ConsoleHandler
		vmiStore              cache.Store
		vmi                   *v1.VirtualMachineInstance
		tmpDir                string
		socketDir             string
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		mockIsolationDetector = isolation.NewMockPodIsolationDetector(ctrl)
		mockIsolationResult = isolation.NewMockIsolationResult(ctrl)

		tmpDir = GinkgoT().TempDir()
		socketDir = filepath.Join(tmpDir, "run", "kubevirt-private", string(testUID))
		Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())

		root, err := safepath.JoinAndResolveWithRelativeRoot(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		mockIsolationDetector.EXPECT().Detect(gomock.Any()).Return(mockIsolationResult, nil).AnyTimes()
		mockIsolationResult.EXPECT().MountRoot().Return(root, nil).AnyTimes()
		mockIsolationResult.EXPECT().Pid().Return(1234).AnyTimes()

		vmi = libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithName(testVMIName),
			libvmi.WithUID(testUID),
		)
		vmiStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		Expect(vmiStore.Add(vmi)).To(Succeed())

		handler = NewConsoleHandler(mockIsolationDetector, vmiStore, nil)
	})

	// listenLauncher stands in for virt-launcher: it creates the unix socket
	// the handler is expected to proxy to and hands over accepted connections.
	listenLauncher := func(socketName string) chan net.Conn {
		listener, err := net.Listen("unix", filepath.Join(socketDir, socketName))
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { listener.Close() })

		conns := make(chan net.Conn, 1)
		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				conns <- conn
			}
		}()
		return conns
	}

	// startServer exposes handlerFn over HTTP with the namespace/name path
	// parameters the handlers read already resolved.
	startServer := func(handlerFn restful.RouteFunction) *httptest.Server {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, httpRequest *http.Request) {
			defer GinkgoRecover()
			request := restful.NewRequest(httpRequest)
			request.PathParameters()["namespace"] = testNamespace
			request.PathParameters()["name"] = testVMIName
			handlerFn(request, restful.NewResponse(writer))
		}))
		DeferCleanup(func() { server.Close() })
		return server
	}

	dialWebsocket := func(server *httptest.Server, query string) *websocket.Conn {
		clientSocket, _, err := kvcorev1.Dial("ws"+strings.TrimPrefix(server.URL, "http")+query, nil)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { clientSocket.Close() })
		return clientSocket
	}

	acceptLauncherConn := func(conns chan net.Conn) net.Conn {
		var conn net.Conn
		Eventually(conns, streamWait).Should(Receive(&conn))
		DeferCleanup(func() { conn.Close() })
		return conn
	}

	readWebsocket := func(clientSocket *websocket.Conn) string {
		Expect(clientSocket.SetReadDeadline(time.Now().Add(streamWait))).To(Succeed())
		msgType, data, err := clientSocket.ReadMessage()
		Expect(err).ToNot(HaveOccurred())
		Expect(msgType).To(Equal(websocket.BinaryMessage))
		return string(data)
	}

	readConn := func(conn net.Conn, size int) string {
		Expect(conn.SetReadDeadline(time.Now().Add(streamWait))).To(Succeed())
		buf := make([]byte, size)
		_, err := io.ReadFull(conn, buf)
		Expect(err).ToNot(HaveOccurred())
		return string(buf)
	}

	// expectProxied asserts that bytes flow in both directions between the
	// websocket client and the socket virt-launcher is listening on.
	expectProxied := func(clientSocket *websocket.Conn, conn net.Conn) {
		const toGuest = "data-for-the-guest"
		Expect(clientSocket.WriteMessage(websocket.BinaryMessage, []byte(toGuest))).To(Succeed())
		Expect(readConn(conn, len(toGuest))).To(Equal(toGuest))

		const fromGuest = "data-from-the-guest"
		_, err := conn.Write([]byte(fromGuest))
		Expect(err).ToNot(HaveOccurred())
		Expect(readWebsocket(clientSocket)).To(Equal(fromGuest))
	}

	countStopChans := func(lock *sync.Mutex, stopChans map[types.UID]chan struct{}) int {
		lock.Lock()
		defer lock.Unlock()
		return len(stopChans)
	}

	countUSBSlots := func() int {
		handler.usbredirLock.Lock()
		defer handler.usbredirLock.Unlock()
		return len(handler.usbredir[testUID].stopChans)
	}

	reserveUSBSlots := func(slots ...int) {
		handler.usbredirLock.Lock()
		defer handler.usbredirLock.Unlock()
		stopChans := make(map[int]chan struct{})
		for _, slot := range slots {
			stopChans[slot] = make(chan struct{})
		}
		handler.usbredir[testUID] = UsbredirHandlerVMI{stopChans: stopChans}
	}

	get := func(server *httptest.Server, query string) *http.Response {
		response, err := http.Get(server.URL + query)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { response.Body.Close() })
		return response
	}

	DescribeTable("should proxy the websocket to the launcher socket", func(
		socketName string,
		handlerFn func() restful.RouteFunction,
	) {
		conns := listenLauncher(socketName)
		server := startServer(handlerFn())

		clientSocket := dialWebsocket(server, "")

		expectProxied(clientSocket, acceptLauncherConn(conns))
	},
		Entry("serial console", "virt-serial0",
			func() restful.RouteFunction { return handler.SerialHandler }),
		Entry("VNC", "virt-vnc",
			func() restful.RouteFunction { return handler.VNCHandler }),
		Entry("USB redirection", "virt-usbredir-0",
			func() restful.RouteFunction { return handler.USBRedirHandler }),
	)

	DescribeTable("should reject a request for a VMI that is not on this node", func(
		handlerFn func() restful.RouteFunction,
	) {
		Expect(vmiStore.Delete(vmi)).To(Succeed())
		server := startServer(handlerFn())

		Expect(get(server, "").StatusCode).To(Equal(http.StatusNotFound))
	},
		Entry("serial console", func() restful.RouteFunction { return handler.SerialHandler }),
		Entry("VNC", func() restful.RouteFunction { return handler.VNCHandler }),
		Entry("USB redirection", func() restful.RouteFunction { return handler.USBRedirHandler }),
		Entry("VSOCK", func() restful.RouteFunction { return handler.VSOCKHandler }),
	)

	DescribeTable("should reject a request when the launcher socket is missing", func(
		handlerFn func() restful.RouteFunction,
	) {
		// No listener is created, so the socket cannot be resolved
		server := startServer(handlerFn())

		Expect(get(server, "").StatusCode).To(Equal(http.StatusBadRequest))
	},
		Entry("serial console", func() restful.RouteFunction { return handler.SerialHandler }),
		Entry("VNC", func() restful.RouteFunction { return handler.VNCHandler }),
		Entry("USB redirection", func() restful.RouteFunction { return handler.USBRedirHandler }),
	)

	DescribeTable("should fail the request when it is not a websocket handshake", func(
		socketName string,
		handlerFn func() restful.RouteFunction,
	) {
		listenLauncher(socketName)
		server := startServer(handlerFn())

		Expect(get(server, "").StatusCode).To(Equal(http.StatusBadRequest))
	},
		Entry("serial console", "virt-serial0",
			func() restful.RouteFunction { return handler.SerialHandler }),
		Entry("VNC", "virt-vnc",
			func() restful.RouteFunction { return handler.VNCHandler }),
		Entry("USB redirection", "virt-usbredir-0",
			func() restful.RouteFunction { return handler.USBRedirHandler }),
	)

	Describe("SerialHandler", func() {
		It("should track the stream in the stop channel cache and clean it up afterwards", func() {
			conns := listenLauncher("virt-serial0")
			server := startServer(handler.SerialHandler)

			clientSocket := dialWebsocket(server, "")
			expectProxied(clientSocket, acceptLauncherConn(conns))

			Expect(countStopChans(handler.serialLock, handler.serialStopChans)).To(Equal(1))

			Expect(clientSocket.Close()).To(Succeed())
			Eventually(func() int {
				return countStopChans(handler.serialLock, handler.serialStopChans)
			}, streamWait).Should(BeZero())
		})

		It("should terminate a previous session when a new one connects", func() {
			conns := listenLauncher("virt-serial0")
			server := startServer(handler.SerialHandler)

			firstSocket := dialWebsocket(server, "")
			expectProxied(firstSocket, acceptLauncherConn(conns))

			secondSocket := dialWebsocket(server, "")
			expectProxied(secondSocket, acceptLauncherConn(conns))

			Expect(firstSocket.SetReadDeadline(time.Now().Add(streamWait))).To(Succeed())
			_, _, err := firstSocket.ReadMessage()
			Expect(err).To(HaveOccurred())

			Expect(countStopChans(handler.serialLock, handler.serialStopChans)).To(Equal(1))
		})
	})

	Describe("VNCHandler", func() {
		It("should deny a second connection when preserveSession is requested", func() {
			conns := listenLauncher("virt-vnc")
			server := startServer(handler.VNCHandler)

			clientSocket := dialWebsocket(server, "")
			expectProxied(clientSocket, acceptLauncherConn(conns))

			Expect(get(server, "?preserveSession=true").StatusCode).To(Equal(http.StatusServiceUnavailable))
		})

		It("should replace an active session when preserveSession is not requested", func() {
			conns := listenLauncher("virt-vnc")
			server := startServer(handler.VNCHandler)

			firstSocket := dialWebsocket(server, "")
			expectProxied(firstSocket, acceptLauncherConn(conns))

			secondSocket := dialWebsocket(server, "?preserveSession=false")
			expectProxied(secondSocket, acceptLauncherConn(conns))

			Expect(firstSocket.SetReadDeadline(time.Now().Add(streamWait))).To(Succeed())
			_, _, err := firstSocket.ReadMessage()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("USBRedirHandler", func() {
		It("should use the first free slot when earlier ones are taken", func() {
			reserveUSBSlots(0, 1)
			conns := listenLauncher("virt-usbredir-2")
			server := startServer(handler.USBRedirHandler)

			clientSocket := dialWebsocket(server, "")

			expectProxied(clientSocket, acceptLauncherConn(conns))
			Expect(countUSBSlots()).To(Equal(3))
		})

		It("should release the slot once the stream ends", func() {
			conns := listenLauncher("virt-usbredir-0")
			server := startServer(handler.USBRedirHandler)

			clientSocket := dialWebsocket(server, "")
			expectProxied(clientSocket, acceptLauncherConn(conns))
			Expect(countUSBSlots()).To(Equal(1))

			Expect(clientSocket.Close()).To(Succeed())
			Eventually(countUSBSlots, streamWait).Should(BeZero())
		})

		It("should reject the request when all slots are in use", func() {
			slots := make([]int, v1.UsbClientPassthroughMaxNumberOf)
			for i := range slots {
				slots[i] = i
			}
			reserveUSBSlots(slots...)
			server := startServer(handler.USBRedirHandler)

			Expect(get(server, "").StatusCode).To(Equal(http.StatusServiceUnavailable))
		})
	})

	Describe("VSOCKHandler", func() {
		// newVSOCKHandler returns a handler whose dialer hands the stream the
		// given connection instead of a real AF_VSOCK socket.
		newVSOCKHandler := func(guestConn net.Conn, dialErr error) *ConsoleHandler {
			dialer := virtvsock.NewDialer(
				mockIsolationDetector,
				virtvsock.NewServerCache(nil),
				filepath.Join(tmpDir, "no-proc"),
				func(_ int, fn func() error) error { return fn() },
				func(uint32, uint32, *vsock.Config) (*vsock.Conn, error) { return nil, dialErr },
				func(net.Conn) virtvsock.TLSConn { return fakeTLSConn{Conn: guestConn} },
			)
			return NewConsoleHandler(mockIsolationDetector, vmiStore, dialer)
		}

		enableVSOCK := func(cid uint32) {
			Expect(vmiStore.Delete(vmi)).To(Succeed())
			vmi = libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithName(testVMIName),
				libvmi.WithUID(testUID),
				libvmi.WithAutoattachVSOCK(true),
			)
			vmi.Status.VSOCKCID = &cid
			Expect(vmiStore.Add(vmi)).To(Succeed())
		}

		// connPair returns the two ends of a connected unix socket.
		connPair := func() (guestSide, testSide net.Conn) {
			listener, err := net.Listen("unix", filepath.Join(tmpDir, "vsock-stub.sock"))
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() { listener.Close() })

			accepted := make(chan net.Conn, 1)
			go func() {
				conn, err := listener.Accept()
				if err == nil {
					accepted <- conn
				}
			}()

			guestSide, err = net.Dial("unix", listener.Addr().String())
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() { guestSide.Close() })

			Eventually(accepted, streamWait).Should(Receive(&testSide))
			DeferCleanup(func() { testSide.Close() })
			return guestSide, testSide
		}

		It("should proxy the websocket to the VSOCK connection", func() {
			enableVSOCK(3)
			guestSide, testSide := connPair()
			handler = newVSOCKHandler(guestSide, nil)
			server := startServer(handler.VSOCKHandler)

			clientSocket := dialWebsocket(server, "?port=1234&tls=true")

			expectProxied(clientSocket, testSide)
		})

		It("should close the websocket when the VSOCK dial fails", func() {
			enableVSOCK(3)
			handler = newVSOCKHandler(nil, fmt.Errorf("no listener in the guest"))
			server := startServer(handler.VSOCKHandler)

			clientSocket := dialWebsocket(server, "?port=1234&tls=true")

			Expect(clientSocket.SetReadDeadline(time.Now().Add(streamWait))).To(Succeed())
			_, _, err := clientSocket.ReadMessage()
			Expect(err).To(HaveOccurred())
		})

		It("should reject a VMI without VSOCK enabled", func() {
			handler = newVSOCKHandler(nil, nil)
			server := startServer(handler.VSOCKHandler)

			Expect(get(server, "?port=1234").StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("should fail when the VSOCK CID is not assigned yet", func() {
			Expect(vmiStore.Delete(vmi)).To(Succeed())
			vmi = libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithName(testVMIName),
				libvmi.WithUID(testUID),
				libvmi.WithAutoattachVSOCK(true),
			)
			Expect(vmiStore.Add(vmi)).To(Succeed())
			handler = newVSOCKHandler(nil, nil)
			server := startServer(handler.VSOCKHandler)

			Expect(get(server, "?port=1234").StatusCode).To(Equal(http.StatusInternalServerError))
		})

		DescribeTable("should reject an unparsable query parameter", func(query string) {
			enableVSOCK(3)
			handler = newVSOCKHandler(nil, nil)
			server := startServer(handler.VSOCKHandler)

			Expect(get(server, query).StatusCode).To(Equal(http.StatusBadRequest))
		},
			Entry("missing port", ""),
			Entry("non numeric port", "?port=not-a-port"),
			Entry("non boolean tls", "?port=1234&tls=maybe"),
		)
	})
})
