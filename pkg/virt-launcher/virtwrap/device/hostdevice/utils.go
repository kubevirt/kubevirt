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

package hostdevice

import (
	"fmt"
	"strings"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func ComposePCIAddressFromFields(addr *api.Address) (string, error) {
	if addr.Type != api.AddressPCI {
		return "", fmt.Errorf("not a PCI address")
	}
	fields := []string{addr.Domain, addr.Bus, addr.Slot, addr.Function}
	for _, field := range fields {
		if !strings.HasPrefix(field, "0x") {
			return "", fmt.Errorf("invalid PCI address")
		}
	}

	d := strings.TrimPrefix(addr.Domain, "0x")
	b := strings.TrimPrefix(addr.Bus, "0x")
	s := strings.TrimPrefix(addr.Slot, "0x")
	f := strings.TrimPrefix(addr.Function, "0x")
	return fmt.Sprintf("%s:%s:%s.%s", d, b, s, f), nil
}
