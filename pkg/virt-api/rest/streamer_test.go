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
	"context"
	goerrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

var _ = Describe("Streamer", func() {
	var (
		streamer               *Streamer
		httpReq                *http.Request
		req                    *restful.Request
		respRecorder           *httptest.ResponseRecorder
		resp                   *restful.Response
		serverConn, serverPipe net.Conn

		testVMI *v1.VirtualMachineInstance
	)

	const (
		testNamespace, testName = "test-namespace", "test-name"
		defaultTestTimeout      = 5 * time.Second
	)

	var (
		fetchVMICalled       bool
		validateVMICalled    bool
		dialCalled           bool
		streamToClientCalled chan struct{}
		streamToServerCalled chan struct{}
		directDialer         *DirectDialer
	)
	BeforeEach(func() {
		testVMI = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "test-vmi"}}
		streamToClientCalled = make(chan struct{}, 1)
		streamToServerCalled = make(chan struct{}, 1)
		serverConn, serverPipe = net.Pipe()

		directDialer = NewDirectDialer(
			// fake fetch vmi function
			func(_, _ string) (*v1.VirtualMachineInstance, *errors.StatusError) {
				fetchVMICalled = true
				return testVMI, nil
			},
			func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
				validateVMICalled = true
				return nil
			},
			mockDialer{
				dialUnderlying: func(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError) {
					dialCalled = true
					return serverConn, nil
				},
			},
		)
		streamer = &Streamer{
			dialer: directDialer,
			streamToClient: func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
				result <- nil
				streamToClientCalled <- struct{}{}
			},
			streamToServer: func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
				result <- nil
				streamToServerCalled <- struct{}{}
			},
		}

		httpReq = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/apis/subresources.kubevirt.io/v1alpha3/namespaces/%s/virtualmachineinstances/%s/ssh/22", testNamespace, testName), nil)
		req = restful.NewRequest(httpReq)
		req.Request.URL.Scheme = "wss"
		respRecorder = httptest.NewRecorder()
		resp = restful.NewResponse(respRecorder)

		fetchVMICalled = false
		validateVMICalled = false
		dialCalled = false
	})
	AfterEach(func() {
		emptyAndCloseChannel(streamToClientCalled)
		emptyAndCloseChannel(streamToServerCalled)
	})
	It("fetches a VirtualMachineInstance", func() {
		streamer.Handle(req, resp)
		Expect(fetchVMICalled).To(BeTrue())
	})
	It("fetches the VMI specified in the request params", func() {
		params := req.PathParameters()
		params[definitions.NamespaceParamName] = testNamespace
		params[definitions.NameParamName] = testName

		streamer.dialer.fetchVMI = func(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
			Expect(namespace).To(Equal(testNamespace))
			Expect(name).To(Equal(testName))
			fetchVMICalled = true
			return nil, nil
		}

		streamer.Handle(req, resp)
		Expect(fetchVMICalled).To(BeTrue())
	})
	It("validates the VMI", func() {
		streamer.Handle(req, resp)
		Expect(validateVMICalled).To(BeTrue())
	})
	It("validates the fetched VMI", func() {
		directDialer.validateVMI = func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
			Expect(vmi).To(Equal(testVMI))
			return nil
		}
		streamer.Handle(req, resp)
	})
	It("does not validate the VMI if it can't be fetched", func() {
		directDialer.fetchVMI = func(_, _ string) (*v1.VirtualMachineInstance, *errors.StatusError) {
			return nil, errors.NewInternalError(goerrors.New("test error"))
		}

		Expect(streamer.Handle(req, resp)).To(HaveOccurred())
		Expect(validateVMICalled).To(BeFalse())
	})
	It("dials the VMI", func() {
		streamer.Handle(req, resp)
		Expect(dialCalled).To(BeTrue())
	})
	It("dials the fetched VMI", func() {
		directDialer.dial = mockDialer{
			dialUnderlying: func(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError) {
				Expect(vmi).To(Equal(testVMI))
				return nil, nil
			},
		}
		streamer.Handle(req, resp)
	})
	It("does not dial when VMI is invalid", func() {
		directDialer.validateVMI = func(_ *v1.VirtualMachineInstance) *errors.StatusError {
			return errors.NewInternalError(goerrors.New("test error"))
		}

		Expect(streamer.Handle(req, resp)).To(HaveOccurred())
		Expect(dialCalled).To(BeFalse())
	})
	It("upgrades the client connection", func() {
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, wsResp, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		Expect(wsResp.StatusCode).To(Equal(101))
		defer ws.Close()
		wg.Wait()
	})
	It("does not attempt the client connection upgrade on a failed dial", func() {
		directDialer.dial = mockDialer{
			dialUnderlying: func(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError) {
				return nil, errors.NewInternalError(goerrors.New("test error"))
			},
		}
		var wg sync.WaitGroup
		wg.Add(1)
		srv, _, wsResp, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).To(HaveOccurred())
		}))
		Expect(err).To(HaveOccurred())
		defer srv.Close()
		Expect(wsResp.StatusCode).To(Equal(http.StatusInternalServerError))
		wg.Wait()
	})
	Context("clientConnectionUpgrade", func() {
		It("does not fail the upgrade on a correct request", func() {
			var wg sync.WaitGroup
			wg.Add(1)
			srv, ws, wsResp, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				defer wg.Done()
				defer GinkgoRecover()
				_, upgradeErr := clientConnectionUpgrade(restful.NewRequest(r), restful.NewResponse(rw))
				Expect(upgradeErr).NotTo(HaveOccurred())
			}))
			Expect(err).NotTo(HaveOccurred())
			defer srv.Close()
			Expect(wsResp.StatusCode).To(Equal(101))
			defer ws.Close()
			wg.Wait()
		})
	})
	It("calls keepAliveClient if set", func() {
		call := make(chan struct{})
		streamer.keepAliveClient = func(ctx context.Context, conn *websocket.Conn, _ func()) {
			defer GinkgoRecover()
			call <- struct{}{}
			Expect(ctx).NotTo(BeNil())
			Expect(conn).NotTo(BeNil())
		}
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(call, defaultTestTimeout).Should(Receive())
		wg.Wait()
	})
	It("returns if the client connection upgrade failed", func() {
		Expect(streamer.Handle(req, resp)).To(HaveOccurred())
		response, err := io.ReadAll(respRecorder.Body)
		Expect(err).To(Not(HaveOccurred()))
		Expect(string(response)).To(ContainSubstring("the client is not using the websocket protocol"))
	})
	It("does start streamToClient with connections", func() {
		streamer.streamToClient = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			defer GinkgoRecover()
			Expect(clientSocket).NotTo(BeNil())
			Expect(serverConn).NotTo(BeNil())
			result <- nil
			streamToClientCalled <- struct{}{}
		}
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(streamToClientCalled, defaultTestTimeout).Should(Receive())
		wg.Wait()
	})
	It("does start streamToServer with connections", func() {
		streamer.streamToServer = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			defer GinkgoRecover()
			Expect(clientSocket).NotTo(BeNil())
			Expect(serverConn).NotTo(BeNil())
			result <- nil
			streamToServerCalled <- struct{}{}
		}
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(streamToServerCalled, defaultTestTimeout).Should(Receive())
		wg.Wait()
	})
	It("closes clientSocket when streamToClient returns", func() {
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		ws.SetReadDeadline(time.Now().Add(defaultTestTimeout))
		_, _, err = ws.ReadMessage()
		Expect(err).To(BeAssignableToTypeOf(&websocket.CloseError{}))
		wg.Wait()
	})
	It("closes serverSocket when streamToServer returns", func() {
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		serverPipe.SetReadDeadline(time.Now().Add(defaultTestTimeout))
		_, err = serverPipe.Read([]byte{})
		Expect(err).To(Equal(io.EOF))
		wg.Wait()
	})
	It("closes clientSocket when keepAliveClient cancels context", func() {
		streamer.streamToClient = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			streamToClientCalled <- struct{}{}
			_, _ = io.Copy(io.Discard, serverConn)
			result <- nil
		}
		streamer.streamToServer = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			streamToServerCalled <- struct{}{}
			_, _ = io.Copy(io.Discard, clientSocket.UnderlyingConn())
			result <- nil
		}
		streamer.keepAliveClient = func(ctx context.Context, conn *websocket.Conn, cancel func()) {
			cancel()
		}
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		ws.SetReadDeadline(time.Now().Add(defaultTestTimeout))
		_, _, err = ws.ReadMessage()
		Expect(err).To(BeAssignableToTypeOf(&websocket.CloseError{}))
		wg.Wait()
	})
	It("calls the streamToClient goroutine", func() {
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(streamToClientCalled, defaultTestTimeout).Should(Receive())
		wg.Wait()
	})
	It("calls the streamToServer goroutine", func() {
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(streamToServerCalled, defaultTestTimeout).Should(Receive())
		wg.Wait()
	})
	It("starts to cleanup after the first stream returns", func() {
		streamer.streamToServer = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			defer GinkgoRecover()
			serverConn.SetReadDeadline(time.Now().Add(defaultTestTimeout))
			_, err := serverConn.Read([]byte{})
			Expect(err).To(Equal(io.ErrClosedPipe))
			result <- nil
			streamToServerCalled <- struct{}{}
		}
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).NotTo(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(streamToClientCalled, defaultTestTimeout).Should(Receive())
		Eventually(streamToServerCalled, defaultTestTimeout).Should(Receive())
	})
	It("returns the first stream result/error if streamToClient terminates", func() {
		testErrStreamEnded := goerrors.New("stream ended")
		streamer.streamToClient = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			result <- testErrStreamEnded
			streamToClientCalled <- struct{}{}
		}
		streamer.streamToServer = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			streamToServerCalled <- struct{}{}
			_, _ = io.Copy(io.Discard, clientSocket.UnderlyingConn())
			result <- nil
		}
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).To(Equal(testErrStreamEnded))
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(streamToClientCalled, defaultTestTimeout).Should(Receive())
		wg.Wait()
	})
	It("returns the first stream result/error if streamToServer terminates", func() {
		testErrStreamEnded := goerrors.New("stream ended")
		streamer.streamToClient = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			streamToClientCalled <- struct{}{}
			_, _ = io.Copy(io.Discard, serverConn)
			result <- nil
		}
		streamer.streamToServer = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			result <- testErrStreamEnded
			streamToServerCalled <- struct{}{}
		}
		var wg sync.WaitGroup
		wg.Add(1)
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer wg.Done()
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).To(Equal(testErrStreamEnded))
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(streamToServerCalled, defaultTestTimeout).Should(Receive())
		wg.Wait()
	})
	It("closes the result channel after both streams have returned", func() {
		// This test is mutually exclusive with the `-race` flag, as there is no other way to check
		// if a write-only channel is closed than to write to it and catch the panic.
		if raceDetectorEnabled {
			Skip("Data Race Detector is enabled")
		}

		var results chan<- streamFuncResult
		streamer.streamToClient = func(clientSocket *websocket.Conn, serverConn net.Conn, result chan<- streamFuncResult) {
			result <- goerrors.New("done")
			results = result
			streamToClientCalled <- struct{}{}
		}
		srv, ws, _, err := testWebsocketDial(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()
			handleErr := streamer.Handle(restful.NewRequest(r), restful.NewResponse(rw))
			Expect(handleErr).To(HaveOccurred())
		}))
		Expect(err).NotTo(HaveOccurred())
		defer srv.Close()
		defer ws.Close()
		Eventually(streamToClientCalled, defaultTestTimeout).Should(Receive())
		Eventually(streamToServerCalled, defaultTestTimeout).Should(Receive())
		Expect(streamFuncResultChannelIsClosed(results, defaultTestTimeout)).To(BeTrue())
	})
})

func streamFuncResultChannelIsClosed(channel chan<- streamFuncResult, timeout time.Duration) bool {
	closed := make(chan bool)
	defer close(closed)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(err)
				closed <- true
			}
		}()
		select {
		case channel <- nil:
			closed <- false
		case <-time.After(timeout):
			closed <- false
		}
	}()

	return <-closed
}

func emptyAndCloseChannel(channel chan struct{}) {
	select {
	case <-channel:
	default:
	}
	close(channel)
}

func testWebsocketDial(handler http.HandlerFunc) (*httptest.Server, *websocket.Conn, *http.Response, error) {
	srv := httptest.NewServer(handler)
	ws, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	return srv, ws, resp, err
}

type mockDialer struct {
	dial           func(vmi *v1.VirtualMachineInstance) (*websocket.Conn, *errors.StatusError)
	dialUnderlying func(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError)
}

func (m mockDialer) Dial(vmi *v1.VirtualMachineInstance) (*websocket.Conn, *errors.StatusError) {
	return m.dial(vmi)
}

func (m mockDialer) DialUnderlying(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError) {
	return m.dialUnderlying(vmi)
}
