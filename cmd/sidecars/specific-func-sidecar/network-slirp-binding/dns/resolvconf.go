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

package dns

import (
	"fmt"
	"os"
	"strings"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/dns"
)

const resolvConf = "/etc/resolv.conf"

func ReadResolvConfSearchDomains() ([]string, error) {

	// #nosec No risk for path injection. resolvConf is static "/etc/resolve.conf"
	resolvConfRaw, err := os.ReadFile(resolvConf)
	if err != nil {
		return nil, fmt.Errorf("failed to read resolv.conf at %q: %v", resolvConf, err)
	}

	searchDomains, err := dns.ParseSearchDomains(string(resolvConfRaw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse search domains out of %q: %v", resolvConf, err)
	}

	log.Log.Infof("Found search domains in %s: %s", resolvConf, strings.Join(searchDomains, " "))

	return searchDomains, nil
}
