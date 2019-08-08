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

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descStatefulSetLabelsName          = "kube_statefulset_labels"
	descStatefulSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descStatefulSetLabelsDefaultLabels = []string{"namespace", "statefulset"}

	statefulSetMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_statefulset_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
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
			Name: "kube_statefulset_status_replicas",
			Type: metric.Gauge,
			Help: "The number of replicas per StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.Replicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_statefulset_status_replicas_current",
			Type: metric.Gauge,
			Help: "The number of current replicas per StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.CurrentReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_statefulset_status_replicas_ready",
			Type: metric.Gauge,
			Help: "The number of ready replicas per StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.ReadyReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_statefulset_status_replicas_updated",
			Type: metric.Gauge,
			Help: "The number of updated replicas per StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.UpdatedReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_statefulset_status_observed_generation",
			Type: metric.Gauge,
			Help: "The generation observed by the StatefulSet controller.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.ObservedGeneration),
						},
					},
				}
			}),
		},
		{
			Name: "kube_statefulset_replicas",
			Type: metric.Gauge,
			Help: "Number of desired pods for a StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				ms := []*metric.Metric{}

				if s.Spec.Replicas != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*s.Spec.Replicas),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_statefulset_metadata_generation",
			Type: metric.Gauge,
			Help: "Sequence number representing a specific generation of the desired state for the StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.ObjectMeta.Generation),
						},
					},
				}
			}),
		},
		{
			Name: descStatefulSetLabelsName,
			Type: metric.Gauge,
			Help: descStatefulSetLabelsHelp,
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
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
			Name: "kube_statefulset_status_current_revision",
			Type: metric.Gauge,
			Help: "Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"revision"},
							LabelValues: []string{s.Status.CurrentRevision},
							Value:       1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_statefulset_status_update_revision",
			Type: metric.Gauge,
			Help: "Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"revision"},
							LabelValues: []string{s.Status.UpdateRevision},
							Value:       1,
						},
					},
				}
			}),
		},
	}
)

func wrapStatefulSetFunc(f func(*v1.StatefulSet) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		statefulSet := obj.(*v1.StatefulSet)

		metricFamily := f(statefulSet)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descStatefulSetLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{statefulSet.Namespace, statefulSet.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createStatefulSetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1().StatefulSets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1().StatefulSets(ns).Watch(opts)
		},
	}
}
