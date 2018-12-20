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
 *
 */

package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/pflag"
)

type templateData struct {
	Namespace          string
	CDINamespace       string
	DockerTag          string
	DockerPrefix       string
	ImagePullPolicy    string
	GeneratedManifests map[string]string
	DevManifests       map[string]string
}

func main() {
	namespace := flag.String("namespace", "", "")
	cdiNamespace := flag.String("cdi-namespace", "", "")
	dockerPrefix := flag.String("container-prefix", "", "")
	dockerTag := flag.String("container-tag", "", "")
	imagePullPolicy := flag.String("image-pull-policy", "IfNotPresent", "")
	genDir := flag.String("generated-manifests-dir", "", "")
	devDir := flag.String("dev-manifests-dir", "", "")
	inputFile := flag.String("input-file", "", "")
	processFiles := flag.Bool("process-files", false, "")
	processVars := flag.Bool("process-vars", false, "")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
	pflag.Parse()

	if !(*processFiles || *processVars) {
		panic("at least one of process-files or process-vars must be true")
	}

	data := templateData{
		GeneratedManifests: make(map[string]string),
		DevManifests:       make(map[string]string),
	}

	if *processVars {
		data.Namespace = *namespace
		data.CDINamespace = *cdiNamespace
		data.DockerTag = *dockerTag
		data.DockerPrefix = *dockerPrefix
		data.ImagePullPolicy = *imagePullPolicy
	} else {
		// keep templates
		data.Namespace = "{{.Namespace}}"
		data.CDINamespace = "{{.CDINamespace}}"
		data.DockerTag = "{{.DockerTag}}"
		data.DockerPrefix = "{{.DockerPrefix}}"
		data.ImagePullPolicy = "{{.ImagePullPolicy}}"
	}

	if *processFiles {
		manifests, err := ioutil.ReadDir(*genDir)
		if err != nil {
			panic(err)
		}

		for _, manifest := range manifests {
			if manifest.IsDir() {
				continue
			}
			b, err := ioutil.ReadFile(filepath.Join(*genDir, manifest.Name()))
			if err != nil {
				panic(err)
			}
			data.GeneratedManifests[manifest.Name()] = string(b)
		}

		manifests, err = ioutil.ReadDir(*devDir)
		if err != nil {
			panic(err)
		}

		for _, manifest := range manifests {
			if manifest.IsDir() {
				continue
			}
			b, err := ioutil.ReadFile(filepath.Join(*devDir, manifest.Name()))
			if err != nil {
				panic(err)
			}
			data.DevManifests[manifest.Name()] = string(b)
		}
	}

	tmpl := template.Must(template.ParseFiles(*inputFile))
	err := tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}
