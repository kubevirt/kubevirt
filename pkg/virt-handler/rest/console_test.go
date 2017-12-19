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

	"encoding/xml"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
)

var _ = Describe("Console", func() {
	var handler http.Handler

	var mockConn *cli.MockConnection
	var mockDomain *cli.MockVirDomain
	var mockStream *cli.MockStream
	var ctrl *gomock.Controller
	var mockRecorder *record.FakeRecorder
	var server *httptest.Server
	var wsUrl *url.URL
	var serverDone chan bool

	log.Log.SetIOWriter(GinkgoWriter)

	dial := func(vm string, console string) *websocket.Conn {
		wsUrl.Scheme = "ws"
		wsUrl.Path = "/api/v1/namespaces/" + k8sv1.NamespaceDefault + "/virtualmachines/" + vm + "/console"
		wsUrl.RawQuery = "console=" + console
		c, _, err := websocket.DefaultDialer.Dial(wsUrl.String(), nil)
		Expect(err).ToNot(HaveOccurred())
		return c
	}

	get := func(vm string) (*http.Response, error) {
		wsUrl.Scheme = "http"
		wsUrl.Path = "/api/v1/namespaces/" + k8sv1.NamespaceDefault + "/virtualmachines/" + vm + "/console"
		return http.DefaultClient.Get(wsUrl.String())
	}

	BeforeEach(func() {
		var err error
		// Set up mocks
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		mockStream = cli.NewMockStream(ctrl)
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
		ws.Route(ws.GET("/api/v1/namespaces/{namespace}/virtualmachines/{name}/console").To(waiter))
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
		var specXML string

		BeforeEach(func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			spec := api.NewMinimalDomainSpec("default_testvm")
			spec.Metadata.KubeVirt.UID = uuid.NewUUID()
			data, err := xml.Marshal(spec)
			specXML = string(data)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return 500 if metadata of domain can't be looked up", func() {
			mockConn.EXPECT().LookupDomainByName("default_testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return("", libvirt.Error{Code: libvirt.ERR_INVALID_CONN})

			r, err := get("testvm")
			Expect(err).ToNot(HaveOccurred())
			Expect(r.StatusCode).To(Equal(http.StatusInternalServerError))
		})
		It("should return 500 if creating a stream fails", func() {
			mockConn.EXPECT().LookupDomainByName("default_testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(specXML, nil)
			mockConn.EXPECT().NewStream(libvirt.StreamFlags(0)).Return(nil, libvirt.Error{Code: libvirt.ERR_INVALID_CONN})
			r, err := get("testvm")
			Expect(err).ToNot(HaveOccurred())
			Expect(r.StatusCode).To(Equal(http.StatusInternalServerError))
		})
		It("should return 500 if opening a console connection fails", func() {
			mockConn.EXPECT().LookupDomainByName("default_testvm").Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(specXML, nil)
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
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(specXML, nil)
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
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(specXML, nil)
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
