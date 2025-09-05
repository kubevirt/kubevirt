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
	DefaultAllowlist  = []string{"*"}
	DefaultIgnorelist = []string{}

	initOnce sync.Once
	cfg      *configImpl
)

const (
	ConfigMapName = "kubevirt-vm-labels-config"
)

type Config interface {
	ShouldReport(label string) bool
}

type configImpl struct {
	mu         sync.RWMutex
	allowlist  []string
	ignorelist []string
}

func ensureCfg() *configImpl {
	if cfg == nil {
		cfg = &configImpl{
			allowlist:  append([]string{}, DefaultAllowlist...),
			ignorelist: append([]string{}, DefaultIgnorelist...),
		}
	}
	return cfg
}

func New(client kubecli.KubevirtClient) Config {
	initOnce.Do(func() {
		ensureCfg()
		startWatcher(client)
	})
	return ensureCfg()
}

func startWatcher(client kubecli.KubevirtClient) {
	if client == nil {
		return
	}

	namespace, err := clientutil.GetNamespace()
	if err != nil {
		log.Log.Errorf("vm-labels: failed to determine namespace for watcher: %v", err)
		return
	}

	lw := cache.NewListWatchFromClient(
		client.CoreV1().RESTClient(),
		"configmaps",
		namespace,
		fields.OneTermEqualSelector("metadata.name", ConfigMapName),
	)

	informer := cache.NewSharedIndexInformer(
		lw,
		&k8sv1.ConfigMap{},
		0,
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cm, ok := obj.(*k8sv1.ConfigMap)
			if !ok {
				return
			}
			ensureCfg().updateFromConfigMap(cm)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			cm, ok := newObj.(*k8sv1.ConfigMap)
			if !ok {
				return
			}
			ensureCfg().updateFromConfigMap(cm)
		},
		DeleteFunc: func(obj interface{}) {
			ensureCfg().resetToDefaults()
		},
	})

	stop := make(chan struct{})
	go func() {
		defer close(stop)
		informer.Run(stop)
	}()
}

func (c *configImpl) ShouldReport(label string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.allowlist) == 0 {
		return false
	}

	allowAll := false
	for _, a := range c.allowlist {
		if a == "*" {
			allowAll = true
			break
		}
	}

	for _, ig := range c.ignorelist {
		if ig == label {
			return false
		}
	}

	if allowAll {
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
	if configMap.Data == nil {
		c.resetToDefaults()
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if d, ok := configMap.Data["allowlist"]; ok {
		c.allowlist = parseLabels(d)
	} else {
		c.allowlist = append([]string{}, DefaultAllowlist...)
	}
	if d, ok := configMap.Data["ignorelist"]; ok {
		c.ignorelist = parseLabels(d)
	} else {
		c.ignorelist = append([]string{}, DefaultIgnorelist...)
	}
}

func (c *configImpl) resetToDefaults() {
	c.mu.Lock()
	c.allowlist = append([]string{}, DefaultAllowlist...)
	c.ignorelist = append([]string{}, DefaultIgnorelist...)
	c.mu.Unlock()
}

func parseLabels(data string) []string {
	if strings.TrimSpace(data) == "" {
		return []string{}
	}
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

// test-only helpers
func SetAllowlistForTest(list []string) {
	ensureCfg().mu.Lock()
	ensureCfg().allowlist = append([]string{}, list...)
	ensureCfg().mu.Unlock()
}

func SetIgnorelistForTest(list []string) {
	ensureCfg().mu.Lock()
	ensureCfg().ignorelist = append([]string{}, list...)
	ensureCfg().mu.Unlock()
}
