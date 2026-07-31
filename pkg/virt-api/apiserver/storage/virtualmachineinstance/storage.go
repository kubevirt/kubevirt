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

package virtualmachineinstance

import (
	"crypto/tls"

	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/client-go/kubecli"

	subresourcerest "kubevirt.io/kubevirt/pkg/virt-api/rest"
	"kubevirt.io/kubevirt/pkg/virt-api/streaming"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func NewStorageMap(virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, consoleServerPort int, tlsConfig *tls.Config, clusterConfig *virtconfig.ClusterConfig) map[string]rest.Storage {
	streamer := streaming.NewStreamer(virtClient, consoleServerPort, tlsConfig)
	subresourceApp := subresourcerest.NewSubresourceAPIApp(virtClient, k8sClient, consoleServerPort, tlsConfig, clusterConfig)
	return map[string]rest.Storage{
		"virtualmachineinstances":                     NewDummyREST(),
		"virtualmachineinstances/console":             NewConsoleREST(streamer),
		"virtualmachineinstances/vnc":                 NewVNCREST(streamer, subresourceApp),
		"virtualmachineinstances/usbredir":            NewUSBRedirREST(streamer),
		"virtualmachineinstances/vsock":               NewVSOCKREST(streamer),
		"virtualmachineinstances/portforward":         NewPortForwardREST(streamer),
		"virtualmachineinstances/addvolume":           NewAddVolumeREST(subresourceApp),
		"virtualmachineinstances/removevolume":        NewRemoveVolumeREST(subresourceApp),
		"virtualmachineinstances/freeze":              NewFreezeREST(subresourceApp),
		"virtualmachineinstances/unfreeze":            NewUnfreezeREST(subresourceApp),
		"virtualmachineinstances/pause":               NewPauseREST(subresourceApp),
		"virtualmachineinstances/unpause":             NewUnpauseREST(subresourceApp),
		"virtualmachineinstances/reset":               NewResetREST(subresourceApp),
		"virtualmachineinstances/softreboot":          NewSoftRebootREST(subresourceApp),
		"virtualmachineinstances/guestosinfo":         NewGuestOSInfoREST(subresourceApp),
		"virtualmachineinstances/userlist":            NewUserListREST(subresourceApp),
		"virtualmachineinstances/filesystemlist":      NewFilesystemListREST(subresourceApp),
		"virtualmachineinstances/objectgraph":         NewObjectGraphREST(subresourceApp),
		"virtualmachineinstances/evacuate":            NewEvacuateCancelREST(subresourceApp),
		"virtualmachineinstances/sev":                 NewSEVREST(subresourceApp),
		"virtualmachineinstances/backup":              NewBackupREST(subresourceApp),
		"virtualmachineinstances/redefine-checkpoint": NewRedefineCheckpointREST(subresourceApp),
	}
}
