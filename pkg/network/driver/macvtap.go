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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package driver

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/network/driver/nmstate"
)

// LoadMacvtapDriver loads the macvtap kmod.
// The kmod is loaded implicitly by creating a macvtap link which is then removed.
func LoadMacvtapDriver() error {
	const randCharLen = 10
	randSuffix := rand.String(randCharLen)
	baseIfaceName := "dummy" + randSuffix
	nmState := nmstate.New()
	err := nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
		{
			Name:     baseIfaceName,
			TypeName: nmstate.TypeDummy,
			State:    nmstate.IfaceStateUp,
		},
		{
			Name:     "mvtap" + randSuffix,
			TypeName: nmstate.TypeMacvtap,
			State:    nmstate.IfaceStateUp,
			Macvtap: &nmstate.MacvtapDevice{
				BaseIface: baseIfaceName,
				Mode:      "passthru",
				UID:       0,
				GID:       0,
			},
		},
	}})
	if err != nil {
		return fmt.Errorf("failed to create macvtap device: %w", err)
	}
	err = nmState.Apply(&nmstate.Spec{Interfaces: []nmstate.Interface{
		{
			Name:     baseIfaceName,
			TypeName: nmstate.TypeDummy,
			State:    nmstate.IfaceStateAbsent,
		},
	}})
	if err != nil {
		return fmt.Errorf("failed to remove macvtap device: %w", err)
	}
	return nil
}
