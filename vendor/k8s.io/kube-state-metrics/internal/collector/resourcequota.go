/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	descResourceQuotaLabelsDefaultLabels = []string{"namespace", "resourcequota"}

	resourceQuotaMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_resourcequota_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapResourceQuotaFunc(func(r *v1.ResourceQuota) *metric.Family {
				ms := []*metric.Metric{}

				if !r.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{

						Value: float64(r.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_resourcequota",
			Type: metric.Gauge,
			Help: "Information about resource quota.",
			GenerateFunc: wrapResourceQuotaFunc(func(r *v1.ResourceQuota) *metric.Family {
				ms := []*metric.Metric{}

				for res, qty := range r.Status.Hard {
					ms = append(ms, &metric.Metric{
						LabelValues: []string{string(res), "hard"},
						Value:       float64(qty.MilliValue()) / 1000,
					})
				}
				for res, qty := range r.Status.Used {
					ms = append(ms, &metric.Metric{
						LabelValues: []string{string(res), "used"},
						Value:       float64(qty.MilliValue()) / 1000,
					})
				}

				for _, m := range ms {
					m.LabelKeys = []string{"resource", "type"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
	}
)

func wrapResourceQuotaFunc(f func(*v1.ResourceQuota) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		resourceQuota := obj.(*v1.ResourceQuota)

		metricFamily := f(resourceQuota)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descResourceQuotaLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{resourceQuota.Namespace, resourceQuota.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createResourceQuotaListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().ResourceQuotas(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().ResourceQuotas(ns).Watch(opts)
		},
	}
}
