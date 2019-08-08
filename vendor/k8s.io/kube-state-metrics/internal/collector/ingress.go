/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

	"k8s.io/api/extensions/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descIngressLabelsName          = "kube_ingress_labels"
	descIngressLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descIngressLabelsDefaultLabels = []string{"namespace", "ingress"}

	ingressMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_ingress_info",
			Type: metric.Gauge,
			Help: "Information about ingress.",
			GenerateFunc: wrapIngressFunc(func(s *v1beta1.Ingress) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					}}
			}),
		},
		{
			Name: descIngressLabelsName,
			Type: metric.Gauge,
			Help: descIngressLabelsHelp,
			GenerateFunc: wrapIngressFunc(func(i *v1beta1.Ingress) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(i.Labels)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					}}

			}),
		},
		{
			Name: "kube_ingress_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapIngressFunc(func(i *v1beta1.Ingress) *metric.Family {
				ms := []*metric.Metric{}

				if !i.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(i.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_ingress_metadata_resource_version",
			Type: metric.Gauge,
			Help: "Resource version representing a specific version of ingress.",
			GenerateFunc: wrapIngressFunc(func(i *v1beta1.Ingress) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"resource_version"},
							LabelValues: []string{string(i.ObjectMeta.ResourceVersion)},
							Value:       1,
						},
					}}
			}),
		},
	}
)

func wrapIngressFunc(f func(*v1beta1.Ingress) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		ingress := obj.(*v1beta1.Ingress)

		metricFamily := f(ingress)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descIngressLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{ingress.Namespace, ingress.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createIngressListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ExtensionsV1beta1().Ingresses(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ExtensionsV1beta1().Ingresses(ns).Watch(opts)
		},
	}
}
