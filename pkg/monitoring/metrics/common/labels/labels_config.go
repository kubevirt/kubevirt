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

package labels

import (
	"fmt"
	"strings"
	"sync"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
)

var (
	defaultAllowlist  = []string{"*"}
	defaultIgnorelist = []string{}
)

const (
	configMapName = "kubevirt-vm-labels-config"
)

type Config interface {
	ShouldReport(label string) bool
}

type configImpl struct {
	mu         sync.RWMutex
	allowlist  []string
	ignorelist []string
	allowAll   bool
}

// New creates a new labels config instance and, if a client is provided,
// starts a watcher to keep it updated from the ConfigMap.
func New(client kubecli.KubevirtClient) (Config, error) {
	cfg := &configImpl{
		allowlist:  append([]string{}, defaultAllowlist...),
		ignorelist: append([]string{}, defaultIgnorelist...),
		allowAll:   true,
	}
	if err := cfg.startWatcherWithClient(client); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *configImpl) startWatcherWithClient(client kubecli.KubevirtClient) error {
	if client == nil {
		return fmt.Errorf("nil kubevirt client")
	}

	namespace, err := clientutil.GetNamespace()
	if err != nil {
		return fmt.Errorf("failed to determine namespace for watcher: %w", err)
	}

	lw := cache.NewListWatchFromClient(
		client.CoreV1().RESTClient(),
		"configmaps",
		namespace,
		fields.OneTermEqualSelector("metadata.name", configMapName),
	)

	informer := cache.NewSharedIndexInformer(
		lw,
		&k8sv1.ConfigMap{},
		0,
		cache.Indexers{},
	)

	c.attachHandlersToInformer(informer)
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		informer.Run(stop)
	}()
	return nil
}

func (c *configImpl) attachHandlersToInformer(informer cache.SharedIndexInformer) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cm, ok := obj.(*k8sv1.ConfigMap)
			if !ok {
				log.Log.Warningf("vm-labels: Add handler received unexpected object type %T", obj)
				return
			}
			c.updateFromConfigMap(cm)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			cm, ok := newObj.(*k8sv1.ConfigMap)
			if !ok {
				log.Log.Warningf("vm-labels: Update handler received unexpected object type %T", newObj)
				return
			}
			c.updateFromConfigMap(cm)
		},
		DeleteFunc: func(obj interface{}) {
			c.resetToDefaults()
		},
	})
}

func (c *configImpl) ShouldReport(label string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.allowlist) == 0 {
		return false
	}

	for _, ig := range c.ignorelist {
		if ig == label {
			return false
		}
	}

	if c.allowAll {
		return true
	}

	for _, a := range c.allowlist {
		if a == label {
			return true
		}
	}

	return false
}

func (c *configImpl) updateFromConfigMap(configMap *k8sv1.ConfigMap) {
	if configMap == nil || configMap.Data == nil {
		c.resetToDefaults()
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if d, ok := configMap.Data["allowlist"]; ok {
		c.allowlist = parseLabels(d)
	} else {
		c.allowlist = append([]string{}, defaultAllowlist...)
	}
	c.allowAll = false
	for _, a := range c.allowlist {
		if a == "*" {
			c.allowAll = true
			break
		}
	}
	if d, ok := configMap.Data["ignorelist"]; ok {
		c.ignorelist = parseLabels(d)
	} else {
		c.ignorelist = append([]string{}, defaultIgnorelist...)
	}
}

func (c *configImpl) resetToDefaults() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.allowlist = append([]string{}, defaultAllowlist...)
	c.ignorelist = append([]string{}, defaultIgnorelist...)
	c.allowAll = true
}

func parseLabels(data string) []string {
	parts := strings.Split(data, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			out = append(out, s)
		}
	}

	return out
}
