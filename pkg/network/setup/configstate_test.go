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
	"errors"
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/cache"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
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
		networkNames     []string
		ns               nsExecutorStub
	)

	Context("Run", func() {
		BeforeEach(func() {
			configStateCache = newConfigStateCacheStub()
			ns = nsExecutorStub{}
			configState = NewConfigState(&configStateCache, ns)
			networkNames = []string{testNet0}
		})

		It("runs with no current state (cache is empty)", func() {
			config := &configStub{}

			Expect(configState.Run(networkNames, config.f)).To(Succeed())

			Expect(configStateCache.Read(testNet0)).To(Equal(cache.PodIfaceNetworkPreparationFinished))
		})

		It("runs with current pending state", func() {
			Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationPending)).To(Succeed())
			config := &configStub{}

			Expect(configState.Run(networkNames, config.f)).To(Succeed())

			Expect(configStateCache.Read(testNet0)).To(Equal(cache.PodIfaceNetworkPreparationFinished))
		})

		It("runs with current started state", func() {
			Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationStarted)).To(Succeed())
			config := &configStub{}

			ns.shouldNotBeExecuted = true
			err := configState.Run(networkNames, config.f)

			Expect(err).To(HaveOccurred())
			var criticalNetErr *neterrors.CriticalNetworkError
			Expect(errors.As(err, &criticalNetErr)).To(BeTrue())

			Expect(config.executed).To(BeFalse(), "the config step should not be execute")

			Expect(configStateCache.Read(testNet0)).To(Equal(cache.PodIfaceNetworkPreparationStarted))
		})

		It("runs with current finished state", func() {
			Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())
			config := &configStub{}

			ns.shouldNotBeExecuted = true
			Expect(configState.Run(networkNames, config.f)).To(Succeed())

			Expect(config.executed).To(BeFalse(), "the config step should not execute")

			Expect(configStateCache.Read(testNet0)).To(Equal(cache.PodIfaceNetworkPreparationFinished))
		})

		It("runs and fails at the setup step", func() {
			injectedErr := fmt.Errorf("fail config")
			config := &configStub{errRun: injectedErr}

			Expect(configState.Run(networkNames, config.f)).To(MatchError(injectedErr))

			Expect(config.executed).To(BeTrue(), "the config step should execute")

			Expect(configStateCache.Read(testNet0)).To(Equal(cache.PodIfaceNetworkPreparationStarted))
		})

		It("runs and fails reading the cache", func() {
			injectedErr := fmt.Errorf("fail read cache")
			configStateCache.readErr = injectedErr
			configState = NewConfigState(&configStateCache, ns)

			config := &configStub{}

			ns.shouldNotBeExecuted = true
			Expect(configState.Run(networkNames, config.f)).To(MatchError(injectedErr))

			Expect(config.executed).To(BeFalse(), "the config step shouldn't execute")
		})

		It("runs and fails writing the cache", func() {
			injectedErr := fmt.Errorf("fail write cache")
			configStateCache.writeErr = injectedErr
			configState = NewConfigState(&configStateCache, ns)

			config := &configStub{}

			Expect(configState.Run(networkNames, config.f)).To(MatchError(ContainSubstring(injectedErr.Error())))

			Expect(config.executed).To(BeFalse(), "the config step shouldn't execute")
		})

		When("with multiple interfaces", func() {
			BeforeEach(func() {
				networkNames = append(networkNames, testNet1, testNet2)
			})

			It("runs with no current state (cache is empty)", func() {
				config := &configStub{}

				Expect(configState.Run(networkNames, config.f)).To(Succeed())

				Expect(config.executed).To(BeTrue())

				for _, testNet := range []string{testNet0, testNet1, testNet2} {
					Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationFinished))
				}
			})

			It("runs with current state set as pending and finished", func() {
				Expect(configStateCache.Write(testNet1, cache.PodIfaceNetworkPreparationPending)).To(Succeed())
				Expect(configStateCache.Write(testNet2, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())

				config := &configStub{}

				Expect(configState.Run(networkNames, config.f)).To(Succeed())

				Expect(config.executed).To(BeTrue())

				for _, testNet := range []string{testNet0, testNet1, testNet2} {
					Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationFinished))
				}
			})

			It("runs with current state (for one network) set as started causes critical error", func() {
				Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationPending)).To(Succeed())
				Expect(configStateCache.Write(testNet1, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())
				Expect(configStateCache.Write(testNet2, cache.PodIfaceNetworkPreparationStarted)).To(Succeed())

				config := &configStub{}

				err := configState.Run(networkNames, config.f)

				var criticalNetErr *neterrors.CriticalNetworkError
				Expect(errors.As(err, &criticalNetErr)).To(BeTrue())
				Expect(err).To(MatchError("Critical network error: network testnet2 preparation cannot be restarted"))

				Expect(config.executed).To(BeFalse())

				Expect(configStateCache.Read(testNet0)).To(Equal(cache.PodIfaceNetworkPreparationPending))
				Expect(configStateCache.Read(testNet1)).To(Equal(cache.PodIfaceNetworkPreparationFinished))
				Expect(configStateCache.Read(testNet2)).To(Equal(cache.PodIfaceNetworkPreparationStarted))
			})

			It("runs and fails at the setup step", func() {
				injectedErr := fmt.Errorf("fail write cache")
				configStateCache.writeErr = injectedErr
				configState = NewConfigState(&configStateCache, ns)
				config := &configStub{}

				Expect(configState.Run(networkNames, config.f)).To(MatchError(ContainSubstring(injectedErr.Error())))

				Expect(config.executed).To(BeFalse(), "the config step shouldn't execute")
				for _, testNet := range []string{testNet0, testNet1, testNet2} {
					Expect(configStateCache.Read(testNet)).To(Equal(cache.PodIfaceNetworkPreparationPending))
				}
			})
		})
	})

	Context("Unplug", func() {
		var (
			filterFunc *filterFuncStub
		)

		BeforeEach(func() {
			configStateCache = newConfigStateCacheStub()
			configState = NewConfigState(&configStateCache, nsExecutorStub{})

			Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())
			Expect(configStateCache.Write(testNet1, cache.PodIfaceNetworkPreparationStarted)).To(Succeed())
			Expect(configStateCache.Write(testNet2, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())
		})
		It("There are no networks to unplug", func() {
			specNetworks := []v1.Network{}
			filterFunc = &filterFuncStub{[]string{}}
			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, filterFunc.f, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(BeEmpty(), "the unplug step shouldn't execute")
		})
		It("There are no networks to unplug since they are filtered out", func() {
			specNetworks := []v1.Network{{Name: testNet0}, {Name: testNet1}, {Name: testNet2}}
			filterFunc = &filterFuncStub{[]string{}}
			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, filterFunc.f, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(BeEmpty(), "the unplug step shouldn't execute")
		})
		It("There is one network to unplug", func() {
			specNetworks := []v1.Network{{Name: testNet0}}
			filterFunc = &filterFuncStub{nil}

			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, filterFunc.f, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(ConsistOf([]string{testNet0}))
		})
		It("There is one network to unplug but it is Pending", func() {
			specNetworks := []v1.Network{{Name: testNet0}}
			ns.shouldNotBeExecuted = true
			filterFunc = &filterFuncStub{nil}
			Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationPending)).To(Succeed())
			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, filterFunc.f, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(BeEmpty(), "the unplug step shouldn't execute")
		})
		It("There are multiple networks to unplug but one is Pending", func() {
			specNetworks := []v1.Network{{Name: testNet0}, {Name: testNet1}, {Name: testNet2}}
			filterFunc = &filterFuncStub{nil}
			Expect(configStateCache.Write(testNet2, cache.PodIfaceNetworkPreparationPending)).To(Succeed())
			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, filterFunc.f, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(ConsistOf([]string{testNet0, testNet1}))
		})
		It("There are multiple networks to unplug but one is filtered out", func() {
			specNetworks := []v1.Network{{Name: testNet0}, {Name: testNet1}, {Name: testNet2}}
			filterFunc = &filterFuncStub{[]string{testNet0, testNet1}}
			unplugFunc := &unplugFuncStub{}
			err := configState.Unplug(specNetworks, filterFunc.f, unplugFunc.f)
			Expect(err).NotTo(HaveOccurred())
			Expect(unplugFunc.executedNetworks).To(ConsistOf([]string{testNet0, testNet1}))
		})
		It("There are multiple networks to unplug and some have errors on cleanup", func() {
			specNetworks := []v1.Network{{Name: testNet0}, {Name: testNet1}, {Name: testNet2}}
			filterFunc = &filterFuncStub{nil}
			injectedErr := fmt.Errorf("fails unplug")
			injectedErr2 := fmt.Errorf("fails unplug2")
			unplugFunc := &unplugFuncStub{errRunForPodIfaces: map[string]error{testNet0: injectedErr, testNet2: injectedErr2}}
			err := configState.Unplug(specNetworks, filterFunc.f, unplugFunc.f)
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
