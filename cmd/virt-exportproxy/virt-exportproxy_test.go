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

package main

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	testExportService = "virt-export"
	testNamespace     = "default"
	testBackendHost   = testExportService + "." + testNamespace + ".svc"
)

func newExportService(namespace, name, clusterIP string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: clusterIP,
		},
	}
}

var _ = Describe("backendAddr", func() {
	var app *exportProxyApp

	BeforeEach(func() {
		app = &exportProxyApp{
			serviceStore: cache.NewStore(cache.MetaNamespaceKeyFunc),
		}
	})

	It("dials the container port for headless export Services", func() {
		Expect(app.serviceStore.Add(newExportService(testNamespace, testExportService, corev1.ClusterIPNone))).To(Succeed())

		addr, status := app.backendAddr(testNamespace, testExportService)

		Expect(status).To(BeZero())
		Expect(addr).To(Equal(fmt.Sprintf("%s:%d", testBackendHost, storagetypes.ExportServerPort)))
	})

	It("dials the historical Service port for ClusterIP export Services", func() {
		Expect(app.serviceStore.Add(newExportService(testNamespace, testExportService, "10.96.0.10"))).To(Succeed())

		addr, status := app.backendAddr(testNamespace, testExportService)

		Expect(status).To(BeZero())
		Expect(addr).To(Equal(fmt.Sprintf("%s:%d", testBackendHost, storagetypes.ExportClusterIPServicePort)))
	})

	It("returns 503 when the export Service is not in the cache", func() {
		addr, status := app.backendAddr(testNamespace, testExportService)

		Expect(status).To(Equal(http.StatusServiceUnavailable))
		Expect(addr).To(BeEmpty())
	})

	It("returns 503 when the export has no Service name yet", func() {
		addr, status := app.backendAddr(testNamespace, "")

		Expect(status).To(Equal(http.StatusServiceUnavailable))
		Expect(addr).To(BeEmpty())
	})

	It("returns 503 when the Service informer is not initialized", func() {
		app.serviceStore = nil

		addr, status := app.backendAddr(testNamespace, testExportService)

		Expect(status).To(Equal(http.StatusServiceUnavailable))
		Expect(addr).To(BeEmpty())
	})
})
