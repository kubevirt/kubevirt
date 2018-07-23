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
 * Copyright 2018 Red Hat, Inc.
 * Copyright 2018 The Kubernetes Authors.
 *
 */

package kubecli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Plugin holds everything needed to register a
// plugin as a command. Usually comes from a descriptor file.
type Plugin struct {
	Name      string  `json:"name"`
	ShortDesc string  `json:"shortDesc"`
	LongDesc  string  `json:"longDesc,omitempty"`
	Example   string  `json:"example,omitempty"`
	Command   string  `json:"command"`
	Flags     []Flag  `json:"flags,omitempty"`
	Tree      Plugins `json:"tree,omitempty"`
}

// Source holds the location of a given plugin in the filesystem.
type Source struct {
	Dir            string `json:"-"`
	DescriptorName string `json:"-"`
}

// Plugins is a list of plugins.
type Plugins []*Plugin

// Flag describes a single flag supported by a given plugin.
type Flag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Desc      string `json:"desc"`
	DefValue  string `json:"defValue,omitempty"`
}

func MakePluginConfiguration(cmd *cobra.Command) *Plugin {
	tree := make(Plugins, 0)
	for _, command := range cmd.Commands() {
		flags := make([]Flag, 0)

		checkFlags := func(f *pflag.Flag) {
			flags = append(flags, Flag{Name: f.Name})
		}
		command.Flags().VisitAll(checkFlags)
		tree = append(tree, &Plugin{Name: command.Name(), ShortDesc: command.Short, Command: command.Name(), Flags: flags})
	}

	plugin := &Plugin{Name: "virt", ShortDesc: "kubevirt command plugin", Command: "./virtctl", Tree: tree}

	return plugin
}

func WritePluginYaml(plugin *Plugin) error {
	return nil
}

func CopyVirtctlFile() error {
	return nil
}
