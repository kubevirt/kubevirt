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

package apply

import (
	"sync"
	"time"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const exportProxyHPAMetricsProfileCacheTTL = 5 * time.Minute

type exportProxyHPAMetricsProfileCacheEntry struct {
	profile  components.ExportProxyHPAMetricsProfile
	cachedAt time.Time
}

// ExportProxyHPAMetricsProfileCache memoizes export-proxy HPA metrics profile
// detection across KubeVirt reconciles.
type ExportProxyHPAMetricsProfileCache struct {
	mu      sync.Mutex
	entries map[string]exportProxyHPAMetricsProfileCacheEntry
}

func NewExportProxyHPAMetricsProfileCache() *ExportProxyHPAMetricsProfileCache {
	return &ExportProxyHPAMetricsProfileCache{
		entries: map[string]exportProxyHPAMetricsProfileCacheEntry{},
	}
}

// Resolve returns the metrics profile for namespace, probing only when needed.
// Results are cached for exportProxyHPAMetricsProfileCacheTTL so brief probe or
// adapter outages do not flip the HPA every reconcile. When the TTL expires the
// profile is re-detected, including downgrading from custom-metrics to resource
// if prometheus-adapter or export-proxy transfer metrics are gone.
func (c *ExportProxyHPAMetricsProfileCache) Resolve(
	namespace string,
	detect func() components.ExportProxyHPAMetricsProfile,
) components.ExportProxyHPAMetricsProfile {
	if c == nil {
		return detect()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[namespace]; ok && time.Since(entry.cachedAt) < exportProxyHPAMetricsProfileCacheTTL {
		log.Log.V(4).Infof("export-proxy HPA metrics profile cache hit for namespace %s: %s (age=%s)",
			namespace, entry.profile, time.Since(entry.cachedAt).Round(time.Second))
		return entry.profile
	}

	profile := detect()
	c.entries[namespace] = exportProxyHPAMetricsProfileCacheEntry{
		profile:  profile,
		cachedAt: time.Now(),
	}
	log.Log.Infof("export-proxy HPA metrics profile cached for namespace %s: %s (ttl=%s)",
		namespace, profile, exportProxyHPAMetricsProfileCacheTTL)
	return profile
}
