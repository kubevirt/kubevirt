/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
	descSecretLabelsName          = "kube_secret_labels"
	descSecretLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descSecretLabelsDefaultLabels = []string{"namespace", "secret"}

	secretMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_secret_info",
			Type: metric.Gauge,
			Help: "Information about secret.",
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) *metric.Family {
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
			Name: "kube_secret_type",
			Type: metric.Gauge,
			Help: "Type about secret.",
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"type"},
							LabelValues: []string{string(s.Type)},
							Value:       1,
						},
					},
				}
			}),
		},
		{
			Name: descSecretLabelsName,
			Type: metric.Gauge,
			Help: descSecretLabelsHelp,
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
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
			Name: "kube_secret_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				ms := []*metric.Metric{}

				if !s.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(s.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_secret_metadata_resource_version",
			Type: metric.Gauge,
			Help: "Resource version representing a specific version of secret.",
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"resource_version"},
							LabelValues: []string{string(s.ObjectMeta.ResourceVersion)},
							Value:       1,
						},
					},
				}
			}),
		},
	}
)

func wrapSecretFunc(f func(*v1.Secret) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		secret := obj.(*v1.Secret)

		metricFamily := f(secret)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descSecretLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{secret.Namespace, secret.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createSecretListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Secrets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Secrets(ns).Watch(opts)
		},
	}
}
