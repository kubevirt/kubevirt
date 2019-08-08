/*
Copyright 2017 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collector

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/kube-state-metrics/pkg/metric"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descEndpointLabelsName          = "kube_endpoint_labels"
	descEndpointLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descEndpointLabelsDefaultLabels = []string{"namespace", "endpoint"}

	endpointMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_endpoint_info",
			Type: metric.Gauge,
			Help: "Information about endpoint.",
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_endpoint_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				ms := []*metric.Metric{}

				if !e.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{

						Value: float64(e.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: descEndpointLabelsName,
			Type: metric.Gauge,
			Help: descEndpointLabelsHelp,
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(e.Labels)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_endpoint_address_available",
			Type: metric.Gauge,
			Help: "Number of addresses available in endpoint.",
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				var available int
				for _, s := range e.Subsets {
					available += len(s.Addresses) * len(s.Ports)
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(available),
						},
					},
				}
			}),
		},
		{
			Name: "kube_endpoint_address_not_ready",
			Type: metric.Gauge,
			Help: "Number of addresses not ready in endpoint",
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				var notReady int
				for _, s := range e.Subsets {
					notReady += len(s.NotReadyAddresses) * len(s.Ports)
				}
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(notReady),
						},
					},
				}
			}),
		},
	}
)

func wrapEndpointFunc(f func(*v1.Endpoints) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		endpoint := obj.(*v1.Endpoints)

		metricFamily := f(endpoint)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descEndpointLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{endpoint.Namespace, endpoint.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createEndpointsListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Endpoints(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Endpoints(ns).Watch(opts)
		},
	}
}
