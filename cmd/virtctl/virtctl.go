/*
 * This file is part of the kubevirt project
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
	"os"

	"fmt"
	"log"

	flag "github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/virtctl"
	"kubevirt.io/kubevirt/pkg/virtctl/console"
	"kubevirt.io/kubevirt/pkg/virtctl/convert"
	"kubevirt.io/kubevirt/pkg/virtctl/spice"
)

func main() {

	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	registry := map[string]virtctl.App{
		"console":      &console.Console{},
		"options":      &virtctl.Options{},
		"spice":        &spice.Spice{},
		"convert-spec": convert.NewConvertCommand(),
	}

	if len(os.Args) > 1 {
		for cmd, app := range registry {
			f := app.FlagSet()
			f.Bool("help", false, "Print usage.")
			f.MarkHidden("help")
			f.Usage = func() {
				fmt.Fprint(os.Stderr, app.Usage())
			}

			if os.Args[1] != cmd {
				continue
			}
			flags, err := Parse(f)

			h, _ := flags.GetBool("help")
			if h || err != nil {
				f.Usage()
				return
			}
			os.Exit(app.Run(flags))
		}
	}

	Usage()
	os.Exit(1)
}

func Parse(flags *flag.FlagSet) (*flag.FlagSet, error) {
	flags.AddFlagSet((&virtctl.Options{}).FlagSet())
	err := flags.Parse(os.Args[1:])
	return flags, err
}

func Usage() {
	fmt.Fprintln(os.Stderr,
		`virtctl controls VM related operations on your kubernetes cluster.

Basic Commands:
  console        Connect to a serial console on a VM
  convert-spec   Convert between Libvirt and KubeVirt specifications

Use "virtctl <command> --help" for more information about a given command.
Use "virtctl options" for a list of global command-line options (applies to all commands).
	`)
}
