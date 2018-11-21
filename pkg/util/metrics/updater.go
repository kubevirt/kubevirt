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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package metrics

import (
	"time"

	"kubevirt.io/kubevirt/pkg/log"
	promvm "kubevirt.io/kubevirt/pkg/monitoring/vms/prometheus"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type Updater struct {
	virtShareDir string
	interval     time.Duration
}

func NewUpdater(virtShareDir string, interval time.Duration) *Updater {
	log.Log.Infof("starting updater: sharedir=%v, interval=%v", virtShareDir, interval)
	return &Updater{
		virtShareDir: virtShareDir,
		interval:     interval,
	}
}

func (u *Updater) Run() {
	log.Log.Infof("running updater: every %v", u.interval)
	ticker := time.NewTicker(u.interval)
	for {
		<-ticker.C
		u.update()
	}
}

func (u *Updater) update() error {
	socketFiles, err := cmdclient.ListAllSockets(u.virtShareDir)
	if err != nil {
		return err
	}
	for _, socketFile := range socketFiles {
		log.Log.V(3).Infof("Getting stats from sock %s", socketFile)
		client, err := cmdclient.GetClient(socketFile)
		if err != nil {
			log.Log.Reason(err).Error("failed to connect to cmd client socket")
			// Ignore failure to connect to client.
			// These are all local connections via unix socket.
			// A failure to connect means there's nothing on the other
			// end listening.
			continue
		}
		defer client.Close()

		err = promvm.Update(client)
		if err != nil {
			log.Log.Reason(err).Error("failed to connect to update stats from socket")
			continue
		}
		log.Log.V(3).Infof("Updated stats from sock %s", socketFile)
	}
	return nil
}
