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
* Copyright 2023 Red Hat, Inc.
*
 */

package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	network "kubevirt.io/kubevirt/pkg/network/setup"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

var _ = Describe("config state cache", func() {
	const (
		uid     = "123"
		testNet = "testnet"
	)
	var (
		configStateCache      network.ConfigStateCache
		baseCacheCreator      tempCacheCreator
		volatilePodIfaceState map[string]cache.PodIfaceState
	)

	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()
		volatilePodIfaceState = map[string]cache.PodIfaceState{}
		configStateCache = network.NewConfigStateCacheWithPodIfaceStateData(uid, &baseCacheCreator, volatilePodIfaceState)
	})
	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})

	Context("read", func() {
		It("from an empty cache", func() {
			Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationPending))
			Expect(volatilePodIfaceState).To(HaveKeyWithValue(testNet, cache.PodIfaceNetworkPreparationPending))
		})
		It("state stored only in the file cache", func() {
			podIfaceCacheData := &cache.PodIfaceCacheData{State: cache.PodIfaceNetworkPreparationStarted}
			Expect(cache.WritePodInterfaceCache(&baseCacheCreator, uid, testNet, podIfaceCacheData)).To(Succeed())

			Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationStarted))
			Expect(volatilePodIfaceState[testNet]).To(Equal(cache.PodIfaceNetworkPreparationStarted))
		})
		It("state stored only in file cache while the memory cache is not empty", func() {
			podIfaceCacheData := &cache.PodIfaceCacheData{State: cache.PodIfaceNetworkPreparationStarted}
			Expect(cache.WritePodInterfaceCache(&baseCacheCreator, uid, testNet, podIfaceCacheData)).To(Succeed())

			volatilePodIfaceState["not"+testNet] = cache.PodIfaceNetworkPreparationStarted

			Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationStarted))
			Expect(volatilePodIfaceState[testNet]).To(Equal(cache.PodIfaceNetworkPreparationStarted))
		})
		It("state stored only in the memory", func() {
			volatilePodIfaceState[testNet] = cache.PodIfaceNetworkPreparationStarted

			state, err := configStateCache.Read(testNet)
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal(cache.PodIfaceNetworkPreparationStarted))
		})
		It("state stored in both the file cache and the memory, prefer memory", func() {
			podIfaceCacheData := &cache.PodIfaceCacheData{State: cache.PodIfaceNetworkPreparationStarted}
			Expect(cache.WritePodInterfaceCache(&baseCacheCreator, uid, testNet, podIfaceCacheData)).To(Succeed())
			volatilePodIfaceState[testNet] = cache.PodIfaceNetworkPreparationFinished

			Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationFinished))
		})
	})

	Context("write", func() {
		It("to an empty cache", func() {
			err := configStateCache.Write(testNet, cache.PodIfaceNetworkPreparationStarted)
			Expect(err).NotTo(HaveOccurred())
			Expect(volatilePodIfaceState[testNet]).To(Equal(cache.PodIfaceNetworkPreparationStarted))
			podIfaceCacheData, err := cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
			Expect(err).NotTo(HaveOccurred())
			Expect(podIfaceCacheData.State).To(Equal(cache.PodIfaceNetworkPreparationStarted))

		})
		It("to a non empty cache", func() {
			err := configStateCache.Write(testNet, cache.PodIfaceNetworkPreparationStarted)
			Expect(err).NotTo(HaveOccurred())
			Expect(volatilePodIfaceState[testNet]).To(Equal(cache.PodIfaceNetworkPreparationStarted))
			podIfaceCacheData, err := cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
			Expect(err).NotTo(HaveOccurred())
			Expect(podIfaceCacheData.State).To(Equal(cache.PodIfaceNetworkPreparationStarted))

			err = configStateCache.Write(testNet, cache.PodIfaceNetworkPreparationFinished)
			Expect(err).NotTo(HaveOccurred())
			Expect(volatilePodIfaceState[testNet]).To(Equal(cache.PodIfaceNetworkPreparationFinished))
			podIfaceCacheData, err = cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
			Expect(err).NotTo(HaveOccurred())
			Expect(podIfaceCacheData.State).To(Equal(cache.PodIfaceNetworkPreparationFinished))

		})
	})

	Context("delete", func() {
		It("from an empty cache", func() {
			Expect(configStateCache.Delete(testNet)).To(Succeed())
		})
		It("successfully", func() {
			Expect(configStateCache.Write(testNet, cache.PodIfaceNetworkPreparationStarted)).To(Succeed())
			Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationStarted))
			Expect(configStateCache.Delete(testNet)).To(Succeed())
			Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationPending))
		})
	})
})
