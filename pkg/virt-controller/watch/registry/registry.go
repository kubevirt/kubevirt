/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package registry

import (
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

// RunnableController represents a generic controller that can be started.
type RunnableController interface {
	Run(threadiness int, stopCh <-chan struct{}) error
	Name() string
}

// ControllerInitFunc is the signature for building a storage controller.
type ControllerInitFunc func(ctx *ControllerContext) (RunnableController, error)

// ControllerContext holds ONLY the common shared dependencies.
// Controller-specific dependencies should be passed directly to the constructor.
type ControllerContext struct {
	ClientSet         kubecli.KubevirtClient
	InformerFactory   controller.KubeInformerFactory
	Recorder          record.EventRecorder
	KubevirtNamespace string
	TemplateService   *services.TemplateService
	IngressCache      cache.Store
	RouteCache        cache.Store
}
