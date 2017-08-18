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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package main

import (
	"flag"

	"github.com/spf13/pflag"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	configdisk "kubevirt.io/kubevirt/pkg/config-disk"
	"kubevirt.io/kubevirt/pkg/logging"
)

func main() {
	logging.InitializeLogging("virt-dynamic-disk")
	cloudInitDir := flag.String("cloud-init-dir", "/var/run/libvirt/cloud-init-dir", "Base directory for ephemeral cloud init data")
	configDiskSocket := flag.String("config-disk-socket", "/var/run/libvirt/config-disk-sock", "Base directory for ephemeral cloud init data")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	cloudinit.SetLocalDirectory(*cloudInitDir)

	configdisk.HttpServe(*configDiskSocket)
}
