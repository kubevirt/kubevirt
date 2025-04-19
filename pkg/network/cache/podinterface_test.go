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

package cache_test

import (
	"os"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	netcache "kubevirt.io/kubevirt/pkg/network/cache"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Pod Interface", func() {

	const UID = "123"
	var cacheCreator tempCacheCreator
	var podIfaceCache netcache.PodInterfaceCache
	var cacheData netcache.PodIfaceCacheData

	BeforeEach(dutils.MockDefaultOwnershipManager)

	BeforeEach(func() {
		podCache := netcache.NewPodInterfaceCache(&cacheCreator, UID)

		var err error
		podIfaceCache, err = podCache.IfaceEntry("net0")
		Expect(err).NotTo(HaveOccurred())

		cacheData = netcache.PodIfaceCacheData{
			Iface: &v1.Interface{
				Model: "nice model",
			},
			PodIP: "random ip",
			PodIPs: []string{
				"ip1", "ip2",
			},
		}
	})

	AfterEach(func() { Expect(cacheCreator.New("").Delete()).To(Succeed()) })

	It("should return os.ErrNotExist if no cache entry exists", func() {
		_, err := podIfaceCache.Read()
		Expect(err).To(MatchError(os.ErrNotExist))
	})
	It("should save and restore pod interface information", func() {
		Expect(podIfaceCache.Write(&cacheData)).To(Succeed())
		Expect(podIfaceCache.Read()).To(Equal(&cacheData))
	})
	It("should remove the cache file", func() {
		Expect(podIfaceCache.Write(&cacheData)).To(Succeed())
		Expect(podIfaceCache.Remove()).To(Succeed())

		_, err := podIfaceCache.Read()
		Expect(err).To(MatchError(os.ErrNotExist))
	})
})
