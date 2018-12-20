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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package virtconfig

/*
 This module is intended for determining whether an optional feature is enabled or not at the system-level.
 Note that the virtconfig package needs to be initialized before using this (see config-map.Init)
*/

import (
	"os"
	"strings"
)

const (
	dataVolumesGate      = "DataVolumes"
	cpuManager           = "CPUManager"
	liveMigrationGate    = "LiveMigration"
	SRIOVGate            = "SRIOV"
	CPUNodeDiscoveryGate = "CPUNodeDiscovery"
)

func DataVolumesEnabled() bool {
	return strings.Contains(os.Getenv(featureGateEnvVar), dataVolumesGate)
}

func CPUManagerEnabled() bool {
	return strings.Contains(os.Getenv(featureGateEnvVar), cpuManager)
}

func LiveMigrationEnabled() bool {
	return strings.Contains(os.Getenv(featureGateEnvVar), liveMigrationGate)
}

func SRIOVEnabled() bool {
	return strings.Contains(os.Getenv(featureGateEnvVar), SRIOVGate)
}
