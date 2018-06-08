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

package kvm_monitor

import (
	"fmt"
	"os"

	"k8s.io/client-go/tools/cache"

	"github.com/fsnotify/fsnotify"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

const (
	KVMLabel = "kubevirt.io/kvm"
)

type KVMController struct {
	clientset  kubecli.KubevirtClient
	dpi        *KVMDevicePlugin
	host       string
	vmInformer cache.SharedIndexInformer
}

func NewKVMController(vmInformer cache.SharedIndexInformer, clientset kubecli.KubevirtClient, host string) *KVMController {
	return &KVMController{
		clientset:  clientset,
		dpi:        NewKVMDevicePlugin(),
		host:       host,
		vmInformer: vmInformer,
	}
}

func (c *KVMController) isNodeKVMCapable() bool {
	_, err := os.Stat(KVMPath)
	// Since this is a boolean question, any error means "no"
	return (err == nil)
}

func (c *KVMController) waitForPath(path string, stop chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	defer watcher.Close()

	watcher.Add(path)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op == fsnotify.Create {
				return nil
			}
		case <-stop:
			return fmt.Errorf("shutting down")
		}
	}
}

func (c *KVMController) Run(stop chan struct{}) error {
	logger := log.DefaultLogger()
	logger.Info("Starting KVM device controller")

	if !c.isNodeKVMCapable() {
		logger.Infof("KVM device not found. Waiting.")
		err := c.waitForPath(KVMPath, stop)
		if err != nil {
			logger.Errorf("error waiting for kvm device: %v", err)
			return err
		}
	}

	err := c.dpi.Start(stop)
	if err != nil {
		logger.Errorf("Error starting KVM device plugin: %v", err)
		return err
	}

	// FIXME: need to monitor for changes in the overall
	// number of VM's (and allocate more devices as needed)
	// block until shut down
	<-stop

	logger.Info("Shutting down KVM device controller")
	return nil
}
