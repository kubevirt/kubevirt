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

package netpod

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
)

type stateCacheReaderWriterDeleter interface {
	Read(networkName string) (cache.PodIfaceState, error)
	Write(networkName string, state cache.PodIfaceState) error
	Delete(networkName string) error
}

type State struct {
	cache stateCacheReaderWriterDeleter

	NSExec NSExecutor
}

func NewState(cache stateCacheReaderWriterDeleter, ns NSExecutor) *State {
	return &State{cache: cache, NSExec: ns}
}

func (s *State) PendingStartedFinished(nets []v1.Network) ([]v1.Network, []v1.Network, []v1.Network, error) {
	var pendingNets []v1.Network
	var startedNets []v1.Network
	var finishedNets []v1.Network
	for _, net := range nets {
		state, err := s.cache.Read(net.Name)
		if err != nil {
			return nil, nil, nil, err
		}

		switch state {
		case cache.PodIfaceNetworkPreparationPending:
			pendingNets = append(pendingNets, net)
		case cache.PodIfaceNetworkPreparationStarted:
			startedNets = append(startedNets, net)
		case cache.PodIfaceNetworkPreparationFinished:
			finishedNets = append(finishedNets, net)
		}
	}
	return pendingNets, startedNets, finishedNets, nil
}

func (s *State) SetStarted(nets []v1.Network) error {
	for _, net := range nets {
		if werr := s.cache.Write(net.Name, cache.PodIfaceNetworkPreparationStarted); werr != nil {
			return fmt.Errorf("failed to mark configuration as started for %s: %v", net.Name, werr)
		}
	}
	return nil
}

func (s *State) SetFinished(nets []v1.Network) error {
	for _, net := range nets {
		if werr := s.cache.Write(net.Name, cache.PodIfaceNetworkPreparationFinished); werr != nil {
			return neterrors.CreateCriticalNetworkError(
				fmt.Errorf("failed to mark configuration as finished for %s: %w", net.Name, werr),
			)
		}
	}
	return nil
}

func (s *State) Delete(nets []v1.Network) error {
	for _, net := range nets {
		if err := s.cache.Delete(net.Name); err != nil {
			return fmt.Errorf("failed to clear state cache for %s: %w", net.Name, err)
		}
	}
	return nil
}
