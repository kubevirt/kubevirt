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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

var _ = Describe("export-proxy HPA metrics profile cache", func() {
	It("serves cached profiles without re-probing before the TTL expires", func() {
		profileCache := NewExportProxyHPAMetricsProfileCache()
		profileCache.entries["kubevirt"] = exportProxyHPAMetricsProfileCacheEntry{
			profile:  components.ExportProxyHPAMetricsProfileCustomMetrics,
			cachedAt: time.Now(),
		}

		detectCalls := 0
		profile := profileCache.Resolve("kubevirt", func() components.ExportProxyHPAMetricsProfile {
			detectCalls++
			return components.ExportProxyHPAMetricsProfileResource
		})

		Expect(profile).To(Equal(components.ExportProxyHPAMetricsProfileCustomMetrics))
		Expect(detectCalls).To(Equal(0))
	})

	It("refreshes detection after the TTL expires and then serves cached results", func() {
		profileCache := NewExportProxyHPAMetricsProfileCache()
		profileCache.entries["kubevirt"] = exportProxyHPAMetricsProfileCacheEntry{
			profile:  components.ExportProxyHPAMetricsProfileResource,
			cachedAt: time.Now().Add(-exportProxyHPAMetricsProfileCacheTTL - time.Second),
		}

		detectCalls := 0
		profile := profileCache.Resolve("kubevirt", func() components.ExportProxyHPAMetricsProfile {
			detectCalls++
			return components.ExportProxyHPAMetricsProfileCustomMetrics
		})

		Expect(profile).To(Equal(components.ExportProxyHPAMetricsProfileCustomMetrics))
		Expect(detectCalls).To(Equal(1))

		detectCalls = 0
		profile = profileCache.Resolve("kubevirt", func() components.ExportProxyHPAMetricsProfile {
			detectCalls++
			return components.ExportProxyHPAMetricsProfileResource
		})
		Expect(profile).To(Equal(components.ExportProxyHPAMetricsProfileCustomMetrics))
		Expect(detectCalls).To(Equal(0))
	})

	It("downgrades from custom-metrics after the TTL expires when detection fails", func() {
		profileCache := NewExportProxyHPAMetricsProfileCache()
		profileCache.entries["kubevirt"] = exportProxyHPAMetricsProfileCacheEntry{
			profile:  components.ExportProxyHPAMetricsProfileCustomMetrics,
			cachedAt: time.Now().Add(-exportProxyHPAMetricsProfileCacheTTL - time.Second),
		}

		detectCalls := 0
		profile := profileCache.Resolve("kubevirt", func() components.ExportProxyHPAMetricsProfile {
			detectCalls++
			return components.ExportProxyHPAMetricsProfileResource
		})

		Expect(profile).To(Equal(components.ExportProxyHPAMetricsProfileResource))
		Expect(detectCalls).To(Equal(1))
	})

	It("detects immediately when the cache is nil", func() {
		detectCalls := 0
		var profileCache *ExportProxyHPAMetricsProfileCache
		profile := profileCache.Resolve("kubevirt", func() components.ExportProxyHPAMetricsProfile {
			detectCalls++
			return components.ExportProxyHPAMetricsProfileResource
		})
		Expect(profile).To(Equal(components.ExportProxyHPAMetricsProfileResource))
		Expect(detectCalls).To(Equal(1))
	})
})
