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
	descReplicationControllerLabelsDefaultLabels = []string{"namespace", "replicationcontroller"}

	replicationControllerMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_replicationcontroller_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
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
			Name: "kube_replicationcontroller_status_replicas",
			Type: metric.Gauge,
			Help: "The number of replicas per ReplicationController.",
			GenerateFunc: wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.Replicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_replicationcontroller_status_fully_labeled_replicas",
			Type: metric.Gauge,
			Help: "The number of fully labeled replicas per ReplicationController.",
			GenerateFunc: wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.FullyLabeledReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_replicationcontroller_status_ready_replicas",
			Type: metric.Gauge,
			Help: "The number of ready replicas per ReplicationController.",
			GenerateFunc: wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.ReadyReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_replicationcontroller_status_available_replicas",
			Type: metric.Gauge,
			Help: "The number of available replicas per ReplicationController.",
			GenerateFunc: wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.AvailableReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_replicationcontroller_status_observed_generation",
			Type: metric.Gauge,
			Help: "The generation observed by the ReplicationController controller.",
			GenerateFunc: wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.ObservedGeneration),
						},
					},
				}
			}),
		},
		{
			Name: "kube_replicationcontroller_spec_replicas",
			Type: metric.Gauge,
			Help: "Number of desired pods for a ReplicationController.",
			GenerateFunc: wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				ms := []*metric.Metric{}

				if r.Spec.Replicas != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*r.Spec.Replicas),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_replicationcontroller_metadata_generation",
			Type: metric.Gauge,
			Help: "Sequence number representing a specific generation of the desired state.",
			GenerateFunc: wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.ObjectMeta.Generation),
						},
					},
				}
			}),
		},
	}
)

func wrapReplicationControllerFunc(f func(*v1.ReplicationController) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		replicationController := obj.(*v1.ReplicationController)

		metricFamily := f(replicationController)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descReplicationControllerLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{replicationController.Namespace, replicationController.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createReplicationControllerListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().ReplicationControllers(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().ReplicationControllers(ns).Watch(opts)
		},
	}
}
