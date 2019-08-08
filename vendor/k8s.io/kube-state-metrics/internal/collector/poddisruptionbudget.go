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
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/metric"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
)

var (
	descPodDisruptionBudgetLabelsDefaultLabels = []string{"namespace", "poddisruptionbudget"}

	podDisruptionBudgetMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_poddisruptionbudget_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapPodDisruptionBudgetFunc(func(p *v1beta1.PodDisruptionBudget) *metric.Family {
				ms := []*metric.Metric{}

				if !p.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(p.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_poddisruptionbudget_status_current_healthy",
			Type: metric.Gauge,
			Help: "Current number of healthy pods",
			GenerateFunc: wrapPodDisruptionBudgetFunc(func(p *v1beta1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.CurrentHealthy),
						},
					},
				}
			}),
		},
		{
			Name: "kube_poddisruptionbudget_status_desired_healthy",
			Type: metric.Gauge,
			Help: "Minimum desired number of healthy pods",
			GenerateFunc: wrapPodDisruptionBudgetFunc(func(p *v1beta1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.DesiredHealthy),
						},
					},
				}
			}),
		},
		{
			Name: "kube_poddisruptionbudget_status_pod_disruptions_allowed",
			Type: metric.Gauge,
			Help: "Number of pod disruptions that are currently allowed",
			GenerateFunc: wrapPodDisruptionBudgetFunc(func(p *v1beta1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.PodDisruptionsAllowed),
						},
					},
				}
			}),
		},
		{
			Name: "kube_poddisruptionbudget_status_expected_pods",
			Type: metric.Gauge,
			Help: "Total number of pods counted by this disruption budget",
			GenerateFunc: wrapPodDisruptionBudgetFunc(func(p *v1beta1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.ExpectedPods),
						},
					},
				}
			}),
		},
		{
			Name: "kube_poddisruptionbudget_status_observed_generation",
			Type: metric.Gauge,
			Help: "Most recent generation observed when updating this PDB status",
			GenerateFunc: wrapPodDisruptionBudgetFunc(func(p *v1beta1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.ObservedGeneration),
						},
					},
				}
			}),
		},
	}
)

func wrapPodDisruptionBudgetFunc(f func(*v1beta1.PodDisruptionBudget) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		podDisruptionBudget := obj.(*v1beta1.PodDisruptionBudget)

		metricFamily := f(podDisruptionBudget)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descPodDisruptionBudgetLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{podDisruptionBudget.Namespace, podDisruptionBudget.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createPodDisruptionBudgetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.PolicyV1beta1().PodDisruptionBudgets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.PolicyV1beta1().PodDisruptionBudgets(ns).Watch(opts)
		},
	}
}
