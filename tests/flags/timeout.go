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

package flags

import (
	"flag"
	"time"
)

const (
	DefaultVMIStartupTimeout = 360
	DefaultVMStopTimeout     = 300
	DefaultVMReadyTimeout    = 360
	DefaultMigrationTimeout  = 240
)

var (
	vmiStartupTimeout int
	vmStopTimeout     int
	vmReadyTimeout    int
	migrationTimeout  int
)

func init() {
	flag.IntVar(&vmiStartupTimeout, "vmi-startup-timeout", DefaultVMIStartupTimeout,
		"Timeout in seconds to wait for a VMI to start")
	flag.IntVar(&vmStopTimeout, "vm-stop-timeout", DefaultVMStopTimeout,
		"Timeout in seconds to wait for a VM to stop")
	flag.IntVar(&vmReadyTimeout, "vm-ready-timeout", DefaultVMReadyTimeout,
		"Timeout in seconds to wait for a VM to become ready")
	flag.IntVar(&migrationTimeout, "migration-timeout", DefaultMigrationTimeout,
		"Timeout in seconds to wait for a migration to complete")
}

func VMIStartupTimeout() int { return vmiStartupTimeout }
func VMStopTimeout() int     { return vmStopTimeout }
func VMReadyTimeout() int    { return vmReadyTimeout }
func MigrationTimeout() int  { return migrationTimeout }

func VMIStartupTimeoutInSeconds() time.Duration {
	return time.Duration(vmiStartupTimeout) * time.Second
}

func VMStopTimeoutInSeconds() time.Duration {
	return time.Duration(vmStopTimeout) * time.Second
}

func VMReadyTimeoutInSeconds() time.Duration {
	return time.Duration(vmReadyTimeout) * time.Second
}

func MigrationTimeoutInSeconds() time.Duration {
	return time.Duration(migrationTimeout) * time.Second
}

func scaledStartupTimeout(base int) int {
	return base * vmiStartupTimeout / DefaultVMIStartupTimeout
}

const (
	defaultStartupTimeoutSecondsTiny   = 30
	defaultStartupTimeoutSecondsSmall  = 60
	defaultStartupTimeoutSecondsMedium = 90
	defaultStartupTimeoutSecondsLarge  = 120
	defaultStartupTimeoutSecondsXLarge = 180
	defaultStartupTimeoutSecondsHuge   = 240
	defaultStartupTimeoutSecondsXHuge  = 360
)

func StartupTimeoutSecondsTiny() int {
	return scaledStartupTimeout(defaultStartupTimeoutSecondsTiny)
}

func StartupTimeoutSecondsSmall() int {
	return scaledStartupTimeout(defaultStartupTimeoutSecondsSmall)
}

func StartupTimeoutSecondsMedium() int {
	return scaledStartupTimeout(defaultStartupTimeoutSecondsMedium)
}

func StartupTimeoutSecondsLarge() int {
	return scaledStartupTimeout(defaultStartupTimeoutSecondsLarge)
}

func StartupTimeoutSecondsXLarge() int {
	return scaledStartupTimeout(defaultStartupTimeoutSecondsXLarge)
}

func StartupTimeoutSecondsHuge() int {
	return scaledStartupTimeout(defaultStartupTimeoutSecondsHuge)
}

func StartupTimeoutSecondsXHuge() int {
	return scaledStartupTimeout(defaultStartupTimeoutSecondsXHuge)
}
