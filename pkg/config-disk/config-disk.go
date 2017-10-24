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

package configdisk

import (
	"errors"
	"fmt"
	"sync"

	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/precond"
)

type ConfigDiskClient interface {
	Define(vm *v1.VirtualMachine) (bool, error)
	Undefine(vm *v1.VirtualMachine) error
	UndefineUnseen(indexer cache.Store) error
}

type configDiskClient struct {
	lock      sync.Mutex
	jobs      map[string]chan string
	clientset kubecli.KubevirtClient
}

func NewConfigDiskClient(clientset kubecli.KubevirtClient) ConfigDiskClient {
	return &configDiskClient{
		clientset: clientset,
		jobs:      make(map[string]chan string),
	}
}

func createKey(vm *v1.VirtualMachine) string {
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	return fmt.Sprintf("%s/%s", namespace, domain)
}

func (c *configDiskClient) Define(vm *v1.VirtualMachine) (bool, error) {
	pending := false

	cloudInitSpec := cloudinit.GetCloudInitSpec(vm)
	if cloudInitSpec == nil {
		return false, nil
	}
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	c.lock.Lock()
	defer c.lock.Unlock()

	jobKey := createKey(vm)

	// FIXME support re-creating the disks if the VM is down and the data changed
	v, ok := c.jobs[jobKey]
	if ok == false {
		v = make(chan string, 1)

		go func() {
			cloudinit.ApplyMetadata(vm)
			err := cloudinit.ResolveSecrets(cloudInitSpec, namespace, c.clientset)
			if err != nil {
				v <- fmt.Sprintf("config-disk failure: %v", err)
				return
			}

			err = cloudinit.GenerateLocalData(domain, namespace, cloudInitSpec)
			if err != nil {
				v <- fmt.Sprintf("config-disk failure: %v", err)
				return
			}
			v <- "success"
		}()

		c.jobs[jobKey] = v
		pending = true
	} else {
		select {
		case response := <-v:
			if response != "success" {
				err := errors.New(response)
				if err != nil {
					return pending, err
				}
			}
			delete(c.jobs, jobKey)
		default:
			pending = true
		}
	}

	return pending, nil
}

func (c *configDiskClient) Undefine(vm *v1.VirtualMachine) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	jobKey := createKey(vm)
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	delete(c.jobs, jobKey)

	return cloudinit.RemoveLocalData(domain, namespace)
}

func (c *configDiskClient) UndefineUnseen(indexer cache.Store) error {
	vms, err := cloudinit.ListVmWithLocalData()
	if err != nil {
		return err
	}

	for _, vm := range vms {
		namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
		domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

		cleanup := false

		key, err := cache.MetaNamespaceKeyFunc(vm)
		if err != nil {
			return err
		}

		obj, exists, _ := indexer.GetByKey(key)
		if exists == false {
			cleanup = true
		} else {
			vm := obj.(*v1.VirtualMachine)
			if vm.IsFinal() {
				cleanup = true
			}
		}
		if cleanup {
			err := cloudinit.RemoveLocalData(domain, namespace)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
