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
	"strconv"

	"k8s.io/kube-state-metrics/pkg/metric"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descReplicaSetLabelsDefaultLabels = []string{"namespace", "replicaset"}
	descReplicaSetLabelsName          = "kube_replicaset_labels"
	descReplicaSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."

	replicaSetMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_replicaset_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) *metric.Family {
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
			Name: "kube_replicaset_status_replicas",
			Type: metric.Gauge,
			Help: "The number of replicas per ReplicaSet.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) *metric.Family {
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
			Name: "kube_replicaset_status_fully_labeled_replicas",
			Type: metric.Gauge,
			Help: "The number of fully labeled replicas per ReplicaSet.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) *metric.Family {
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
			Name: "kube_replicaset_status_ready_replicas",
			Type: metric.Gauge,
			Help: "The number of ready replicas per ReplicaSet.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) *metric.Family {
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
			Name: "kube_replicaset_status_observed_generation",
			Type: metric.Gauge,
			Help: "The generation observed by the ReplicaSet controller.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) *metric.Family {
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
			Name: "kube_replicaset_spec_replicas",
			Type: metric.Gauge,
			Help: "Number of desired pods for a ReplicaSet.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) *metric.Family {
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
			Name: "kube_replicaset_metadata_generation",
			Type: metric.Gauge,
			Help: "Sequence number representing a specific generation of the desired state.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.ObjectMeta.Generation),
						},
					},
				}
			}),
		},
		{
			Name: "kube_replicaset_owner",
			Type: metric.Gauge,
			Help: "Information about the ReplicaSet's owner.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) *metric.Family {
				owners := r.GetOwnerReferences()

				if len(owners) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{
							{
								LabelKeys:   []string{"owner_kind", "owner_name", "owner_is_controller"},
								LabelValues: []string{"<none>", "<none>", "<none>"},
								Value:       1,
							},
						},
					}
				}

				ms := make([]*metric.Metric, len(owners))

				for i, owner := range owners {
					if owner.Controller != nil {
						ms[i] = &metric.Metric{
							LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
						}
					} else {
						ms[i] = &metric.Metric{
							LabelValues: []string{owner.Kind, owner.Name, "false"},
						}
					}
				}

				for _, m := range ms {
					m.LabelKeys = []string{"owner_kind", "owner_name", "owner_is_controller"}
					m.Value = 1
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: descReplicaSetLabelsName,
			Type: metric.Gauge,
			Help: descReplicaSetLabelsHelp,
			GenerateFunc: wrapReplicaSetFunc(func(d *v1beta1.ReplicaSet) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(d.Labels)
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
	}
)

func wrapReplicaSetFunc(f func(*v1beta1.ReplicaSet) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		replicaSet := obj.(*v1beta1.ReplicaSet)

		metricFamily := f(replicaSet)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descReplicaSetLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{replicaSet.Namespace, replicaSet.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createReplicaSetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ExtensionsV1beta1().ReplicaSets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ExtensionsV1beta1().ReplicaSets(ns).Watch(opts)
		},
	}
}
