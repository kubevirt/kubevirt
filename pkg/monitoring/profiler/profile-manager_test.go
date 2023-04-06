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
 * Copyright 2021 Red Hat, Inc.
 *
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
	It("should deny request when ClusterProfiler feature gate is not enabled", func() {
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				FeatureGates: []string{"MadeUpGate"},
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
