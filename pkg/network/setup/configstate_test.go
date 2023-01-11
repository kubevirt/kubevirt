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
	"errors"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"kubevirt.io/kubevirt/pkg/network/cache"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	network "kubevirt.io/kubevirt/pkg/network/setup"
)

const (
	uid = "123"

	testNet = "testnet"
)

var _ = Describe("config state", func() {
	var (
		configState      network.ConfigState
		baseCacheCreator tempCacheCreator
	)

	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()
		configState = network.NewConfigState(&baseCacheCreator, uid)
	})
	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})

	It("runs with no current state (cache is empty)", func() {
		discover, config := &funcStub{}, &funcStub{}

		Expect(configState.Run(testNet, discover.f, config.f)).To(Succeed())

		Expect(discover.executed).To(BeTrue(), "the discover step should execute")
		Expect(config.executed).To(BeTrue(), "the config step should execute")

		podIFaceCacheDataResult, err := cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
		Expect(err).NotTo(HaveOccurred())
		Expect(podIFaceCacheDataResult.State).To(Equal(cache.PodIfaceNetworkPreparationFinished))
	})

	It("runs with current pending state", func() {
		podIfaceCacheData := &cache.PodIfaceCacheData{State: cache.PodIfaceNetworkPreparationPending}
		Expect(cache.WritePodInterfaceCache(&baseCacheCreator, uid, testNet, podIfaceCacheData)).To(Succeed())
		discover, config := &funcStub{}, &funcStub{}

		Expect(configState.Run(testNet, discover.f, config.f)).To(Succeed())

		Expect(discover.executed).To(BeTrue(), "the discover step should execute")
		Expect(config.executed).To(BeTrue(), "the config step should execute")

		podIFaceCacheDataResult, err := cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
		Expect(err).NotTo(HaveOccurred())
		Expect(podIFaceCacheDataResult.State).To(Equal(cache.PodIfaceNetworkPreparationFinished))
	})

	It("runs with current started state", func() {
		podIfaceCacheData := &cache.PodIfaceCacheData{State: cache.PodIfaceNetworkPreparationStarted}
		Expect(cache.WritePodInterfaceCache(&baseCacheCreator, uid, testNet, podIfaceCacheData)).To(Succeed())
		discover, config := &funcStub{}, &funcStub{}

		err := configState.Run(testNet, discover.f, config.f)
		var criticalNetErr *neterrors.CriticalNetworkError
		Expect(errors.As(err, &criticalNetErr)).To(BeTrue())

		Expect(discover.executed).To(BeFalse(), "the discover step should not be execute")
		Expect(config.executed).To(BeFalse(), "the config step should not be execute")

		podIFaceCacheDataResult, err := cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
		Expect(err).NotTo(HaveOccurred())
		Expect(podIFaceCacheDataResult.State).To(Equal(cache.PodIfaceNetworkPreparationStarted))
	})

	It("runs with current finished state", func() {
		podIfaceCacheData := &cache.PodIfaceCacheData{State: cache.PodIfaceNetworkPreparationFinished}
		Expect(cache.WritePodInterfaceCache(&baseCacheCreator, uid, testNet, podIfaceCacheData)).To(Succeed())
		discover, config := &funcStub{}, &funcStub{}

		Expect(configState.Run(testNet, discover.f, config.f)).To(Succeed())

		Expect(discover.executed).To(BeFalse(), "the discover step should not be execute")
		Expect(config.executed).To(BeFalse(), "the config step should not be execute")

		podIFaceCacheDataResult, err := cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
		Expect(err).NotTo(HaveOccurred())
		Expect(podIFaceCacheDataResult.State).To(Equal(cache.PodIfaceNetworkPreparationFinished))
	})

	It("runs and fails at the discover step", func() {
		injectedErr := fmt.Errorf("fail discovery")
		discover, config := &funcStub{errRun: injectedErr}, &funcStub{}

		Expect(configState.Run(testNet, discover.f, config.f)).To(MatchError(injectedErr))

		Expect(discover.executed).To(BeTrue(), "the discover step should execute")
		Expect(config.executed).To(BeFalse(), "the config step should not execute")

		_, err := cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
		Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue(), "expect no cache to exist")
	})

	It("runs and fails at the config step", func() {
		injectedErr := fmt.Errorf("fail config")
		discover, config := &funcStub{}, &funcStub{errRun: injectedErr}

		Expect(configState.Run(testNet, discover.f, config.f)).To(MatchError(injectedErr))

		Expect(discover.executed).To(BeTrue(), "the discover step should execute")
		Expect(config.executed).To(BeTrue(), "the config step should execute")

		podIFaceCacheDataResult, err := cache.ReadPodInterfaceCache(&baseCacheCreator, uid, testNet)
		Expect(err).NotTo(HaveOccurred())
		Expect(podIFaceCacheDataResult.State).To(Equal(cache.PodIfaceNetworkPreparationStarted))
	})
})

type funcStub struct {
	errRun   error
	executed bool
}

func (f *funcStub) f() error {
	f.executed = true
	return f.errRun
}
