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

package apply

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
)

type fakeAPIServiceClient struct {
	available bool
}

var errFakeAPIServiceClientNotImplemented = errors.New("fake APIService client: not implemented")

func (f *fakeAPIServiceClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*apiregv1.APIService, error) {
	if name != customMetricsAPIServiceName {
		return nil, errFakeAPIServiceClientNotImplemented
	}
	status := apiregv1.ConditionFalse
	if f.available {
		status = apiregv1.ConditionTrue
	}
	return &apiregv1.APIService{
		Status: apiregv1.APIServiceStatus{
			Conditions: []apiregv1.APIServiceCondition{{
				Type:   apiregv1.Available,
				Status: status,
			}},
		},
	}, nil
}

func (f *fakeAPIServiceClient) Create(context.Context, *apiregv1.APIService, metav1.CreateOptions) (*apiregv1.APIService, error) {
	return nil, errFakeAPIServiceClientNotImplemented
}

func (f *fakeAPIServiceClient) Delete(context.Context, string, metav1.DeleteOptions) error {
	return errFakeAPIServiceClientNotImplemented
}

func (f *fakeAPIServiceClient) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*apiregv1.APIService, error) {
	return nil, errFakeAPIServiceClientNotImplemented
}

var _ install.APIServiceInterface = (*fakeAPIServiceClient)(nil)

var _ = Describe("export-proxy HPA metrics detection", func() {
	It("uses the resource profile when custom.metrics.k8s.io is unavailable", func() {
		client := fake.NewSimpleClientset()
		profile := detectExportProxyHPAMetricsProfile(context.Background(), &fakeAPIServiceClient{available: false}, client, "kubevirt")
		Expect(profile).To(Equal(components.ExportProxyHPAMetricsProfileResource))
	})

	It("builds the namespace object metric probe path for HPA", func() {
		Expect(exportProxyNamespaceObjectMetricProbePath("kubevirt")).To(Equal(
			"/apis/custom.metrics.k8s.io/v1beta1/namespaces/kubevirt/metrics/kubevirt_exportproxy_active_transfers_pod_max",
		))
		Expect(exportProxyPodMetricProbePath("kubevirt")).To(Equal(
			"/apis/custom.metrics.k8s.io/v1beta1/namespaces/kubevirt/pods/*/kubevirt_exportproxy_active_transfers",
		))
	})

	It("detects export-proxy custom metrics when both probe paths succeed", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case exportProxyPodMetricProbePath("kubevirt"),
				exportProxyNamespaceObjectMetricProbePath("kubevirt"):
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"kind":"MetricValueList","items":[]}`))
			default:
				Fail("unexpected path: " + r.URL.Path)
			}
		}))
		defer server.Close()

		restClient, err := rest.UnversionedRESTClientFor(&rest.Config{
			Host: server.URL,
			ContentConfig: rest.ContentConfig{
				NegotiatedSerializer: scheme.Codecs,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(exportProxyCustomMetricsAvailable(context.Background(), restClient, "kubevirt")).To(Succeed())
	})

	It("treats probe timeouts as unavailable metrics", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		restClient, err := rest.UnversionedRESTClientFor(&rest.Config{
			Host: server.URL,
			ContentConfig: rest.ContentConfig{
				NegotiatedSerializer: scheme.Codecs,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		Expect(probeCustomMetricPath(ctx, restClient, exportProxyPodMetricProbePath("kubevirt"))).To(HaveOccurred())
	})

	It("treats probe errors as unavailable metrics", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		restClient, err := rest.UnversionedRESTClientFor(&rest.Config{
			Host: server.URL,
			ContentConfig: rest.ContentConfig{
				NegotiatedSerializer: scheme.Codecs,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(probeCustomMetricPath(context.Background(), restClient, exportProxyPodMetricProbePath("kubevirt"))).To(HaveOccurred())
	})
})
