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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

var _ = Describe("Authorizer", func() {
	Describe("KubeVirt Subresources", func() {
		var (
			req       *restful.Request
			allowedFn func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error)
			app       VirtApiAuthorizor
		)

		BeforeEach(func() {
			req = &restful.Request{}
			req.Request = &http.Request{}
			req.Request.URL = &url.URL{}
			req.Request.Header = make(map[string][]string)
			req.Request.Header[userHeader] = []string{"user"}
			req.Request.Header[groupHeader] = []string{"userGroup"}
			req.Request.Header[userExtraHeaderPrefix+"test"] = []string{"userExtraValue"}
			req.Request.Header[userExtraHeaderPrefix+"test%2fencoded"] = []string{"encodedUserExtraValue"}
			req.Request.TLS = &tls.ConnectionState{}
			req.Request.TLS.PeerCertificates = append(req.Request.TLS.PeerCertificates, &x509.Certificate{})

			allowedFn = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
				panic("unexpected call to allowedFn")
			}

			kubeClient := fake.NewSimpleClientset()
			kubeClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				sar, ok := create.GetObject().(*authv1.SubjectAccessReview)
				Expect(ok).To(BeTrue())
				sarOut, err := allowedFn(sar)
				return true, sarOut, err
			})

			app = NewAuthorizorFromClient(kubeClient.AuthorizationV1().SubjectAccessReviews())
		})

		Context("Subresource api with namespaced resource", func() {
			Context("with namespaced resource", func() {
				allowed := func(allowed bool) func(review *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
					return func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
						Expect(sar.Spec.User).To(Equal("user"))
						Expect(sar.Spec.Groups).To(Equal([]string{"userGroup"}))
						Expect(sar.Spec.Extra).To(HaveKeyWithValue("test", authv1.ExtraValue{"userExtraValue"}))
						Expect(sar.Spec.Extra).To(HaveKeyWithValue("test/encoded", authv1.ExtraValue{"encodedUserExtraValue"}))
						Expect(sar.Spec.NonResourceAttributes).To(BeNil())
						Expect(sar.Spec.ResourceAttributes).ToNot(BeNil())
						Expect(sar.Spec.ResourceAttributes.Namespace).To(Equal("default"))
						Expect(sar.Spec.ResourceAttributes.Verb).To(Equal("get"))
						Expect(sar.Spec.ResourceAttributes.Group).To(Equal("subresources.kubevirt.io"))
						Expect(sar.Spec.ResourceAttributes.Version).To(Equal("v1alpha3"))
						Expect(sar.Spec.ResourceAttributes.Resource).To(Equal("virtualmachineinstances"))
						Expect(sar.Spec.ResourceAttributes.Subresource).To(Equal("console"))
						Expect(sar.Spec.ResourceAttributes.Name).To(Equal("testvmi"))
						sar.Status.Allowed = allowed
						sar.Status.Reason = "just because"
						return sar, nil
					}
				}

				BeforeEach(func() {
					req.Request.Method = http.MethodGet
					req.Request.URL.Path = "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi/console"
				})

				It("should reject unauthenticated user", func() {
					req.Request.TLS = nil
					result, reason, err := app.Authorize(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(BeFalse())
					Expect(reason).To(Equal("request is not authenticated"))
				})

				It("should reject if auth check fails", func() {
					allowedFn = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
						return nil, errors.New("internal error")
					}

					result, _, err := app.Authorize(req)
					Expect(err).To(HaveOccurred())
					Expect(result).To(BeFalse())
				})

				It("should reject unauthorized user", func() {
					allowedFn = allowed(false)
					result, reason, err := app.Authorize(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(BeFalse())
					Expect(reason).To(Equal("just because"))
				})

				It("should allow authorized user", func() {
					allowedFn = allowed(true)
					result, _, err := app.Authorize(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(BeTrue())
				})
			})

			Context("with namespaced base resource", func() {
				allowed := func(allowed bool) func(review *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
					return func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
						Expect(sar.Spec.NonResourceAttributes).To(BeNil())
						Expect(sar.Spec.ResourceAttributes).ToNot(BeNil())
						Expect(sar.Spec.ResourceAttributes.Namespace).To(Equal("default"))
						Expect(sar.Spec.ResourceAttributes.Verb).To(Equal("update"))
						Expect(sar.Spec.ResourceAttributes.Group).To(Equal("subresources.kubevirt.io"))
						Expect(sar.Spec.ResourceAttributes.Version).To(Equal("v1alpha3"))
						Expect(sar.Spec.ResourceAttributes.Resource).To(Equal("expand-vm-spec"))
						sar.Status.Allowed = allowed
						sar.Status.Reason = "just because"
						return sar, nil
					}
				}

				BeforeEach(func() {
					req.Request.Method = http.MethodPut
					req.Request.URL.Path = "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/expand-vm-spec"
				})

				It("should reject unauthenticated user", func() {
					req.Request.TLS = nil

					result, reason, err := app.Authorize(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(BeFalse())
					Expect(reason).To(Equal("request is not authenticated"))
				})

				It("should reject if auth check fails", func() {
					allowedFn = func(sar *authv1.SubjectAccessReview) (*authv1.SubjectAccessReview, error) {
						return nil, errors.New("internal error")
					}

					result, _, err := app.Authorize(req)
					Expect(err).To(HaveOccurred())
					Expect(result).To(BeFalse())
				})

				It("should reject unauthorized user", func() {
					allowedFn = allowed(false)
					result, reason, err := app.Authorize(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(BeFalse())
					Expect(reason).To(Equal("just because"))
				})

				It("should allow authorized user", func() {
					allowedFn = allowed(true)
					result, _, err := app.Authorize(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).To(BeTrue())
				})

			})

			DescribeTable("should allow all users for info endpoints", func(path string) {
				req.Request.TLS = nil
				req.Request.URL.Path = path
				result, _, err := app.Authorize(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeTrue())
			},
				// Root resources
				Entry("root", "/"),
				Entry("apis", "/apis"),
				Entry("healthz", "/healthz"),
				Entry("openapi", "/openapi/v2"),
				Entry("start profiler", "/start-profiler"),
				Entry("stop profiler", "/stop-profiler"),
				Entry("dump profiler", "/dump-profiler"),
				// Subresources v1
				Entry("subresource v1 groupversion", "/apis/subresources.kubevirt.io/v1"),
				Entry("subresource v1 version", "/apis/subresources.kubevirt.io/v1/version"),
				Entry("subresource v1 guestfs", "/apis/subresources.kubevirt.io/v1/guestfs"),
				Entry("subresource v1 healthz", "/apis/subresources.kubevirt.io/v1/healthz"),
				Entry("subresource v1 start profiler", "/apis/subresources.kubevirt.io/v1/start-cluster-profiler"),
				Entry("subresource v1 stop profiler", "/apis/subresources.kubevirt.io/v1/stop-cluster-profiler"),
				Entry("subresource v1 dump profiler", "/apis/subresources.kubevirt.io/v1/dump-cluster-profiler"),
				// Subresource v1alpha3
				Entry("subresource v1alpha3 groupversion", "/apis/subresources.kubevirt.io/v1alpha3"),
				Entry("subresource v1alpha3 version", "/apis/subresources.kubevirt.io/v1alpha3/version"),
				Entry("subresource v1alpha3 guestfs", "/apis/subresources.kubevirt.io/v1alpha3/guestfs"),
				Entry("subresource v1alpha3 healthz", "/apis/subresources.kubevirt.io/v1alpha3/healthz"),
				Entry("subresource v1alpha3 start profiler", "/apis/subresources.kubevirt.io/v1alpha3/start-cluster-profiler"),
				Entry("subresource v1alpha3 stop profiler", "/apis/subresources.kubevirt.io/v1alpha3/stop-cluster-profiler"),
				Entry("subresource v1alpha3 dump profiler", "/apis/subresources.kubevirt.io/v1alpha3/dump-cluster-profiler"),
			)

			DescribeTable("should reject all users for unknown endpoint paths", func(path string) {
				req.Request.URL.Path = path
				result, _, err := app.Authorize(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeFalse())

			},
				Entry("random1", "/apis/subresources.kubevirt.io/v1alpha3/madethisup"),
				Entry("random2", "/1/2/3/4/5/6/7/8/9/0/1/2/3/4/5/6/7/8/9"),
				Entry("no subresource provided", "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi"),
				Entry("invalid resource type", "/apis/subresources.kubevirt.io/v1alpha3/namespaces/default/madeupresource/testvmi/console"),
				Entry("unknown namespaced resource endpoint", "/apis/subresources.kubevirt.io/v1/namespaces/default/madethisup/testvmi/console"),
				Entry("unknown namespaced base resource endpoint", "/apis/subresources.kubevirt.io/v1/namespaces/default/madethisup"),
			)
		})
	})

	DescribeTable("should map verbs", func(httpVerb string, resourceName string, expectedRbacVerb string) {
		Expect(mapHttpVerbToRbacVerb(httpVerb, resourceName)).To(Equal(expectedRbacVerb))
	},
		Entry("http post to create", http.MethodPost, "", "create"),
		Entry("http get with resource to get", http.MethodGet, "foo", "get"),
		Entry("http get without resource to list", http.MethodGet, "", "list"),
		Entry("http put to update", http.MethodPut, "", "update"),
		Entry("http patch to patch", http.MethodPatch, "", "patch"),
		Entry("http delete with reource to delete", http.MethodDelete, "foo", "delete"),
		Entry("http delete without resource to deletecollection", http.MethodDelete, "", "deletecollection"),
	)
})
