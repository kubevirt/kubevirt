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
	descConfigMapLabelsDefaultLabels = []string{"namespace", "configmap"}

	configMapMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_configmap_info",
			Type: metric.Gauge,
			Help: "Information about configmap.",
			GenerateFunc: wrapConfigMapFunc(func(c *v1.ConfigMap) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       1,
					}},
				}
			}),
		},
		{
			Name: "kube_configmap_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapConfigMapFunc(func(c *v1.ConfigMap) *metric.Family {
				ms := []*metric.Metric{}

				if !c.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(c.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_configmap_metadata_resource_version",
			Type: metric.Gauge,
			Help: "Resource version representing a specific version of the configmap.",
			GenerateFunc: wrapConfigMapFunc(func(c *v1.ConfigMap) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"resource_version"},
							LabelValues: []string{string(c.ObjectMeta.ResourceVersion)},
							Value:       1,
						},
					},
				}
			}),
		},
	}
)

func createConfigMapListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().ConfigMaps(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().ConfigMaps(ns).Watch(opts)
		},
	}
}

func wrapConfigMapFunc(f func(*v1.ConfigMap) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		configMap := obj.(*v1.ConfigMap)

		metricFamily := f(configMap)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descConfigMapLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{configMap.Namespace, configMap.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}
