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

package netpod_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	netcache "kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/setup/netpod"
)

var _ = Describe("state", func() {

	const netName = "foo"

	readErr := errors.New("read test error")
	writeErr := errors.New("write test error")
	deleteErr := errors.New("delete test error")

	It("fails reporting", func() {
		cache := newConfigStateCacheStub()
		cache.readErr = readErr

		state := netpod.NewState(cache, nil)
		_, _, _, err := state.PendingStartedFinished([]v1.Network{{Name: netName}})

		Expect(err).To(MatchError(readErr))
	})

	It("fails setting started state", func() {
		cache := newConfigStateCacheStub()
		cache.writeErr = writeErr

		state := netpod.NewState(cache, nil)
		Expect(state.SetStarted([]v1.Network{{Name: netName}})).To(MatchError(ContainSubstring(writeErr.Error())))
	})

	It("fails setting finished state", func() {
		cache := newConfigStateCacheStub()
		cache.writeErr = writeErr

		state := netpod.NewState(cache, nil)
		Expect(state.SetFinished([]v1.Network{{Name: netName}})).To(MatchError(ContainSubstring(writeErr.Error())))
	})

	It("fails deleting state", func() {
		cache := newConfigStateCacheStub()
		cache.deleteErr = deleteErr

		state := netpod.NewState(cache, nil)
		Expect(state.Delete([]v1.Network{{Name: netName}})).To(MatchError(ContainSubstring(deleteErr.Error())))
	})

	It("succeeds setting started state", func() {
		state := netpod.NewState(newConfigStateCacheStub(), nil)
		Expect(state.SetStarted([]v1.Network{{Name: netName}})).To(Succeed())

		pending, started, finished, err := state.PendingStartedFinished([]v1.Network{{Name: netName}})
		Expect(err).NotTo(HaveOccurred())

		Expect(pending).To(BeEmpty())
		Expect(started).To(Equal([]v1.Network{{Name: netName}}))
		Expect(finished).To(BeEmpty())
	})

	It("succeeds setting finished state", func() {
		state := netpod.NewState(newConfigStateCacheStub(), nil)
		Expect(state.SetFinished([]v1.Network{{Name: netName}})).To(Succeed())

		pending, started, finished, err := state.PendingStartedFinished([]v1.Network{{Name: netName}})
		Expect(err).NotTo(HaveOccurred())

		Expect(pending).To(BeEmpty())
		Expect(started).To(BeEmpty())
		Expect(finished).To(Equal([]v1.Network{{Name: netName}}))
	})

	It("reports a mix of network states", func() {
		nets := []v1.Network{
			{Name: "netpending1"},
			{Name: "netpending2"},
			{Name: "netstarted1"},
			{Name: "netstarted2"},
			{Name: "netfinished1"},
			{Name: "netfinished2"},
		}
		cache := newConfigStateCacheStub()
		cache.stateCache[nets[0].Name] = netcache.PodIfaceNetworkPreparationPending
		cache.stateCache[nets[1].Name] = netcache.PodIfaceNetworkPreparationPending
		cache.stateCache[nets[2].Name] = netcache.PodIfaceNetworkPreparationStarted
		cache.stateCache[nets[3].Name] = netcache.PodIfaceNetworkPreparationStarted
		cache.stateCache[nets[4].Name] = netcache.PodIfaceNetworkPreparationFinished
		cache.stateCache[nets[5].Name] = netcache.PodIfaceNetworkPreparationFinished

		state := netpod.NewState(cache, nil)
		pending, started, finished, err := state.PendingStartedFinished(nets)

		Expect(err).NotTo(HaveOccurred())
		Expect(pending).To(Equal([]v1.Network{nets[0], nets[1]}))
		Expect(started).To(Equal([]v1.Network{nets[2], nets[3]}))
		Expect(finished).To(Equal([]v1.Network{nets[4], nets[5]}))
	})

	It("succeeds deleting network state", func() {
		state := netpod.NewState(newConfigStateCacheStub(), nil)
		nets := []v1.Network{{Name: netName}}
		Expect(state.SetFinished(nets)).To(Succeed())

		Expect(state.Delete(nets)).To(Succeed())

		pending, started, finished, err := state.PendingStartedFinished(nets)
		Expect(err).NotTo(HaveOccurred())

		// On deletion, all networks cache are initialized back to "pending".
		Expect(pending).To(Equal(nets))
		Expect(started).To(BeEmpty())
		Expect(finished).To(BeEmpty())
	})
})
