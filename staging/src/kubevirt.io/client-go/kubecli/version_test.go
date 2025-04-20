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

package kubecli

import (
	"fmt"
	"net/http"
	"path"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/version"
)

var _ = Describe("Kubevirt Version Client", func() {
	var server *ghttp.Server
	proxyPath := "/proxy/path"

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
	})

	DescribeTable("should fetch version", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		groupInfo := metav1.APIGroup{
			Name:             ApiGroupName,
			PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: ApiGroupName + "/v1alpha3", Version: "v1alpha3"},
		}

		info := version.Info{GitVersion: "v0.5.1-alpha.1.43+fda30004223b51-clean",
			GitCommit:    "fda30004223b51f9e604276419a2b376652cb5ad",
			GitTreeState: "clear",
			BuildDate:    time.Now().Format("%Y-%m-%dT%H:%M:%SZ"),
			GoVersion:    runtime.Version(),
			Compiler:     runtime.Compiler,
			Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)}

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", path.Join(proxyPath, ApiGroupName)),
				ghttp.RespondWithJSONEncoded(http.StatusOK, groupInfo),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", path.Join(proxyPath, "/apis"+groupInfo.PreferredVersion.GroupVersion+"/version")),
				ghttp.RespondWithJSONEncoded(http.StatusOK, info),
			),
		)

		fetchedVersion, err := client.ServerVersion().Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVersion.Compiler).To(Equal(runtime.Compiler))
		Expect(fetchedVersion.GitTreeState).To(Equal(info.GitTreeState))
		Expect(fetchedVersion.BuildDate).To(Equal(info.BuildDate))
		Expect(fetchedVersion.GoVersion).To(Equal(info.GoVersion))
		Expect(fetchedVersion.Platform).To(Equal(info.Platform))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)
})
