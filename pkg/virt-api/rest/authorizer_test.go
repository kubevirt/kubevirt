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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package rest

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"

	restful "github.com/emicklei/go-restful"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	authorizationclient "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/log"
)

var _ = Describe("VirtualMachineInstance Subresources", func() {
	var server *ghttp.Server
	var req *restful.Request

	log.Log.SetIOWriter(GinkgoWriter)
	fakecert := &x509.Certificate{}

	app := authorizor{}
	BeforeEach(func() {
		req = &restful.Request{}
		req.Request = &http.Request{}
		req.Request.URL = &url.URL{}
		req.Request.Header = make(map[string][]string)
		req.Request.Header[userHeader] = []string{"user"}
		req.Request.Header[groupHeader] = []string{"userGroup"}
		req.Request.Header[userExtraHeaderPrefix+"test"] = []string{"userExtraValue"}
		req.Request.URL.Path = "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi/console"

		server = ghttp.NewServer()
		config, err := clientcmd.BuildConfigFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())

		client, err := authorizationclient.NewForConfig(config)
		Expect(err).ToNot(HaveOccurred())

		app.subjectAccessReview = client.SubjectAccessReviews()
		app.userHeaders = append(app.userHeaders, userHeader)
		app.groupHeaders = append(app.groupHeaders, groupHeader)
		app.userExtraHeaderPrefixes = append(app.userExtraHeaderPrefixes, userExtraHeaderPrefix)
	})

	Context("Subresource api", func() {
		It("should reject unauthenticated user", func(done Done) {
			allowed, reason, err := app.Authorize(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(Equal(false))
			Expect(reason).To(Equal("request is not authenticated"))

			close(done)
		}, 5)

		It("should reject unauthorized user", func(done Done) {

			req.Request.TLS = &tls.ConnectionState{}
			req.Request.TLS.PeerCertificates = append(req.Request.TLS.PeerCertificates, fakecert)

			result, err := app.generateAccessReview(req)
			Expect(err).ToNot(HaveOccurred())
			result.Status.Allowed = false
			result.Status.Reason = "just because"

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/apis/authorization.k8s.io/v1beta1/subjectaccessreviews"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, result),
				),
			)

			allowed, reason, err := app.Authorize(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(Equal(false))
			Expect(reason).To(Equal("just because"))

			close(done)
		}, 5)

		It("should allow authorized user", func(done Done) {

			req.Request.TLS = &tls.ConnectionState{}
			req.Request.TLS.PeerCertificates = append(req.Request.TLS.PeerCertificates, fakecert)

			result, err := app.generateAccessReview(req)
			Expect(err).ToNot(HaveOccurred())
			result.Status.Allowed = true

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/apis/authorization.k8s.io/v1beta1/subjectaccessreviews"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, result),
				),
			)

			allowed, _, err := app.Authorize(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(Equal(true))

			close(done)
		}, 5)

		It("should not allow user if auth check fails", func(done Done) {

			req.Request.TLS = &tls.ConnectionState{}
			req.Request.TLS.PeerCertificates = append(req.Request.TLS.PeerCertificates, fakecert)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/apis/authorization.k8s.io/v1beta1/subjectaccessreviews"),
					ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, nil),
				),
			)

			allowed, _, err := app.Authorize(req)
			Expect(err).To(HaveOccurred())
			Expect(allowed).To(Equal(false))

			close(done)
		}, 5)

		table.DescribeTable("should allow all users for info endpoints", func(path string) {
			req.Request.URL.Path = path
			allowed, _, err := app.Authorize(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(Equal(true))
		},
			table.Entry("root", "/"),
			table.Entry("apis", "/apis"),
			table.Entry("group", "/apis/subresources.kubevirt.io"),
			table.Entry("version", "/apis/subresources.kubevirt.io/version"),
			table.Entry("healthz", "/apis/subresources.kubevirt.io/healthz"),
		)

		table.DescribeTable("should reject all users for unknown endpoint paths", func(path string) {
			req.Request.TLS = &tls.ConnectionState{}
			req.Request.TLS.PeerCertificates = append(req.Request.TLS.PeerCertificates, fakecert)
			req.Request.URL.Path = path
			allowed, _, err := app.Authorize(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(allowed).To(Equal(false))

		},
			table.Entry("random1", "/apis/subresources.kubevirt.io/v1alpha3/madethisup"),
			table.Entry("random2", "/1/2/3/4/5/6/7/8/9/0/1/2/3/4/5/6/7/8/9"),
			table.Entry("no subresource provided", "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
			table.Entry("invalid resource type", "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/madeupresource/testvmi/console"),
		)
	})

	AfterEach(func() {
		server.Close()
	})
})
