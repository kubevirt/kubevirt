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

package hotplug

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"

	"github.com/fsnotify/fsnotify"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const VIRTIO_DEVICE_PREFIX = "vd"
const VIRTIO_BUS_TYPE = "virtio"

func isPathWriteable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

func allocateNextDevice(target string, deviceMap map[string]string) string {
	// FormatDeviceName is zero-indexed to the length is already a one-up
	idx := len(deviceMap)
	node := api.FormatDeviceName(VIRTIO_DEVICE_PREFIX, idx)
	deviceMap[target] = node
	return node
}

func addHotpluggedDisk(domainManager virtwrap.DomainManager, nbdDiskPath string, deviceMap map[string]string) error {
	/*virshCmd, err := exec.LookPath("virsh")
	if err != nil {
		log.Log.Reason(err).Error("virsh not found in $PATH")
	}*/

	deviceNode := allocateNextDevice(nbdDiskPath, deviceMap)
	disk := &api.Disk{
		Type:   "network",
		Device: "disk",
		Driver: &api.DiskDriver{
			Name: "qemu",
			Type: "raw",
		},
		Source: api.DiskSource{
			Protocol: "nbd",
			Host: &api.DiskSourceHost{
				Transport: "unix",
				Socket:    nbdDiskPath,
			},
		},
		Target: api.DiskTarget{
			Device: deviceNode,
			Bus:    VIRTIO_BUS_TYPE,
		},
	}

	if !isPathWriteable(nbdDiskPath) {
		disk.ReadOnly = &api.ReadOnly{}
	}

	/*
		data, err := xml.Marshal(disk)
		if err != nil {
			log.Log.Reason(err).Error("Unable to marshal disk data")
			return err
		}*/

	// FIXME: so there's an issue here. golang reponds so fast that the
	// socket isn't completely ready before trying to add the disk
	// a blind sleep is not a great way to fix this.
	time.Sleep(3 * time.Second)

	// FIXME: how can we know which domain to attach to?
	err := domainManager.AttachDisk(disk)
	if err != nil {
		log.Log.Reason(err).Error("Unable to attach disk to domain")
		return err
	}

	/*domainName := "default_vmi-ephemeral"

	f, err := ioutil.TempFile("", fmt.Sprintf("disk-%s-", deviceNode))
	fn := f.Name()
	f.Write([]byte(data))
	f.Close()

	cmd := exec.Command(virshCmd, "attach-device", domainName, fn)
	err = cmd.Run()
	if err != nil {
		log.Log.Reason(err).Error("Unable to attach disk")
		return err
	}*/

	return nil
}

func delHotpluggedDisk(domainManager virtwrap.DomainManager, nbdDiskPath string, deviceMap map[string]string) error {
	//FIXME implement
	return nil
}

// FIXME: in order to be able to remove pluggable disks already present
// at VMI start, the NBD socket directory should contain a stub for each.
// fsnotify watcher can't observe what's not there.
func WatchPluggableDisks(domainManager virtwrap.DomainManager, baseDir string, deviceMap map[string]string, stop chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(baseDir)
	if err != nil {
		log.Log.Reason(err).Errorf("Error attempting to watch path: %s", baseDir)
		return err
	}

	for {
		select {
		case event := <-watcher.Events:
			target := event.Name
			if event.Op == fsnotify.Create {
				err = addHotpluggedDisk(domainManager, target, deviceMap)
				if err != nil {
					log.Log.Reason(err).Errorf("Unable to add NBD disk: %s", target)
					return err
				}
			} else if event.Op == fsnotify.Remove {
				err = delHotpluggedDisk(domainManager, target, deviceMap)
				if err != nil {
					log.Log.Reason(err).Errorf("Unable to add NBD disk: %s", target)
					return err
				}
			}
		case <-stop:
			return fmt.Errorf("shutting down pluggable disk watcher")
		}
	}
}
