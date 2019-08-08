/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package options

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// DefaultNamespaces is the default namespace selector for selecting and filtering across all namespaces.
	DefaultNamespaces = NamespaceList{metav1.NamespaceAll}

	// DefaultCollectors represents the default set of collectors in kube-state-metrics.
	DefaultCollectors = CollectorSet{
		"certificatesigningrequests": struct{}{},
		"configmaps":                 struct{}{},
		"cronjobs":                   struct{}{},
		"daemonsets":                 struct{}{},
		"deployments":                struct{}{},
		"endpoints":                  struct{}{},
		"horizontalpodautoscalers":   struct{}{},
		"ingresses":                  struct{}{},
		"jobs":                       struct{}{},
		"limitranges":                struct{}{},
		"namespaces":                 struct{}{},
		"nodes":                      struct{}{},
		"persistentvolumes":          struct{}{},
		"persistentvolumeclaims":     struct{}{},
		"poddisruptionbudgets":       struct{}{},
		"pods":                       struct{}{},
		"replicasets":                struct{}{},
		"replicationcontrollers":     struct{}{},
		"resourcequotas":             struct{}{},
		"secrets":                    struct{}{},
		"services":                   struct{}{},
		"statefulsets":               struct{}{},
	}
)
