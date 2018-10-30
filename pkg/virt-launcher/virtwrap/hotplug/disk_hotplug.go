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
	"io/ioutil"
	"path"
	"strings"

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
	// FormatDeviceName is zero-indexed so the length is already a one-up
	idx := len(deviceMap)
	node := api.FormatDeviceName(VIRTIO_DEVICE_PREFIX, idx)
	deviceMap[target] = node
	return node
}

func addHotpluggedDisk(domainManager virtwrap.DomainManager, domName string, nbdDiskPath string, deviceMap map[string]string) error {
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

	err := domainManager.AttachDisk(domName, disk)
	if err != nil {
		log.Log.Reason(err).Error("Unable to attach disk to domain")
		return err
	}

	return nil
}

func delHotpluggedDisk(domainManager virtwrap.DomainManager, domName string, nbdDiskPath string, deviceMap map[string]string) error {
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

	err := domainManager.DetachDisk(domName, disk)
	if err != nil {
		log.Log.Reason(err).Error("Unable to attach disk to domain")
		return err
	}

	return nil
}

// Watches for domains and monitors each for hotplug events
func WatchHotplugDomains(domainManager virtwrap.DomainManager, baseDir string, stop chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(baseDir)
	if err != nil {
		log.Log.Reason(err).Errorf("Error attempting to watch base path: %s", baseDir)
		return err
	}

	stopChanPerDomain := make(map[string]chan struct{})

	for {
		select {
		case event := <-watcher.Events:
			target := event.Name
			domName := path.Base(target)
			// ignore prefixes starting with "."
			if !strings.HasPrefix(domName, ".") {
				if event.Op == fsnotify.Create {
					deviceMap := map[string]string{}
					stopChanPerDomain[domName] = make(chan struct{})
					fileList, err := ioutil.ReadDir(target)
					if err != nil {
						return err
					}
					for idx, _ := range fileList {
						// These are just placeholders for now.
						deviceMap[fmt.Sprintf("dev_%d", idx)] = ""
					}
					WatchPluggableDisks(domainManager, domName, target, deviceMap, stopChanPerDomain[domName])
				} else if event.Op == fsnotify.Remove {
					close(stopChanPerDomain[domName])
					delete(stopChanPerDomain, domName)
				}
			}
		case <-stop:
			for _, stopChan := range stopChanPerDomain {
				close(stopChan)
			}
			return fmt.Errorf("shutting down pluggable disk watcher")
		}
	}

	//

}

func WatchPluggableDisks(domainManager virtwrap.DomainManager, domName string, hotplugDir string, deviceMap map[string]string, stop chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(hotplugDir)
	if err != nil {
		log.Log.Reason(err).Errorf("Error attempting to watch domain hotplug path: %s", hotplugDir)
		return err
	}

	for {
		select {
		case event := <-watcher.Events:
			target := event.Name
			basePath := path.Base(target)
			if !strings.HasPrefix(basePath, ".") {
				if event.Op == fsnotify.Create {
					err = addHotpluggedDisk(domainManager, domName, target, deviceMap)
					if err != nil {
						log.Log.Reason(err).Errorf("Unable to add NBD disk: %s", target)
						return err
					}
				} else if event.Op == fsnotify.Remove {
					err = delHotpluggedDisk(domainManager, domName, target, deviceMap)
					if err != nil {
						log.Log.Reason(err).Errorf("Unable to add NBD disk: %s", target)
						return err
					}
				}
			}
		case <-stop:
			return fmt.Errorf("shutting down pluggable disk watcher")
		}
	}
}
