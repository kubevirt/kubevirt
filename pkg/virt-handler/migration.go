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

package virthandler

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// FindMigrationIP looks for dedicated migration network migration0 using the downward API and, if found, sets migration IP to it
func FindMigrationIP(networkStatusPath string, migrationIp string) (string, error) {
	var networkStatus []NetworkStatus
	var dat []byte
	var err error

	for i := 0; i < 5; i++ {
		dat, err = os.ReadFile(networkStatusPath)
		if err != nil {
			return "", fmt.Errorf("failed to read network status from downwards API")
		}
		if len(dat) != 0 {
			break
		}
		if i < 4 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	err = yaml.Unmarshal(dat, &networkStatus)
	if err != nil {
		return "", fmt.Errorf("failed to un-marshall network status")
	}
	for _, ns := range networkStatus {
		if ns.Interface == "migration0" && len(ns.Ips) > 0 {
			migrationIp = ns.Ips[0]
		}
	}

	return migrationIp, nil
}
