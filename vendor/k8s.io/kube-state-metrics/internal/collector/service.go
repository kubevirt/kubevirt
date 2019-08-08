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
	v1 "k8s.io/api/core/v1"
	"k8s.io/kube-state-metrics/pkg/metric"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descServiceLabelsName          = "kube_service_labels"
	descServiceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descServiceLabelsDefaultLabels = []string{"namespace", "service"}

	serviceMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_service_info",
			Type: metric.Gauge,
			Help: "Information about service.",
			GenerateFunc: wrapSvcFunc(func(s *v1.Service) *metric.Family {
				m := metric.Metric{
					LabelKeys:   []string{"cluster_ip", "external_name", "load_balancer_ip"},
					LabelValues: []string{s.Spec.ClusterIP, s.Spec.ExternalName, s.Spec.LoadBalancerIP},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		},
		{
			Name: "kube_service_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapSvcFunc(func(s *v1.Service) *metric.Family {
				if !s.CreationTimestamp.IsZero() {
					m := metric.Metric{
						LabelKeys:   nil,
						LabelValues: nil,
						Value:       float64(s.CreationTimestamp.Unix()),
					}
					return &metric.Family{Metrics: []*metric.Metric{&m}}
				}
				return &metric.Family{Metrics: []*metric.Metric{}}
			}),
		},
		{
			Name: "kube_service_spec_type",
			Type: metric.Gauge,
			Help: "Type about service.",
			GenerateFunc: wrapSvcFunc(func(s *v1.Service) *metric.Family {
				m := metric.Metric{

					LabelKeys:   []string{"type"},
					LabelValues: []string{string(s.Spec.Type)},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		},
		{
			Name: descServiceLabelsName,
			Type: metric.Gauge,
			Help: descServiceLabelsHelp,
			GenerateFunc: wrapSvcFunc(func(s *v1.Service) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
				m := metric.Metric{

					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		},
		{
			Name: "kube_service_spec_external_ip",
			Type: metric.Gauge,
			Help: "Service external ips. One series for each ip",
			GenerateFunc: wrapSvcFunc(func(s *v1.Service) *metric.Family {
				if len(s.Spec.ExternalIPs) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{},
					}
				}

				ms := make([]*metric.Metric, len(s.Spec.ExternalIPs))

				for i, externalIP := range s.Spec.ExternalIPs {
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"external_ip"},
						LabelValues: []string{externalIP},
						Value:       1,
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_service_status_load_balancer_ingress",
			Type: metric.Gauge,
			Help: "Service load balancer ingress status",
			GenerateFunc: wrapSvcFunc(func(s *v1.Service) *metric.Family {
				if len(s.Status.LoadBalancer.Ingress) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{},
					}
				}

				ms := make([]*metric.Metric, len(s.Status.LoadBalancer.Ingress))

				for i, ingress := range s.Status.LoadBalancer.Ingress {
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"ip", "hostname"},
						LabelValues: []string{ingress.IP, ingress.Hostname},
						Value:       1,
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
	}
)

func wrapSvcFunc(f func(*v1.Service) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		svc := obj.(*v1.Service)

		metricFamily := f(svc)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descServiceLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{svc.Namespace, svc.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createServiceListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Services(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Services(ns).Watch(opts)
		},
	}
}
