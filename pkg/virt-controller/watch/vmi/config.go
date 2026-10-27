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

package vmi

import (
	"fmt"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

// ControllerConfig supplies the dependencies and settings for a VMI controller.
type ControllerConfig struct {
	Queue           workqueue.TypedRateLimitingInterface[string]
	Client          kubecli.KubevirtClient
	Recorder        record.EventRecorder
	ClusterConfig   *virtconfig.ClusterConfig
	TemplateService templateService
	Informers       Informers
	Network         NetworkDependencies

	StorageAnnotationsGenerator storageAnnotationsGenerator
	TopologyHinter              topology.Hinter
	VsockAllocator              vsockAllocator

	AdditionalLauncherAnnotationsSync []string
	AdditionalLauncherLabelsSync      []string
}

// Informers supplies the caches and event sources used by a VMI controller.
// Keep syncFunc in sync with the informers declared here.
type Informers struct {
	VMI            cache.SharedIndexInformer
	VM             cache.SharedIndexInformer
	Pod            cache.SharedIndexInformer
	PVC            cache.SharedIndexInformer
	Migration      cache.SharedIndexInformer
	StorageClass   cache.SharedIndexInformer
	DataVolume     cache.SharedIndexInformer
	StorageProfile cache.SharedIndexInformer
	CDI            cache.SharedIndexInformer
	CDIConfig      cache.SharedIndexInformer
	KubeVirt       cache.SharedIndexInformer
}

// syncFunc validates the informers and builds their cache synchronization check.
func (i Informers) syncFunc() (cache.InformerSynced, error) {
	informers := []struct {
		name     string
		informer cache.SharedIndexInformer
	}{
		{"VMI", i.VMI},
		{"VM", i.VM},
		{"Pod", i.Pod},
		{"PVC", i.PVC},
		{"Migration", i.Migration},
		{"StorageClass", i.StorageClass},
		{"DataVolume", i.DataVolume},
		{"StorageProfile", i.StorageProfile},
		{"CDI", i.CDI},
		{"CDIConfig", i.CDIConfig},
		{"KubeVirt", i.KubeVirt},
	}
	syncs := make([]cache.InformerSynced, 0, len(informers))
	for _, entry := range informers {
		if entry.informer == nil {
			return nil, fmt.Errorf("informer %s must not be nil", entry.name)
		}
		syncs = append(syncs, entry.informer.HasSynced)
	}
	return func() bool {
		for _, hasSynced := range syncs {
			if !hasSynced() {
				return false
			}
		}
		return true
	}, nil
}

// NetworkDependencies supplies the network collaborators used by a VMI controller.
type NetworkDependencies struct {
	AnnotationsGenerator annotationsGenerator
	StatusUpdater        statusUpdater
	SpecValidator        specValidator
	MigrationEvaluator   migrationEvaluator
}
