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

package virt_api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	authclientv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/tools/clientcmd"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"kubevirt.io/kubevirt/pkg/util"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const namespaceKubevirt = "kubevirt"

var _ = Describe("Virt-api", func() {
	var app virtAPIApp
	var tmpDir string
	var server *ghttp.Server
	var backend *httptest.Server
	var ctrl *gomock.Controller
	var authorizorMock *rest.MockVirtApiAuthorizor

	BeforeEach(func() {
		app = virtAPIApp{namespace: namespaceKubevirt}
		server = ghttp.NewServer()

		backend = httptest.NewServer(nil)
		tmpDir, err := os.MkdirTemp("", "api_tmp_dir")
		Expect(err).ToNot(HaveOccurred())
		app.virtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		app.certsDirectory = tmpDir

		config, err := clientcmd.BuildConfigFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
		app.aggregatorClient = aggregatorclient.NewForConfigOrDie(config)

		authClient, err := authclientv1.NewForConfig(config)
		Expect(err).ToNot(HaveOccurred())
		app.authorizor = rest.NewAuthorizorFromClient(authClient.SubjectAccessReviews())

		ctrl = gomock.NewController(GinkgoT())
		authorizorMock = rest.NewMockVirtApiAuthorizor(ctrl)

		// Reset go-restful
		http.DefaultServeMux = new(http.ServeMux)
		restful.DefaultContainer = restful.NewContainer()
		restful.DefaultContainer.ServeMux = http.DefaultServeMux
	})

	Context("Virt api server", func() {

		It("should return error if extension-apiserver-authentication ConfigMap doesn't exist", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, nil),
				),
			)

			err := app.readRequestHeader()
			Expect(err).To(HaveOccurred())

		})

		It("should fail without requestheader CA", func() {

			configMap := &k8sv1.ConfigMap{}
			configMap.Data = make(map[string]string)
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, configMap),
				),
			)

			err := app.readRequestHeader()
			Expect(err).To(HaveOccurred())
		})

		It("should auto detect correct request headers from cert configmap", func() {
			configMap := &k8sv1.ConfigMap{}
			configMap.Data = make(map[string]string)
			configMap.Data[util.RequestHeaderClientCAFileKey] = "morefakedata"
			configMap.Data["requestheader-username-headers"] = "[\"fakeheader1\"]"
			configMap.Data["requestheader-group-headers"] = "[\"fakeheader2\"]"
			configMap.Data["requestheader-extra-headers-prefix"] = "[\"fakeheader3-\"]"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-system/configmaps/extension-apiserver-authentication"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, configMap),
				),
			)

			err := app.readRequestHeader()
			Expect(err).ToNot(HaveOccurred())
			Expect(app.authorizor.GetUserHeaders()).To(Equal([]string{"X-Remote-User", "fakeheader1"}))
			Expect(app.authorizor.GetGroupHeaders()).To(Equal([]string{"X-Remote-Group", "fakeheader2"}))
			Expect(app.authorizor.GetExtraPrefixHeaders()).To(Equal([]string{"X-Remote-Extra-", "fakeheader3-"}))
		})

		It("should return internal error on authorizor error", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(false, "", errors.New("fake error at authorizor")).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
		})

		It("should return unauthorized error if not allowed", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(false, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

		It("should return ok on root URL", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should have a version endpoint", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/subresources.kubevirt.io/v1alpha3/version")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// TODO: Check version
		})

		It("should return info on the api group version", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/subresources.kubevirt.io/v1alpha3/")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// TODO: Check list
		})

		It("should return info on the API group", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/subresources.kubevirt.io/")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// TODO: Check list
		})

		It("should return API group list on /apis", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/apis/")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// TODO: Check list
		})

		It("should have default values for flags", func() {
			app.AddFlags()
			Expect(app.SwaggerUI).To(Equal("third_party/swagger-ui"))
			Expect(app.SubresourcesOnly).To(BeFalse())
		})

		It("should return OpenAPI v3 discovery endpoint", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/openapi/v3")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should have OpenAPI v3 spec endpoint for subresources.kubevirt.io/v1", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/openapi/v3/apis/subresources.kubevirt.io/v1")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should have OpenAPI v3 spec endpoint for subresources.kubevirt.io/v1alpha3", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()
			resp, err := http.Get(backend.URL + "/openapi/v3/apis/subresources.kubevirt.io/v1alpha3")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should return OpenAPI v3 discovery with hash in serverRelativeURL", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()

			resp, err := http.Get(backend.URL + "/openapi/v3")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())

			var discovery map[string]interface{}
			err = json.Unmarshal(body, &discovery)
			Expect(err).ToNot(HaveOccurred())

			paths, ok := discovery["paths"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "discovery response should have 'paths' field")
			Expect(paths).ToNot(BeEmpty(), "paths should not be empty")

			// SHA-512 produces 64 bytes = 128 hex characters
			hashPattern := regexp.MustCompile(`\?hash=[A-F0-9]{128}$`)

			for gvPath, pathInfo := range paths {
				pathMap, ok := pathInfo.(map[string]interface{})
				Expect(ok).To(BeTrue(), "path info for %s should be a map", gvPath)

				serverRelativeURL, ok := pathMap["serverRelativeURL"].(string)
				Expect(ok).To(BeTrue(), "path %s should have serverRelativeURL", gvPath)

				// Verify the URL contains a hash query parameter
				Expect(serverRelativeURL).To(ContainSubstring("?hash="),
					"serverRelativeURL for %s should contain hash query parameter", gvPath)

				// Verify the hash is a valid SHA-512 uppercase hex (128 characters)
				Expect(hashPattern.MatchString(serverRelativeURL)).To(BeTrue(),
					"serverRelativeURL %s should end with a valid SHA-512 hash (128 uppercase hex chars)", serverRelativeURL)

				// Verify the path is correctly formatted
				Expect(serverRelativeURL).To(HavePrefix("/openapi/v3/"),
					"serverRelativeURL should start with /openapi/v3/")
			}
		})

		It("should return different hashes for different API versions", func() {
			app.authorizor = authorizorMock
			authorizorMock.EXPECT().
				Authorize(gomock.Not(gomock.Nil())).
				Return(true, "", nil).
				AnyTimes()
			app.Compose()

			resp, err := http.Get(backend.URL + "/openapi/v3")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())

			var discovery map[string]interface{}
			err = json.Unmarshal(body, &discovery)
			Expect(err).ToNot(HaveOccurred())

			paths, ok := discovery["paths"].(map[string]interface{})
			Expect(ok).To(BeTrue())

			// Extract hashes from all paths
			hashes := make(map[string]string)
			for gvPath, pathInfo := range paths {
				pathMap := pathInfo.(map[string]interface{})
				serverRelativeURL := pathMap["serverRelativeURL"].(string)

				// Extract hash from URL
				parts := strings.Split(serverRelativeURL, "?hash=")
				if len(parts) == 2 {
					hashes[gvPath] = parts[1]
				}
			}

			// If we have multiple versions, verify they have different hashes
			// (since each version should have its own filtered spec)
			if len(hashes) > 1 {
				hashValues := make([]string, 0, len(hashes))
				for _, hash := range hashes {
					hashValues = append(hashValues, hash)
				}

				// Check that not all hashes are the same
				// (they should be different since each version has different routes)
				allSame := true
				for i := 1; i < len(hashValues); i++ {
					if hashValues[i] != hashValues[0] {
						allSame = false
						break
					}
				}
				Expect(allSame).To(BeFalse(),
					"Different API versions should have different hashes since they have different specs")
			}
		})
	})

	AfterEach(func() {
		backend.Close()
		server.Close()
		os.RemoveAll(tmpDir)
	})
})
