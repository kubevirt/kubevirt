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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package device

import (
	hwutil "kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func NewPciAddressField(address string) (*api.Address, error) {
	dbsfFields, err := hwutil.ParsePciAddress(address)
	if err != nil {
		return nil, err
	}

	return &api.Address{
		Type:     "pci",
		Domain:   "0x" + dbsfFields[0],
		Bus:      "0x" + dbsfFields[1],
		Slot:     "0x" + dbsfFields[2],
		Function: "0x" + dbsfFields[3],
	}, nil
}
