/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package profiler

import (
	"net/http"
	"net/http/httptest"

	restful "github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("profiler manager http handler callbacks", func() {
	It("should deny request when ClusterProfiler is not enabled", func() {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				ClusterProfiler: false,
			},
		})

		manager := &ProfileManager{
			clusterConfig: clusterConfig,
		}

		recorder := httptest.NewRecorder()
		response := restful.NewResponse(recorder)
		manager.HandleStartProfiler(nil, response)
		Expect(response.StatusCode()).To(Equal(http.StatusForbidden))

		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
		manager.HandleStopProfiler(nil, response)
		Expect(response.StatusCode()).To(Equal(http.StatusForbidden))

		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
		manager.HandleDumpProfiler(nil, response)
		Expect(response.StatusCode()).To(Equal(http.StatusForbidden))
	})

})
