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
 * Copyright 2021 IBM, Inc.
 *
 */

package flags

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
)

var (
	Kubeconfig         string
	Kubemaster         string
	Verbosity          int
	WorkloadConfigFile string
	ContainerPrefix    string
	ContainerTag       string
)

func init() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flag.StringVar(&Kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&Kubemaster, "master", "", "kubernetes master url")
	flag.IntVar(&Verbosity, "v", 2, "log level for V logs")
	flag.StringVar(&WorkloadConfigFile, "workload", "tools/perfscale-load-generator/examples/workload/kubevirt-density/kubevirt-density.yaml", "path to the file containing the worload configuration")
	flag.StringVar(&ContainerPrefix, "container-prefix", "registry:5000/kubevirt/", "Set the repository prefix for all images")
	flag.StringVar(&ContainerTag, "container-tag", "devel", "Set the image tag or digest to use")

	if Kubeconfig == "" {
		if os.Getenv("KUBECONFIG") != "" {
			Kubeconfig = os.Getenv("KUBECONFIG")
		} else {
			_, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".kube", "config"))
			if !os.IsNotExist(err) {
				Kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
			}
		}
	}

	flag.Parse()
}

// GetRootConfigDir returns the path of the directory of the config file
func GetRootConfigDir() string {
	parts := strings.Split(WorkloadConfigFile, "/")
	return strings.Join(parts[0:len(parts)-1], "/")
}
