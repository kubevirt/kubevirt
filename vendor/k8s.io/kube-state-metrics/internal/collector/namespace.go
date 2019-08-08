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
	"k8s.io/kube-state-metrics/pkg/metric"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descNamespaceLabelsName          = "kube_namespace_labels"
	descNamespaceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNamespaceLabelsDefaultLabels = []string{"namespace"}

	descNamespaceAnnotationsName          = "kube_namespace_annotations"
	descNamespaceAnnotationsHelp          = "Kubernetes annotations converted to Prometheus labels."
	descNamespaceAnnotationsDefaultLabels = []string{"namespace"}

	namespaceMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_namespace_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				ms := []*metric.Metric{}
				if !n.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(n.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: descNamespaceLabelsName,
			Type: metric.Gauge,
			Help: descNamespaceLabelsHelp,
			GenerateFunc: wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(n.Labels)
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
			Name: descNamespaceAnnotationsName,
			Type: metric.Gauge,
			Help: descNamespaceAnnotationsHelp,
			GenerateFunc: wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				annotationKeys, annotationValues := kubeAnnotationsToPrometheusAnnotations(n.Annotations)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_namespace_status_phase",
			Type: metric.Gauge,
			Help: "kubernetes namespace status phase.",
			GenerateFunc: wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				ms := []*metric.Metric{
					{
						LabelValues: []string{string(v1.NamespaceActive)},
						Value:       boolFloat64(n.Status.Phase == v1.NamespaceActive),
					},
					{
						LabelValues: []string{string(v1.NamespaceTerminating)},
						Value:       boolFloat64(n.Status.Phase == v1.NamespaceTerminating),
					},
				}

				for _, metric := range ms {
					metric.LabelKeys = []string{"phase"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
	}
)

func wrapNamespaceFunc(f func(*v1.Namespace) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		namespace := obj.(*v1.Namespace)

		metricFamily := f(namespace)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descNamespaceLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{namespace.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createNamespaceListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Namespaces().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Namespaces().Watch(opts)
		},
	}
}
