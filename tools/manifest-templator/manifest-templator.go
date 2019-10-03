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
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/pflag"

	v1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
	"kubevirt.io/kubevirt/tools/marketplace/helper"
	"kubevirt.io/kubevirt/tools/util"
)

type templateData struct {
	Namespace              string
	CDINamespace           string
	CSVNamespace           string
	DockerTag              string
	DockerPrefix           string
	ImagePrefix            string
	ImagePullPolicy        string
	Verbosity              string
	CsvVersion             string
	QuayRepository         string
	ReplacesCsvVersion     string
	OperatorDeploymentSpec string
	OperatorCsv            string
	OperatorRules          string
	KubeVirtLogo           string
	PackageName            string
	CreatedAt              string
	VirtOperatorSha        string
	VirtApiSha             string
	VirtControllerSha      string
	VirtHandlerSha         string
	VirtLauncherSha        string
	GeneratedManifests     map[string]string
}

func main() {
	namespace := flag.String("namespace", "", "")
	csvNamespace := flag.String("csv-namespace", "placeholder", "")
	cdiNamespace := flag.String("cdi-namespace", "", "")
	dockerPrefix := flag.String("container-prefix", "", "")
	imagePrefix := flag.String("image-prefix", "", "")
	dockerTag := flag.String("container-tag", "", "")
	csvVersion := flag.String("csv-version", "", "")
	imagePullPolicy := flag.String("image-pull-policy", "IfNotPresent", "")
	verbosity := flag.String("verbosity", "2", "")
	genDir := flag.String("generated-manifests-dir", "", "")
	inputFile := flag.String("input-file", "", "")
	processFiles := flag.Bool("process-files", false, "")
	processVars := flag.Bool("process-vars", false, "")
	kubeVirtLogoPath := flag.String("kubevirt-logo-path", "", "")
	packageName := flag.String("package-name", "", "")
	bundleOutDir := flag.String("bundle-out-dir", "", "")
	quayRepository := flag.String("quay-repository", "", "")
	virtOperatorSha := flag.String("virt-operator-sha", "", "")
	virtApiSha := flag.String("virt-api-sha", "", "")
	virtControllerSha := flag.String("virt-controller-sha", "", "")
	virtHandlerSha := flag.String("virt-handler-sha", "", "")
	virtLauncherSha := flag.String("virt-launcher-sha", "", "")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
	pflag.Parse()

	if !(*processFiles || *processVars) {
		panic("at least one of process-files or process-vars must be true")
	}

	data := templateData{
		GeneratedManifests: make(map[string]string),
	}

	if *processVars {
		data.Namespace = *namespace
		data.CSVNamespace = *csvNamespace
		data.CDINamespace = *cdiNamespace
		data.DockerTag = *dockerTag
		data.DockerPrefix = *dockerPrefix
		data.ImagePrefix = *imagePrefix
		data.ImagePullPolicy = *imagePullPolicy
		data.Verbosity = fmt.Sprintf("\"%s\"", *verbosity)
		data.CsvVersion = *csvVersion
		data.QuayRepository = *quayRepository
		data.VirtOperatorSha = *virtOperatorSha
		data.VirtApiSha = *virtApiSha
		data.VirtControllerSha = *virtControllerSha
		data.VirtHandlerSha = *virtHandlerSha
		data.VirtLauncherSha = *virtLauncherSha
		data.OperatorRules = getOperatorRules()
		data.KubeVirtLogo = getKubeVirtLogo(*kubeVirtLogoPath)
		data.PackageName = *packageName
		data.CreatedAt = getTimestamp()
		data.ReplacesCsvVersion = ""
		data.OperatorDeploymentSpec = getOperatorDeploymentSpec(data, 2)

		// operator deployment differs a bit in normal manifest and CSV
		if strings.Contains(*inputFile, ".clusterserviceversion.yaml") {
			// prevent loading latest bundle from Quay for every file, only do it for the CSV manifest
			if *bundleOutDir != "" && data.QuayRepository != "" {
				bundleHelper, err := helper.NewBundleHelper(*quayRepository, *packageName)
				if err != nil {
					panic(err)
				}
				latestVersion := bundleHelper.GetLatestPublishedCSVVersion()
				if latestVersion != "" {
					// prevent generating the same version again
					if strings.HasSuffix(latestVersion, *csvVersion) {
						panic(fmt.Errorf("CSV version %s is already published!", *csvVersion))
					}
					data.ReplacesCsvVersion = latestVersion
					// also copy old manifests to out dir
					bundleHelper.AddOldManifests(*bundleOutDir, *csvVersion)
				}
			}
		}

	} else {
		// keep templates
		data.Namespace = "{{.Namespace}}"
		data.CDINamespace = "{{.CDINamespace}}"
		data.DockerTag = "{{.DockerTag}}"
		data.DockerPrefix = "{{.DockerPrefix}}"
		data.ImagePrefix = "{{.ImagePrefix}}"
		data.ImagePullPolicy = "{{.ImagePullPolicy}}"
		data.Verbosity = "{{.Verbosity}}"
		data.CsvVersion = "{{.CsvVersion}}"
		data.QuayRepository = "{{.QuayRepository}}"
		data.VirtOperatorSha = "{{.VirtOperatorSha}}"
		data.VirtApiSha = "{{.VirtApiSha}}"
		data.VirtControllerSha = "{{.VirtControllerSha}}"
		data.VirtHandlerSha = "{{.VirtHandlerSha}}"
		data.VirtLauncherSha = "{{.VirtLauncherSha}}"
		data.ReplacesCsvVersion = "{{.ReplacesCsvVersion}}"
		data.OperatorDeploymentSpec = "{{.OperatorDeploymentSpec}}"
		data.OperatorCsv = "{{.OperatorCsv}}"
		data.OperatorRules = "{{.OperatorRules}}"
		data.KubeVirtLogo = "{{.KubeVirtLogo}}"
		data.PackageName = "{{.PackageName}}"
		data.CreatedAt = "{{.CreatedAt}}"
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
	}

	tmpl := template.Must(template.ParseFiles(*inputFile))
	err := tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

func getOperatorRules() string {
	rules := rbac.NewOperatorClusterRole().Rules
	writer := strings.Builder{}
	for _, rule := range rules {
		err := util.MarshallObject(rule, &writer)
		if err != nil {
			panic(err)
		}
	}
	return fixResourceString(writer.String(), 14)
}

func getOperatorDeploymentSpec(data templateData, indentation int) string {
	version := data.DockerTag
	if data.VirtOperatorSha != "" {
		version = data.VirtOperatorSha
	}

	deployment, err := components.NewOperatorDeployment(
		data.Namespace,
		data.DockerPrefix,
		data.ImagePrefix,
		version,
		v1.PullPolicy(data.ImagePullPolicy),
		data.Verbosity,
		data.DockerTag,
		data.VirtApiSha,
		data.VirtControllerSha,
		data.VirtHandlerSha,
		data.VirtLauncherSha)
	if err != nil {
		panic(err)
	}

	writer := strings.Builder{}
	err = util.MarshallObject(deployment.Spec, &writer)
	if err != nil {
		panic(err)
	}
	spec := writer.String()

	return fixResourceString(spec, indentation)
}

func fixResourceString(in string, indention int) string {
	out := strings.Builder{}
	scanner := bufio.NewScanner(strings.NewReader(in))
	for scanner.Scan() {
		line := scanner.Text()
		// remove separator lines
		if !strings.HasPrefix(line, "---") {
			// indent so that it fits into the manifest
			// spaces is is indention - 2, because we want to have 2 spaces less for being able to start an array
			spaces := strings.Repeat(" ", indention-2)
			if strings.HasPrefix(line, "apiGroups") {
				// spaces + array start
				out.WriteString(spaces + "- " + line + "\n")
			} else {
				// 2 more spaces
				out.WriteString(spaces + "  " + line + "\n")
			}
		}
	}
	return out.String()
}

func getKubeVirtLogo(path string) string {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	// Read entire file into byte slice.
	reader := bufio.NewReader(file)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	// Encode as base64.
	encoded := base64.StdEncoding.EncodeToString(content)
	return encoded
}

func getTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}
