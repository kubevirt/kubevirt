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

	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/evacuate"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachineinstance/backup"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachineinstance/guestinfo"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachineinstance/lifecycle"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachineinstance/sev"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/volumes"
	subresourcerest "kubevirt.io/kubevirt/pkg/virt-api/rest"
	"kubevirt.io/kubevirt/pkg/virt-api/streaming"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

func NewStorageMap(virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, consoleServerPort int, tlsConfig *tls.Config, clusterConfig *virtconfig.ClusterConfig) map[string]rest.Storage {
	streamer := streaming.NewStreamer(virtClient, consoleServerPort, tlsConfig)
	lifecycleHandler := lifecycle.NewHandler(virtClient, consoleServerPort, tlsConfig)
	guestInfoHandler := guestinfo.NewHandler(virtClient, consoleServerPort, tlsConfig)
	backupHandler := backup.NewHandler(virtClient, consoleServerPort, tlsConfig)
	volumesHandler := volumes.NewHandler(virtClient, clusterConfig)
	evacuateHandler := evacuate.NewHandler(virtClient, clusterConfig)
	sevHandler := sev.NewHandler(virtClient, consoleServerPort, tlsConfig, clusterConfig)
	subresourceApp := subresourcerest.NewSubresourceAPIApp(virtClient, k8sClient, consoleServerPort, tlsConfig, clusterConfig)
	return map[string]rest.Storage{
		"virtualmachineinstances":                     NewDummyREST(),
		"virtualmachineinstances/console":             NewConsoleREST(streamer),
		"virtualmachineinstances/vnc":                 NewVNCREST(streamer),
		"virtualmachineinstances/usbredir":            NewUSBRedirREST(streamer),
		"virtualmachineinstances/vsock":               NewVSOCKREST(streamer),
		"virtualmachineinstances/portforward":         NewPortForwardREST(streamer),
		"virtualmachineinstances/addvolume":           NewAddVolumeREST(volumesHandler),
		"virtualmachineinstances/removevolume":        NewRemoveVolumeREST(volumesHandler),
		"virtualmachineinstances/freeze":              NewFreezeREST(lifecycleHandler),
		"virtualmachineinstances/unfreeze":            NewUnfreezeREST(lifecycleHandler),
		"virtualmachineinstances/pause":               NewPauseREST(lifecycleHandler),
		"virtualmachineinstances/unpause":             NewUnpauseREST(lifecycleHandler),
		"virtualmachineinstances/reset":               NewResetREST(lifecycleHandler),
		"virtualmachineinstances/softreboot":          NewSoftRebootREST(lifecycleHandler),
		"virtualmachineinstances/guestosinfo":         NewGuestOSInfoREST(guestInfoHandler),
		"virtualmachineinstances/userlist":            NewUserListREST(guestInfoHandler),
		"virtualmachineinstances/filesystemlist":      NewFilesystemListREST(guestInfoHandler),
		"virtualmachineinstances/objectgraph":         NewObjectGraphREST(subresourceApp),
		"virtualmachineinstances/evacuate":            NewEvacuateCancelREST(evacuateHandler),
		"virtualmachineinstances/sev":                 NewSEVREST(sevHandler),
		"virtualmachineinstances/backup":              NewBackupREST(backupHandler),
		"virtualmachineinstances/redefine-checkpoint": NewRedefineCheckpointREST(backupHandler),
	}
}
