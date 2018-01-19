/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package watchdog

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func NewWatchdogListWatchFromClient(virtShareDir string, watchdogTimeout int) cache.ListerWatcher {

	d := &WatchdogListWatcher{
		fileDir:                  WatchdogFileDirectory(virtShareDir),
		backgroundWatcherStarted: false,
		watchdogTimeout:          watchdogTimeout,
	}
	return d
}

func WatchdogFileDirectory(baseDir string) string {
	return filepath.Join(baseDir, "watchdog-files")
}

func WatchdogFileFromNamespaceName(baseDir string, namespace string, name string) string {
	watchdogFile := namespace + "_" + name
	return filepath.Join(baseDir, "watchdog-files", watchdogFile)
}

func WatchdogFileRemove(baseDir string, vm *v1.VirtualMachine) error {
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	file := WatchdogFileFromNamespaceName(baseDir, namespace, domain)

	return diskutils.RemoveFile(file)
}

func WatchdogFileUpdate(watchdogFile string) error {
	f, err := os.Create(watchdogFile)
	if err != nil {
		return err
	}
	f.Close()

	return nil
}

func WatchdogFileExists(baseDir string, vm *v1.VirtualMachine) (bool, error) {
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	filePath := WatchdogFileFromNamespaceName(baseDir, namespace, domain)
	exists, err := diskutils.FileExists(filePath)
	if err != nil {
		log.Log.Reason(err).Errorf("Error encountered while attempting to verify if watchdog file at path %s exists.", filePath)

		return false, err
	}
	return exists, nil
}

func WatchdogFileIsExpired(timeoutSeconds int, baseDir string, vm *v1.VirtualMachine) (bool, error) {
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	filePath := WatchdogFileFromNamespaceName(baseDir, namespace, domain)

	exists, err := diskutils.FileExists(filePath)
	if err != nil {
		return false, err
	}

	if exists == false {
		return true, nil
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC().Unix()

	return isExpired(now, timeoutSeconds, stat), nil
}

func isExpired(now int64, timeoutSeconds int, stat os.FileInfo) bool {
	mod := stat.ModTime().UTC().Unix()
	diff := now - mod

	if diff > int64(timeoutSeconds) {
		return true
	}
	return false
}

func detectExpiredFiles(timeoutSeconds int, fileDir string) ([]string, error) {
	var expiredFiles []string
	files, err := ioutil.ReadDir(fileDir)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Unix()
	for _, file := range files {
		if isExpired(now, timeoutSeconds, file) == true {
			expiredFiles = append(expiredFiles, file.Name())
		}
	}
	return expiredFiles, nil
}

type WatchdogListWatcher struct {
	lock                     sync.Mutex
	wg                       sync.WaitGroup
	fileDir                  string
	stopChan                 chan struct{}
	eventChan                chan watch.Event
	watchDogTicker           <-chan time.Time
	backgroundWatcherStarted bool
	watchdogTimeout          int
}

func splitFileNamespaceName(fullPath string) (namespace string, domain string, err error) {
	fileName := filepath.Base(fullPath)
	namespaceName := strings.Split(fileName, "_")
	if len(namespaceName) != 2 {
		return "", "", fmt.Errorf("Invalid file path: %s", fullPath)
	}

	namespace = namespaceName[0]
	domain = namespaceName[1]
	return namespace, domain, nil
}

func (d *WatchdogListWatcher) startBackground() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.backgroundWatcherStarted == true {
		return nil
	}

	d.stopChan = make(chan struct{}, 1)
	d.eventChan = make(chan watch.Event, 100)

	tickRate := 1

	if d.watchdogTimeout > 1 {
		tickRate = d.watchdogTimeout / 2
	}
	d.watchDogTicker = time.NewTicker(time.Duration(tickRate) * time.Second).C

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		for {
			select {
			case <-d.stopChan:
				return
			case <-d.watchDogTicker:
				expiredKeys, err := detectExpiredFiles(d.watchdogTimeout, d.fileDir)
				if err != nil {
					log.Log.Reason(err).Error("Invalid content detected during watchdog tick, ignoring and continuing.")
					continue
				}

				for _, key := range expiredKeys {
					namespace, name, err := splitFileNamespaceName(key)
					if err != nil {
						log.Log.Reason(err).Errorf("Invalid key (%s) detected during watchdog tick, ignoring and continuing.", key)
						continue
					}
					d.eventChan <- watch.Event{Type: watch.Modified, Object: api.NewMinimalDomainWithNS(namespace, name)}
				}
			}
		}
	}()

	d.backgroundWatcherStarted = true
	return nil
}

func (d *WatchdogListWatcher) List(options k8sv1.ListOptions) (runtime.Object, error) {
	err := d.startBackground()
	if err != nil {
		return nil, err
	}

	files, err := detectExpiredFiles(d.watchdogTimeout, d.fileDir)
	domainList := &api.DomainList{
		Items: []api.Domain{},
	}
	for _, file := range files {
		namespace, name, err := splitFileNamespaceName(file)
		if err != nil {
			log.Log.Reason(err).Error("Invalid content detected, ignoring and continuing.")
			continue
		}
		domainList.Items = append(domainList.Items, *api.NewMinimalDomainWithNS(namespace, name))

	}
	return domainList, nil
}

func (d *WatchdogListWatcher) Watch(options k8sv1.ListOptions) (watch.Interface, error) {
	return d, nil
}

func (d *WatchdogListWatcher) Stop() {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.backgroundWatcherStarted == false {
		return
	}
	close(d.stopChan)
	d.wg.Wait()
	d.backgroundWatcherStarted = false
}

func (d *WatchdogListWatcher) ResultChan() <-chan watch.Event {
	return d.eventChan
}
