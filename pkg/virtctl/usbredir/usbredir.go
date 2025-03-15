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
 * Copyright 2017, 2021 Red Hat, Inc.
 *
 */

package usbredir

import (
	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	usbredirClient = "usbredirect"

	optDisableClientLaunch = "no-launch"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "usbredir (vendor:product)|(bus-device) (VMI)",
		Short:   "Redirect an USB device to a virtual machine instance.",
		Example: usage(),
		Args:    cobra.RangeArgs(1, 2),
		RunE:    Run,
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	cmd.Flags().Bool(optDisableClientLaunch, false, "If set, you should launch the usbredir client yourself")
	return cmd
}

func usage() string {
	return `# Find the device you want to redirect (linux):
	$ lsusb | grep Kingston
	Bus 002 Device 003: ID 0951:1666 Kingston Technology DataTraveler 100 G3/G4/SE9 G2/50

	# Redirect it with vendor:product to testvmi:
    {{ProgramName}} usbredir 0951:1666 testvmi

	# Redirect it with bus-device:
    {{ProgramName}} usbredir 02-03 testvmi

	# Disabling auto-launch of usbredir client
	{{ProgramName}} usbredir testvmi --no-launch
	`
}
