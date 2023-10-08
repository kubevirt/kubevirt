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

package network

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/cache"
)

const (
	testNet0 = "testnet0"
	testNet1 = "testnet1"
	testNet2 = "testnet2"

	launcherPid = 0
)

var _ = Describe("config state", func() {
	var (
		configState      ConfigState
		configStateCache configStateCacheStub
		ns               nsExecutorStub
	)

	Context("Unplug", func() {
		BeforeEach(func() {
			configStateCache = newConfigStateCacheStub()
			configState = NewConfigState(&configStateCache, nsExecutorStub{})

			Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())
			Expect(configStateCache.Write(testNet1, cache.PodIfaceNetworkPreparationStarted)).To(Succeed())
			Expect(configStateCache.Write(testNet2, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())
		})
		It("There are no networks to unplug", func() {
			specNetworks := []v1.Network{}
			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(BeEmpty(), "the unplug step shouldn't execute")
		})
		It("There is one network to unplug", func() {
			specNetworks := []v1.Network{{Name: testNet0}}

			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(ConsistOf([]string{testNet0}))
		})
		It("There is one network to unplug but it is Pending", func() {
			specNetworks := []v1.Network{{Name: testNet0}}
			ns.shouldNotBeExecuted = true
			Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationPending)).To(Succeed())
			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(BeEmpty(), "the unplug step shouldn't execute")
		})
		It("There are multiple networks to unplug but one is Pending", func() {
			specNetworks := []v1.Network{{Name: testNet0}, {Name: testNet1}, {Name: testNet2}}
			Expect(configStateCache.Write(testNet2, cache.PodIfaceNetworkPreparationPending)).To(Succeed())
			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(ConsistOf([]string{testNet0, testNet1}))
		})
		It("There are multiple networks to unplug and some have errors on cleanup", func() {
			specNetworks := []v1.Network{{Name: testNet0}, {Name: testNet1}, {Name: testNet2}}
			injectedErr := fmt.Errorf("fails unplug")
			injectedErr2 := fmt.Errorf("fails unplug2")
			unplugFunc := &unplugFuncStub{errRunForPodIfaces: map[string]error{testNet0: injectedErr, testNet2: injectedErr2}}
			err := configState.Unplug(specNetworks, unplugFunc.f)
			Expect(err.Error()).To(ContainSubstring(injectedErr.Error()))
			Expect(err.Error()).To(ContainSubstring(injectedErr2.Error()))
			Expect(unplugFunc.executedNetworks).To(ConsistOf([]string{testNet0, testNet1, testNet2}))
		})
	})
})

type configStub struct {
	errRun   error
	executed bool
}

func (c *configStub) f(hook func() error) error {
	if err := hook(); err != nil {
		return err
	}
	c.executed = true
	return c.errRun
}

type unplugFuncStub struct {
	executedNetworks   []string
	errRunForPodIfaces map[string]error
}

func (f *unplugFuncStub) f(name string) error {
	f.executedNetworks = append(f.executedNetworks, name)
	return f.errRunForPodIfaces[name]
}

type filterFuncStub struct {
	networks []string
}

func (f *filterFuncStub) f(networks []v1.Network) ([]string, error) {
	if f.networks == nil {
		netNames := []string{}
		for _, network := range networks {
			netNames = append(netNames, network.Name)
		}
		return netNames, nil
	}
	return f.networks, nil
}
