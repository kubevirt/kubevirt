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
	"crypto/tls"

	"k8s.io/apiserver/pkg/registry/rest"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachine/lifecycle"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachine/memorydump"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/volumes"
	subresourcerest "kubevirt.io/kubevirt/pkg/virt-api/rest"
	"kubevirt.io/kubevirt/pkg/virt-api/streaming"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func NewStorageMap(virtClient kubecli.KubevirtClient, consoleServerPort int, tlsConfig *tls.Config, clusterConfig *virtconfig.ClusterConfig) map[string]rest.Storage {
	subresourceApp := subresourcerest.NewSubresourceAPIApp(virtClient, consoleServerPort, tlsConfig, clusterConfig)
	lifecycleHandler := lifecycle.NewHandler(virtClient)
	volumesHandler := volumes.NewHandler(virtClient, clusterConfig)
	memoryDumpHandler := memorydump.NewHandler(virtClient, clusterConfig)
	streamer := streaming.NewStreamer(virtClient, consoleServerPort, tlsConfig)
	return map[string]rest.Storage{
		"virtualmachines":                  NewDummyREST(),
		"virtualmachines/expand-spec":      NewExpandSpecREST(virtClient, clusterConfig),
		"virtualmachines/start":            NewStartREST(lifecycleHandler),
		"virtualmachines/stop":             NewStopREST(lifecycleHandler),
		"virtualmachines/restart":          NewRestartREST(lifecycleHandler),
		"virtualmachines/migrate":          NewMigrateREST(lifecycleHandler),
		"virtualmachines/addvolume":        NewAddVolumeREST(volumesHandler),
		"virtualmachines/removevolume":     NewRemoveVolumeREST(volumesHandler),
		"virtualmachines/memorydump":       NewMemoryDumpREST(memoryDumpHandler),
		"virtualmachines/removememorydump": NewRemoveMemoryDumpREST(memoryDumpHandler),
		"virtualmachines/objectgraph":      NewObjectGraphREST(subresourceApp),
		"virtualmachines/evacuate":         NewEvacuateCancelREST(subresourceApp),
		"virtualmachines/portforward":      NewPortForwardREST(streamer),
	}
}
