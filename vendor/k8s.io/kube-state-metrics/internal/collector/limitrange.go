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
	descLimitRangeLabelsDefaultLabels = []string{"namespace", "limitrange"}

	limitRangeMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_limitrange",
			Type: metric.Gauge,
			Help: "Information about limit range.",
			GenerateFunc: wrapLimitRangeFunc(func(r *v1.LimitRange) *metric.Family {
				ms := []*metric.Metric{}

				rawLimitRanges := r.Spec.Limits
				for _, rawLimitRange := range rawLimitRanges {
					for resource, min := range rawLimitRange.Min {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "min"},
							Value:       float64(min.MilliValue()) / 1000,
						})
					}

					for resource, max := range rawLimitRange.Max {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "max"},
							Value:       float64(max.MilliValue()) / 1000,
						})
					}

					for resource, df := range rawLimitRange.Default {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "default"},
							Value:       float64(df.MilliValue()) / 1000,
						})
					}

					for resource, dfR := range rawLimitRange.DefaultRequest {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "defaultRequest"},
							Value:       float64(dfR.MilliValue()) / 1000,
						})
					}

					for resource, mLR := range rawLimitRange.MaxLimitRequestRatio {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "maxLimitRequestRatio"},
							Value:       float64(mLR.MilliValue()) / 1000,
						})
					}
				}

				for _, m := range ms {
					m.LabelKeys = []string{"resource", "type", "constraint"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_limitrange_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapLimitRangeFunc(func(r *v1.LimitRange) *metric.Family {
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
	}
)

func wrapLimitRangeFunc(f func(*v1.LimitRange) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		limitRange := obj.(*v1.LimitRange)

		metricFamily := f(limitRange)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descLimitRangeLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{limitRange.Namespace, limitRange.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createLimitRangeListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().LimitRanges(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().LimitRanges(ns).Watch(opts)
		},
	}
}
