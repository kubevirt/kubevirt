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

package virtualmachine

import (
	"k8s.io/apiserver/pkg/registry/rest"

	"kubevirt.io/client-go/kubecli"

	subresourcerest "kubevirt.io/kubevirt/pkg/virt-api/rest"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func NewStorageMap(virtClient kubecli.KubevirtClient, clusterConfig *virtconfig.ClusterConfig) map[string]rest.Storage {
	subresourceApp := subresourcerest.NewSubresourceAPIApp(virtClient, 0, nil, clusterConfig)
	return map[string]rest.Storage{
		"virtualmachines":                  NewDummyREST(),
		"virtualmachines/expand-spec":      NewExpandSpecREST(virtClient, clusterConfig),
		"virtualmachines/start":            NewStartREST(subresourceApp),
		"virtualmachines/stop":             NewStopREST(subresourceApp),
		"virtualmachines/restart":          NewRestartREST(subresourceApp),
		"virtualmachines/migrate":          NewMigrateREST(subresourceApp),
		"virtualmachines/addvolume":        NewAddVolumeREST(subresourceApp),
		"virtualmachines/removevolume":     NewRemoveVolumeREST(subresourceApp),
		"virtualmachines/memorydump":       NewMemoryDumpREST(subresourceApp),
		"virtualmachines/removememorydump": NewRemoveMemoryDumpREST(subresourceApp),
		"virtualmachines/objectgraph":      NewObjectGraphREST(subresourceApp),
		"virtualmachines/evacuate":         NewEvacuateCancelREST(subresourceApp),
	}
}
