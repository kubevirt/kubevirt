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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v2"
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

func InstallVirtPlugin(cmd *cobra.Command) error {
	kubectlPluginPath, err := getPluginFolder()
	if err != nil {
		return err
	}

	// Create virt folder
	if err := os.MkdirAll(kubectlPluginPath, os.ModePerm); err != nil {
		return err
	}

	plugin := MakePluginConfiguration(kubectlPluginPath, cmd)

	if err := writePluginYaml(kubectlPluginPath, plugin); err != nil {
		return err
	}

	return copyVirtctlFile(kubectlPluginPath)
}

func getPluginFolder() (string, error) {

	if globalPluginPath, define := os.LookupEnv("KUBECTL_PLUGINS_PATH"); define {
		return filepath.Join(globalPluginPath, "virt"), nil
	}

	if xdgDataPath, define := os.LookupEnv("XDG_DATA_DIRS"); define {
		return filepath.Join(xdgDataPath, "kubectl", "plugins", "virt"), nil
	}

	if userHomeDir, define := os.LookupEnv("HOME"); define {
		return filepath.Join(userHomeDir, ".kube", "plugins", "virt"), nil
	}

	return "", fmt.Errorf("Fail to find kubernetes plugin folder")
}

func MakePluginConfiguration(kubectlPluginPath string, cmd *cobra.Command) *Plugin {
	tree := make(Plugins, 0)
	for _, command := range cmd.Commands() {
		if command.Name() != "install" && command.Name() != "options" {
			flags := make([]Flag, 0)

			checkFlags := func(f *pflag.Flag) {
				flags = append(flags, Flag{Name: f.Name, Desc: f.Usage, DefValue: f.DefValue})
			}

			command.Flags().VisitAll(checkFlags)
			tree = append(tree, &Plugin{Name: command.Name(), ShortDesc: command.Short, Command: fmt.Sprintf("%s %s", "./virtctl", command.Name()), Flags: flags})
		}
	}

	plugin := &Plugin{Name: "virt", ShortDesc: "kubevirt command plugin", Command: "./virtctl", Tree: tree}

	return plugin
}

func writePluginYaml(kubectlPluginPath string, plugin *Plugin) error {
	yamlData, err := yaml.Marshal(plugin)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(kubectlPluginPath, "plugin.yaml"), yamlData, 0644)
}

func copyVirtctlFile(kubectlPluginPath string) error {
	dst := filepath.Join(kubectlPluginPath, "virtctl")

	srcfd, err := os.Open(os.Args[0])
	if err != nil {
		return err
	}
	defer srcfd.Close()

	dstfd, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}

	srcinfo, err := os.Stat(os.Args[0])
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcinfo.Mode())
}
