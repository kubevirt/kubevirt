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
 * Copyright 2017 John Levon <levon@movementarian.org>
 *
 */

// Package config tracks changes in the kubevirt-config ConfigMap,
// providing access to any of the current key values via Get()
package config

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type KubevirtConfig struct {
	informer cache.SharedInformer
	stop     chan struct{}
}

func NewKubevirtConfig(cli rest.Interface) *KubevirtConfig {
	var config KubevirtConfig

	watcher := cache.NewListWatchFromClient(cli, "configmaps", v1.NamespaceDefault,
		fields.ParseSelectorOrDie("metadata.name=kubevirt-config"))

	config.informer = cache.NewSharedInformer(watcher, &v1.ConfigMap{}, 0)

	go config.informer.Run(config.stop)

	return &config
}

func (c *KubevirtConfig) Get(key string) (value string, ok bool) {
	storeval, ok, _ := c.informer.GetStore().GetByKey("default/kubevirt-config")
	if !ok {
		return
	}

	value, ok = storeval.(*v1.ConfigMap).Data[key]
	return
}
