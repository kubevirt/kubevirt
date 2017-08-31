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

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/precond"
)

type ConfigDiskClient interface {
	Define(vm *v1.VM) (bool, error)
	Undefine(vm *v1.VM) error
}

type configDiskClient struct {
	lock sync.Mutex
	jobs map[string]chan string
}

func NewConfigDiskClient() ConfigDiskClient {
	return &configDiskClient{
		jobs: make(map[string]chan string),
	}
}

func (c *configDiskClient) Define(vm *v1.VM) (bool, error) {
	pending := false

	cloudInitSpec := cloudinit.GetCloudInitSpec(vm)
	if cloudInitSpec == nil {
		return false, nil
	}
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	c.lock.Lock()
	defer c.lock.Unlock()

	v, ok := c.jobs[namespace+domain]
	if ok == false {
		v = make(chan string, 1)

		go func() {
			err := cloudinit.GenerateLocalData(domain, namespace, cloudInitSpec)
			if err == nil {
				v <- "success"
			} else {
				v <- fmt.Sprintf("config-disk failure: %v", err)
			}
		}()

		c.jobs[namespace+domain] = v
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
			delete(c.jobs, namespace+domain)
		default:
			pending = true
		}
	}

	return pending, nil
}

func (c *configDiskClient) Undefine(vm *v1.VM) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	delete(c.jobs, namespace+domain)

	return cloudinit.RemoveLocalData(domain, namespace)
}
