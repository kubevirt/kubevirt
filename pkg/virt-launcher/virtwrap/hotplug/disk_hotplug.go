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

func lookupDomain(domainManager virtwrap.DomainManager) (string, error) {
	domList, err := domainManager.ListAllDomains()
	if err != nil {
		return "", err
	}
	if len(domList) == 0 {
		return "", fmt.Errorf("no active domains")
	}
	domain := domList[0]
	domName := fmt.Sprintf("%s_%s", domain.ObjectMeta.Namespace, domain.ObjectMeta.GetName())
	return domName, nil
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
	deviceNode, ok := deviceMap[nbdDiskPath]
	if !ok {
		return fmt.Errorf("device node for '%s' not found", nbdDiskPath)
	}
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
func WatchHotplugDir(domainManager virtwrap.DomainManager, baseDir string, stop chan struct{}) error {
	deviceMap := map[string]string{}
	// FIXME: we need to learn from the domain how many pluggable devices are present
	limit := 1
	for i := 0; i < limit; i++ {
		deviceMap[fmt.Sprintf("placeholder_%d", i)] = ""
	}
	return WatchPluggableDisks(domainManager, baseDir, deviceMap, stop)
}

func WatchPluggableDisks(domainManager virtwrap.DomainManager, hotplugDir string, deviceMap map[string]string, stop chan struct{}) error {
	domName := ""

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
			if domName == "" {
				domName, err = lookupDomain(domainManager)
				if err != nil {
					return err
				}
			}
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
