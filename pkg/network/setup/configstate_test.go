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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/cache"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
)

const (
	uid = "123"

	testNet0 = "testnet0"
	testNet1 = "testnet1"
	testNet2 = "testnet2"
)

var _ = Describe("config state", func() {
	var (
		configState      ConfigState
		configStateCache configStateCacheStub
		nics             []podNIC
		ns               nsExecutorStub
	)

	BeforeEach(func() {
		configStateCache = newConfigStateCacheStub()
		ns = nsExecutorStub{}
		configState = NewConfigState(&configStateCache, ns)
		nics = []podNIC{{
			podInterfaceName: testNet0,
		}}
	})

	It("runs with no current state (cache is empty)", func() {
		discover, config := &funcStub{}, &funcStub{}

		Expect(configState.Run(nics, discover.f, config.f)).To(Succeed())

		Expect(discover.executedNICs).To(Equal([]string{testNet0}), "the discover step should execute")
		Expect(config.executedNICs).To(Equal([]string{testNet0}), "the config step should execute")

		state, err := configStateCache.Read(testNet0)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(cache.PodIfaceNetworkPreparationFinished))
	})

	It("runs with current pending state", func() {
		Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationPending)).To(Succeed())
		discover, config := &funcStub{}, &funcStub{}

		Expect(configState.Run(nics, discover.f, config.f)).To(Succeed())

		Expect(discover.executedNICs).To(Equal([]string{testNet0}), "the discover step should execute")
		Expect(config.executedNICs).To(Equal([]string{testNet0}), "the config step should execute")

		state, err := configStateCache.Read(testNet0)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(cache.PodIfaceNetworkPreparationFinished))
	})

	It("runs with current started state", func() {
		Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationStarted)).To(Succeed())
		discover, config := &funcStub{}, &funcStub{}

		ns.shouldNotBeExecuted = true
		err := configState.Run(nics, discover.f, config.f)
		Expect(err).To(HaveOccurred())
		var criticalNetErr *neterrors.CriticalNetworkError
		Expect(errors.As(err, &criticalNetErr)).To(BeTrue())

		Expect(discover.executedNICs).To(BeEmpty(), "the discover step should not be execute")
		Expect(config.executedNICs).To(BeEmpty(), "the config step should not be execute")

		state, err := configStateCache.Read(testNet0)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(cache.PodIfaceNetworkPreparationStarted))
	})

	It("runs with current finished state", func() {
		Expect(configStateCache.Write(testNet0, cache.PodIfaceNetworkPreparationFinished)).To(Succeed())
		discover, config := &funcStub{}, &funcStub{}

		ns.shouldNotBeExecuted = true
		Expect(configState.Run(nics, discover.f, config.f)).To(Succeed())

		Expect(discover.executedNICs).To(BeEmpty(), "the discover step should not be execute")
		Expect(config.executedNICs).To(BeEmpty(), "the config step should not be execute")

		state, err := configStateCache.Read(testNet0)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(cache.PodIfaceNetworkPreparationFinished))
	})

	It("runs and fails at the discover step", func() {
		injectedErr := fmt.Errorf("fail discovery")
		discover, config := &funcStub{errRun: injectedErr}, &funcStub{}

		Expect(configState.Run(nics, discover.f, config.f)).To(MatchError(injectedErr))

		Expect(discover.executedNICs).To(Equal([]string{testNet0}), "the discover step should execute")
		Expect(config.executedNICs).To(BeEmpty(), "the config step should not execute")

		state, err := configStateCache.Read(testNet0)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(cache.PodIfaceNetworkPreparationPending))
	})

	It("runs and fails at the config step", func() {
		injectedErr := fmt.Errorf("fail config")
		discover, config := &funcStub{}, &funcStub{errRun: injectedErr}

		Expect(configState.Run(nics, discover.f, config.f)).To(MatchError(injectedErr))

		Expect(discover.executedNICs).To(Equal([]string{testNet0}), "the discover step should execute")
		Expect(config.executedNICs).To(Equal([]string{testNet0}), "the config step should execute")

		state, err := configStateCache.Read(testNet0)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(cache.PodIfaceNetworkPreparationStarted))
	})

	It("runs and fails reading the cache", func() {
		injectedErr := fmt.Errorf("fail read cache")
		configStateCache.readErr = injectedErr
		configState = NewConfigState(&configStateCache, ns)

		discover, config := &funcStub{}, &funcStub{}

		ns.shouldNotBeExecuted = true
		Expect(configState.Run(nics, discover.f, config.f)).To(MatchError(injectedErr))

		Expect(discover.executedNICs).To(BeEmpty(), "the discover step shouldn't execute")
		Expect(config.executedNICs).To(BeEmpty(), "the config step shouldn't execute")
	})

	It("runs and fails writing the cache", func() {
		injectedErr := fmt.Errorf("fail write cache")
		configStateCache.writeErr = injectedErr
		configState = NewConfigState(&configStateCache, ns)

		discover, config := &funcStub{}, &funcStub{}

		Expect(configState.Run(nics, discover.f, config.f)).To(MatchError(injectedErr))

		Expect(discover.executedNICs).To(Equal([]string{testNet0}), "the discover step should execute")
		Expect(config.executedNICs).To(BeEmpty(), "the config step shouldn't execute")
	})

	When("with multiple interfaces", func() {
		BeforeEach(func() {
			nics = append(nics,
				podNIC{podInterfaceName: testNet1},
				podNIC{podInterfaceName: testNet2},
			)
		})

		It("runs with no current state (cache is empty)", func() {
			discover, config := &funcStub{}, &funcStub{}

			Expect(configState.Run(nics, discover.f, config.f)).To(Succeed())

			Expect(discover.executedNICs).To(Equal([]string{testNet0, testNet1, testNet2}))
			Expect(config.executedNICs).To(Equal([]string{testNet0, testNet1, testNet2}))

			for _, testNet := range []string{testNet0, testNet1, testNet2} {
				state, err := configStateCache.Read(testNet)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(cache.PodIfaceNetworkPreparationFinished))
			}
		})

		It("runs and fails at the config step, 2nd network", func() {
			injectedErr := fmt.Errorf("fail config")
			discover, config := &funcStub{}, &funcStub{errRun: injectedErr, errRunForPodIfaceName: testNet1}

			Expect(configState.Run(nics, discover.f, config.f)).To(MatchError(injectedErr))

			Expect(discover.executedNICs).To(Equal([]string{testNet0, testNet1, testNet2}))
			Expect(config.executedNICs).To(Equal([]string{testNet0, testNet1}))

			for _, testNet := range []string{testNet0, testNet1, testNet2} {
				state, err := configStateCache.Read(testNet)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(cache.PodIfaceNetworkPreparationStarted))
			}
		})

	})
})

type funcStub struct {
	executedNICs          []string
	errRun                error
	errRunForPodIfaceName string
}

func (f *funcStub) f(nic *podNIC) error {
	f.executedNICs = append(f.executedNICs, nic.podInterfaceName)

	// If an error is specified, return it if there is no filter at all, or if the filter is specified and matches.
	// The filter is the pod interface name.
	var err error
	if f.errRunForPodIfaceName == "" || f.errRunForPodIfaceName == nic.podInterfaceName {
		err = f.errRun
	}
	return err
}
