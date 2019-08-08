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
	"k8s.io/kube-state-metrics/pkg/metric"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descDeploymentLabelsName          = "kube_deployment_labels"
	descDeploymentLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDeploymentLabelsDefaultLabels = []string{"namespace", "deployment"}

	deploymentMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_deployment_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				ms := []*metric.Metric{}

				if !d.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(d.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_deployment_status_replicas",
			Type: metric.Gauge,
			Help: "The number of replicas per deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.Replicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_status_replicas_available",
			Type: metric.Gauge,
			Help: "The number of available replicas per deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.AvailableReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_status_replicas_unavailable",
			Type: metric.Gauge,
			Help: "The number of unavailable replicas per deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.UnavailableReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_status_replicas_updated",
			Type: metric.Gauge,
			Help: "The number of updated replicas per deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.UpdatedReplicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_status_observed_generation",
			Type: metric.Gauge,
			Help: "The generation observed by the deployment controller.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.ObservedGeneration),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_spec_replicas",
			Type: metric.Gauge,
			Help: "Number of desired pods for a deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(*d.Spec.Replicas),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_spec_paused",
			Type: metric.Gauge,
			Help: "Whether the deployment is paused and will not be processed by the deployment controller.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: boolFloat64(d.Spec.Paused),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_spec_strategy_rollingupdate_max_unavailable",
			Type: metric.Gauge,
			Help: "Maximum number of unavailable replicas during a rolling update of a deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				if d.Spec.Strategy.RollingUpdate == nil {
					return &metric.Family{}
				}

				maxUnavailable, err := intstr.GetValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxUnavailable, int(*d.Spec.Replicas), true)
				if err != nil {
					panic(err)
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(maxUnavailable),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_spec_strategy_rollingupdate_max_surge",
			Type: metric.Gauge,
			Help: "Maximum number of replicas that can be scheduled above the desired number of replicas during a rolling update of a deployment.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				if d.Spec.Strategy.RollingUpdate == nil {
					return &metric.Family{}
				}

				maxSurge, err := intstr.GetValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxSurge, int(*d.Spec.Replicas), true)
				if err != nil {
					panic(err)
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(maxSurge),
						},
					},
				}
			}),
		},
		{
			Name: "kube_deployment_metadata_generation",
			Type: metric.Gauge,
			Help: "Sequence number representing a specific generation of the desired state.",
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.ObjectMeta.Generation),
						},
					},
				}
			}),
		},
		{
			Name: descDeploymentLabelsName,
			Type: metric.Gauge,
			Help: descDeploymentLabelsHelp,
			GenerateFunc: wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
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

func wrapDeploymentFunc(f func(*v1.Deployment) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		deployment := obj.(*v1.Deployment)

		metricFamily := f(deployment)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descDeploymentLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{deployment.Namespace, deployment.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createDeploymentListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1().Deployments(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1().Deployments(ns).Watch(opts)
		},
	}
}
