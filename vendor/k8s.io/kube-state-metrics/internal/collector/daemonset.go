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
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	descDaemonSetLabelsName          = "kube_daemonset_labels"
	descDaemonSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDaemonSetLabelsDefaultLabels = []string{"namespace", "daemonset"}

	daemonSetMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_daemonset_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				ms := []*metric.Metric{}

				if !d.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(d.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_daemonset_status_current_number_scheduled",
			Type: metric.Gauge,
			Help: "The number of nodes running at least one daemon pod and are supposed to.",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.CurrentNumberScheduled),
						},
					},
				}
			}),
		},
		{
			Name: "kube_daemonset_status_desired_number_scheduled",
			Type: metric.Gauge,
			Help: "The number of nodes that should be running the daemon pod.",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.DesiredNumberScheduled),
						},
					},
				}
			}),
		},
		{
			Name: "kube_daemonset_status_number_available",
			Type: metric.Gauge,
			Help: "The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and available",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.NumberAvailable),
						},
					},
				}
			}),
		},
		{
			Name: "kube_daemonset_status_number_misscheduled",
			Type: metric.Gauge,
			Help: "The number of nodes running a daemon pod but are not supposed to.",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.NumberMisscheduled),
						},
					},
				}
			}),
		},
		{
			Name: "kube_daemonset_status_number_ready",
			Type: metric.Gauge,
			Help: "The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.NumberReady),
						},
					},
				}
			}),
		},
		{
			Name: "kube_daemonset_status_number_unavailable",
			Type: metric.Gauge,
			Help: "The number of nodes that should be running the daemon pod and have none of the daemon pod running and available",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.NumberUnavailable),
						},
					},
				}
			}),
		},
		{
			Name: "kube_daemonset_updated_number_scheduled",
			Type: metric.Gauge,
			Help: "The total number of nodes that are running updated daemon pod",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.UpdatedNumberScheduled),
						},
					},
				}
			}),
		},
		{
			Name: "kube_daemonset_metadata_generation",
			Type: metric.Gauge,
			Help: "Sequence number representing a specific generation of the desired state.",
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.ObjectMeta.Generation),
						},
					},
				}
			}),
		},
		{
			Name: descDaemonSetLabelsName,
			Type: metric.Gauge,
			Help: descDaemonSetLabelsHelp,
			GenerateFunc: wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(d.ObjectMeta.Labels)
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

func wrapDaemonSetFunc(f func(*v1.DaemonSet) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		daemonSet := obj.(*v1.DaemonSet)

		metricFamily := f(daemonSet)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descDaemonSetLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{daemonSet.Namespace, daemonSet.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createDaemonSetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1().DaemonSets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1().DaemonSets(ns).Watch(opts)
		},
	}
}
