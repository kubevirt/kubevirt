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
	"k8s.io/kube-state-metrics/pkg/constant"
	"k8s.io/kube-state-metrics/pkg/metric"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descNodeLabelsName          = "kube_node_labels"
	descNodeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNodeLabelsDefaultLabels = []string{"node"}

	nodeMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_node_info",
			Type: metric.Gauge,
			Help: "Information about a cluster node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys: []string{
								"kernel_version",
								"os_image",
								"container_runtime_version",
								"kubelet_version",
								"kubeproxy_version",
								"provider_id",
							},
							LabelValues: []string{
								n.Status.NodeInfo.KernelVersion,
								n.Status.NodeInfo.OSImage,
								n.Status.NodeInfo.ContainerRuntimeVersion,
								n.Status.NodeInfo.KubeletVersion,
								n.Status.NodeInfo.KubeProxyVersion,
								n.Spec.ProviderID,
							},
							Value: 1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_node_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				if !n.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{

						Value: float64(n.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: descNodeLabelsName,
			Type: metric.Gauge,
			Help: descNodeLabelsHelp,
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(n.Labels)
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
			Name: "kube_node_spec_unschedulable",
			Type: metric.Gauge,
			Help: "Whether a node can schedule new pods.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: boolFloat64(n.Spec.Unschedulable),
						},
					},
				}
			}),
		},
		{
			Name: "kube_node_spec_taint",
			Type: metric.Gauge,
			Help: "The taint of a cluster node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := make([]*metric.Metric, len(n.Spec.Taints))

				for i, taint := range n.Spec.Taints {
					// Taints are applied to repel pods from nodes that do not have a corresponding
					// toleration.  Many node conditions are optionally reflected as taints
					// by the node controller in order to simplify scheduling constraints.
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"key", "value", "effect"},
						LabelValues: []string{taint.Key, taint.Value, string(taint.Effect)},
						Value:       1,
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		// This all-in-one metric family contains all conditions for extensibility.
		// Third party plugin may report customized condition for cluster node
		// (e.g. node-problem-detector), and Kubernetes may add new core
		// conditions in future.
		{
			Name: "kube_node_status_condition",
			Type: metric.Gauge,
			Help: "The condition of a cluster node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := make([]*metric.Metric, len(n.Status.Conditions)*len(conditionStatuses))

				// Collect node conditions and while default to false.
				for i, c := range n.Status.Conditions {
					conditionMetrics := addConditionMetrics(c.Status)

					for j, m := range conditionMetrics {
						metric := m

						metric.LabelKeys = []string{"condition", "status"}
						metric.LabelValues = append([]string{string(c.Type)}, metric.LabelValues...)

						ms[i*len(conditionStatuses)+j] = metric
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_phase",
			Type: metric.Gauge,
			Help: "The phase the node is currently in.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				p := n.Status.Phase

				if p == "" {
					return &metric.Family{
						Metrics: []*metric.Metric{},
					}
				}

				// Set current phase to 1, others to 0 if it is set.
				ms := []*metric.Metric{
					{
						LabelValues: []string{string(v1.NodePending)},
						Value:       boolFloat64(p == v1.NodePending),
					},
					{
						LabelValues: []string{string(v1.NodeRunning)},
						Value:       boolFloat64(p == v1.NodeRunning),
					},
					{
						LabelValues: []string{string(v1.NodeTerminated)},
						Value:       boolFloat64(p == v1.NodeTerminated),
					},
				}

				for _, metric := range ms {
					metric.LabelKeys = []string{"phase"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_capacity",
			Type: metric.Gauge,
			Help: "The capacity for different resources of a node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				capacity := n.Status.Capacity
				for resourceName, val := range capacity {
					switch resourceName {
					case v1.ResourceCPU:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitCore),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					case v1.ResourceStorage:
						fallthrough
					case v1.ResourceEphemeralStorage:
						fallthrough
					case v1.ResourceMemory:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitByte),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					case v1.ResourcePods:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitInteger),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					default:
						if isHugePageResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitByte),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
						if isAttachableVolumeResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitByte),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
						if isExtendedResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitInteger),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
					}
				}

				for _, metric := range ms {
					metric.LabelKeys = []string{"resource", "unit"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_capacity_pods",
			Type: metric.Gauge,
			Help: "The total pod resources of the node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Capacity[v1.ResourcePods]; ok {
					ms = append(ms, &metric.Metric{

						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_capacity_cpu_cores",
			Type: metric.Gauge,
			Help: "The total CPU resources of the node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Capacity[v1.ResourceCPU]; ok {
					ms = append(ms, &metric.Metric{
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_capacity_memory_bytes",
			Type: metric.Gauge,
			Help: "The total memory resources of the node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Capacity[v1.ResourceMemory]; ok {
					ms = append(ms, &metric.Metric{
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_allocatable",
			Type: metric.Gauge,
			Help: "The allocatable for different resources of a node that are available for scheduling.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				allocatable := n.Status.Allocatable

				for resourceName, val := range allocatable {
					switch resourceName {
					case v1.ResourceCPU:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitCore),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					case v1.ResourceStorage:
						fallthrough
					case v1.ResourceEphemeralStorage:
						fallthrough
					case v1.ResourceMemory:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitByte),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					case v1.ResourcePods:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitInteger),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					default:
						if isHugePageResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitByte),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
						if isAttachableVolumeResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitByte),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
						if isExtendedResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitInteger),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
					}
				}

				for _, m := range ms {
					m.LabelKeys = []string{"resource", "unit"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_allocatable_pods",
			Type: metric.Gauge,
			Help: "The pod resources of a node that are available for scheduling.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Allocatable[v1.ResourcePods]; ok {
					ms = append(ms, &metric.Metric{
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_allocatable_cpu_cores",
			Type: metric.Gauge,
			Help: "The CPU resources of a node that are available for scheduling.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Allocatable[v1.ResourceCPU]; ok {
					ms = append(ms, &metric.Metric{
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_node_status_allocatable_memory_bytes",
			Type: metric.Gauge,
			Help: "The memory resources of a node that are available for scheduling.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) *metric.Family {
				ms := []*metric.Metric{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Allocatable[v1.ResourceMemory]; ok {
					ms = append(ms, &metric.Metric{

						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
	}
)

func wrapNodeFunc(f func(*v1.Node) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		node := obj.(*v1.Node)

		metricFamily := f(node)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descNodeLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{node.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createNodeListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Nodes().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Nodes().Watch(opts)
		},
	}
}
