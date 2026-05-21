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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package v1

import (
	"fmt"

	netutils "k8s.io/utils/net"
)

func sanitizeIP(address string) (string, error) {
	sanitizedAddress := netutils.ParseIPSloppy(address)
	if sanitizedAddress == nil {
		return "", fmt.Errorf("not a valid IP address")
	}

	return sanitizedAddress.String(), nil
}

func sanitizeCIDR(cidr string) (string, error) {
	ip, net, err := netutils.ParseCIDRSloppy(cidr)
	if err != nil {
		return "", err
	}

	netMaskSize, _ := net.Mask.Size()
	return fmt.Sprintf("%s/%d", ip.String(), netMaskSize), nil
}
