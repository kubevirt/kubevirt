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

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			// Read the response body
			body, err := io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(body).ToNot(BeEmpty())

			// Parse the body as a APIResourceList object
			var apiResourceList metav1.APIResourceList
			err = json.Unmarshal(body, &apiResourceList)
			Expect(err).ToNot(HaveOccurred())

			// Extract the list of resource names exposed by the API
			exposedApiResources := []string{}
			for _, resource := range apiResourceList.APIResources {
				exposedApiResources = append(exposedApiResources, resource.Name)
			}

			// List of expected API resources
			expectedApiResources := []string{
				"expand-vm-spec",
				"virtualmachineinstances/addvolume",
				"virtualmachineinstances/backup",
				"virtualmachineinstances/console",
				"virtualmachineinstances/evacuate/cancel",
				"virtualmachineinstances/filesystemlist",
				"virtualmachineinstances/freeze",
				"virtualmachineinstances/guestosinfo",
				"virtualmachineinstances/objectgraph",
				"virtualmachineinstances/pause",
				"virtualmachineinstances/portforward",
				"virtualmachineinstances/removevolume",
				"virtualmachineinstances/reset",
				"virtualmachineinstances/sev/fetchcertchain",
				"virtualmachineinstances/sev/injectlaunchsecret",
				"virtualmachineinstances/sev/querylaunchmeasurement",
				"virtualmachineinstances/sev/setupsession",
				"virtualmachineinstances/softreboot",
				"virtualmachineinstances/unfreeze",
				"virtualmachineinstances/unpause",
				"virtualmachineinstances/usbredir",
				"virtualmachineinstances/userlist",
				"virtualmachineinstances/vnc",
				"virtualmachineinstances/vnc/screenshot",
				"virtualmachineinstances/vsock",
				"virtualmachines/addvolume",
				"virtualmachines/evacuate/cancel",
				"virtualmachines/expand-spec",
				"virtualmachines/memorydump",
				"virtualmachines/migrate",
				"virtualmachines/objectgraph",
				"virtualmachines/portforward",
				"virtualmachines/removememorydump",
				"virtualmachines/removevolume",
				"virtualmachines/restart",
				"virtualmachines/start",
				"virtualmachines/stop",
			}

			// Check if the list of expected resources exactly matches the exposed Api resources
			Expect(exposedApiResources).To(ConsistOf(expectedApiResources))
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

	})

	AfterEach(func() {
		backend.Close()
		server.Close()
		os.RemoveAll(tmpDir)
	})
})
