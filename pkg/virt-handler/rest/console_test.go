/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package rest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/emicklei/go-restful"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	"github.com/libvirt/libvirt-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

var _ = Describe("Console", func() {
	var handler http.Handler

	var mockConn *virtwrap.MockConnection
	var mockDomain *virtwrap.MockVirDomain
	var mockStream *virtwrap.MockStream
	var ctrl *gomock.Controller
	var mockRecorder *record.FakeRecorder
	var server *httptest.Server
	var wsUrl *url.URL
	var serverDone chan bool

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	dial := func(vm string, console string) *websocket.Conn {
		wsUrl.Scheme = "ws"
		wsUrl.Path = "/api/v1/namespaces/" + k8sv1.NamespaceDefault + "/vms/" + vm + "/console"
		wsUrl.RawQuery = "console=" + console
		c, _, err := websocket.DefaultDialer.Dial(wsUrl.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		return c
	}

	get := func(vm string) (*http.Response, error) {
		wsUrl.Scheme = "http"
		wsUrl.Path = "/api/v1/namespaces/" + k8sv1.NamespaceDefault + "/vms/" + vm + "/console"
		return http.DefaultClient.Get(wsUrl.String())
	}

	BeforeEach(func() {
		var err error
		// Set up mocks
		ctrl = gomock.NewController(GinkgoT())
		mockConn = virtwrap.NewMockConnection(ctrl)
		mockDomain = virtwrap.NewMockVirDomain(ctrl)
		mockStream = virtwrap.NewMockStream(ctrl)
		mockRecorder = record.NewFakeRecorder(10)

		// Set up web service
		ws := new(restful.WebService)
		handler = http.Handler(restful.NewContainer().Add(ws))

		// Give us a chance to detect when the request is done. Otherwise we
		// don't know when to check mock invokations
		serverDone = make(chan bool)
		waiter := func(request *restful.Request, response *restful.Response) {
			NewConsoleResource(mockConn).Console(request, response)
			close(serverDone)
		}
		ws.Route(ws.GET("/api/v1/namespaces/{namespace}/vms/{name}/console").To(waiter))
		server = httptest.NewServer(handler)
		wsUrl, err = url.Parse(server.URL)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return 404 if VM does not exist", func() {
		mockConn.EXPECT().LookupDomainByName("default_testvm").Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
		r, err := get("testvm")
		Expect(err).ToNot(HaveOccurred())
		Expect(r.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("should return 500 if domain can't be looked up", func() {
		mockConn.EXPECT().LookupDomainByName("default_testvm").Return(nil, libvirt.Error{Code: libvirt.ERR_INVALID_CONN})
		r, err := get("testvm")
		Expect(err).ToNot(HaveOccurred())
		Expect(r.StatusCode).To(Equal(http.StatusInternalServerError))
	})

	Context("with existing domain", func() {
		BeforeEach(func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
		})

		It("should return 500 if uid of domain can't be looked up", func() {
			mockConn.EXPECT().LookupDomainByName("default_testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetUUIDString().Return("", libvirt.Error{Code: libvirt.ERR_INVALID_CONN})
			r, err := get("testvm")
			Expect(err).ToNot(HaveOccurred())
			Expect(r.StatusCode).To(Equal(http.StatusInternalServerError))
		})
		It("should return 500 if creating a stream fails", func() {
			mockConn.EXPECT().LookupDomainByName("default_testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetUUIDString().Return(string(uuid.NewUUID()), nil)
			mockConn.EXPECT().NewStream(libvirt.StreamFlags(0)).Return(nil, libvirt.Error{Code: libvirt.ERR_INVALID_CONN})
			r, err := get("testvm")
			Expect(err).ToNot(HaveOccurred())
			Expect(r.StatusCode).To(Equal(http.StatusInternalServerError))
		})
		It("should return 500 if opening a console connection fails", func() {
			mockConn.EXPECT().LookupDomainByName("default_testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetUUIDString().Return(string(uuid.NewUUID()), nil)
			mockConn.EXPECT().NewStream(libvirt.StreamFlags(0)).Return(mockStream, nil)
			stream := &libvirt.Stream{}
			mockStream.EXPECT().UnderlyingStream().Return(stream)
			mockStream.EXPECT().Close()
			mockDomain.EXPECT().OpenConsole("", stream, libvirt.DomainConsoleFlags(libvirt.DOMAIN_CONSOLE_FORCE)).Return(libvirt.Error{Code: libvirt.ERR_INVALID_CONN})
			r, err := get("testvm")
			Expect(err).ToNot(HaveOccurred())
			Expect(r.StatusCode).To(Equal(http.StatusInternalServerError))
		})
		It("should return 400 if ws upgrade does not work", func() {
			mockConn.EXPECT().LookupDomainByName("default_testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetUUIDString().Return(string(uuid.NewUUID()), nil)
			mockConn.EXPECT().NewStream(libvirt.StreamFlags(0)).Return(mockStream, nil)
			stream := &libvirt.Stream{}
			mockStream.EXPECT().UnderlyingStream().Return(stream)
			mockStream.EXPECT().Close()
			mockDomain.EXPECT().OpenConsole("", stream, libvirt.DomainConsoleFlags(libvirt.DOMAIN_CONSOLE_FORCE)).Return(nil)
			r, err := get("testvm")
			Expect(err).ToNot(HaveOccurred())
			Expect(r.StatusCode).To(Equal(http.StatusBadRequest))
		})
		It("should proxy websocket traffic", func() {

			var in []byte
			var out []byte
			inBuf := bytes.NewBuffer(in)
			outBuf := bytes.NewBuffer(out)
			outBuf.WriteString("hello client!")
			stream := &fakeStream{in: inBuf, out: outBuf, s: &libvirt.Stream{}}

			mockConn.EXPECT().LookupDomainByName("default_testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetUUIDString().Return(string(uuid.NewUUID()), nil)
			mockConn.EXPECT().NewStream(libvirt.StreamFlags(0)).Return(stream, nil)
			mockDomain.EXPECT().OpenConsole("console0", stream.s, libvirt.DomainConsoleFlags(libvirt.DOMAIN_CONSOLE_FORCE)).Return(nil)

			con := dial("testvm", "console0")
			defer con.Close()
			err := con.WriteMessage(websocket.TextMessage, []byte("hello console!"))
			Expect(err).ToNot(HaveOccurred())

			// FIXME, there is somewhere a buffer or timeout which delays the actual send
			// Eventually(inBuf.String).Should(Equal("hello console!"))

			t, body, err := con.ReadMessage()
			Expect(t).To(Equal(websocket.TextMessage))
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(Equal([]byte("hello client!")))
		})

	})
	AfterEach(func() {
		server.Close()
		<-serverDone
		ctrl.Finish()
	})
})

type fakeStream struct {
	in  *bytes.Buffer
	out *bytes.Buffer
	s   *libvirt.Stream
}

func (s *fakeStream) Write(p []byte) (n int, err error) {
	return s.in.Write(p)
}

func (s *fakeStream) Read(p []byte) (n int, err error) {
	return s.out.Read(p)
}

func (s *fakeStream) Close() (e error) {
	return nil
}

func (s *fakeStream) UnderlyingStream() *libvirt.Stream {
	return s.s
}
