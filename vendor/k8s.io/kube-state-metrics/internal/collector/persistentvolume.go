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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descPersistentVolumeLabelsName          = "kube_persistentvolume_labels"
	descPersistentVolumeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPersistentVolumeLabelsDefaultLabels = []string{"persistentvolume"}

	persistentVolumeMetricFamilies = []metric.FamilyGenerator{
		{
			Name: descPersistentVolumeLabelsName,
			Type: metric.Gauge,
			Help: descPersistentVolumeLabelsHelp,
			GenerateFunc: wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(p.Labels)
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
			Name: "kube_persistentvolume_status_phase",
			Type: metric.Gauge,
			Help: "The phase indicates if a volume is available, bound to a claim, or released by a claim.",
			GenerateFunc: wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
				phase := p.Status.Phase

				if phase == "" {
					return &metric.Family{
						Metrics: []*metric.Metric{},
					}
				}

				// Set current phase to 1, others to 0 if it is set.
				ms := []*metric.Metric{
					{
						LabelValues: []string{string(v1.VolumePending)},
						Value:       boolFloat64(phase == v1.VolumePending),
					},
					{
						LabelValues: []string{string(v1.VolumeAvailable)},
						Value:       boolFloat64(phase == v1.VolumeAvailable),
					},
					{
						LabelValues: []string{string(v1.VolumeBound)},
						Value:       boolFloat64(phase == v1.VolumeBound),
					},
					{
						LabelValues: []string{string(v1.VolumeReleased)},
						Value:       boolFloat64(phase == v1.VolumeReleased),
					},
					{
						LabelValues: []string{string(v1.VolumeFailed)},
						Value:       boolFloat64(phase == v1.VolumeFailed),
					},
				}

				for _, m := range ms {
					m.LabelKeys = []string{"phase"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_persistentvolume_info",
			Type: metric.Gauge,
			Help: "Information about persistentvolume.",
			GenerateFunc: wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"storageclass"},
							LabelValues: []string{p.Spec.StorageClassName},
							Value:       1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_persistentvolume_capacity_bytes",
			Type: metric.Gauge,
			Help: "Persistentvolume capacity in bytes.",
			GenerateFunc: wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
				storage := p.Spec.Capacity[v1.ResourceStorage]
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(storage.Value()),
						},
					},
				}
			}),
		},
	}
)

func wrapPersistentVolumeFunc(f func(*v1.PersistentVolume) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		persistentVolume := obj.(*v1.PersistentVolume)

		metricFamily := f(persistentVolume)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descPersistentVolumeLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{persistentVolume.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createPersistentVolumeListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().PersistentVolumes().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().PersistentVolumes().Watch(opts)
		},
	}
}
