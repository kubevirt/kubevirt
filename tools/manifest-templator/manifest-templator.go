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
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/pflag"

	v1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
	"kubevirt.io/kubevirt/tools/util"
)

const (
	customImageExample   = "Examples: some.registry.com@sha256:abcdefghijklmnop, other.registry.com:tag1"
	shaEnvDeprecationMsg = "This argument is deprecated. Please use virt-*-image instead"
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
	VirtExportProxySha     string
	VirtExportServerSha    string
	GsSha                  string
	PrHelperSha            string
	RunbookURLTemplate     string
	PriorityClassSpec      string
	FeatureGates           []string
	InfraReplicas          uint8
	GeneratedManifests     map[string]string
	VirtOperatorImage      string
	VirtApiImage           string
	VirtControllerImage    string
	VirtHandlerImage       string
	VirtLauncherImage      string
	VirtExportProxyImage   string
	VirtExportServerImage  string
	GsImage                string
	PrHelperImage          string
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
	quayRepository := flag.String("quay-repository", "", "")
	virtOperatorSha := flag.String("virt-operator-sha", "", shaEnvDeprecationMsg)
	virtApiSha := flag.String("virt-api-sha", "", shaEnvDeprecationMsg)
	virtControllerSha := flag.String("virt-controller-sha", "", shaEnvDeprecationMsg)
	virtHandlerSha := flag.String("virt-handler-sha", "", shaEnvDeprecationMsg)
	virtLauncherSha := flag.String("virt-launcher-sha", "", shaEnvDeprecationMsg)
	virtExportProxySha := flag.String("virt-exportproxy-sha", "", shaEnvDeprecationMsg)
	virtExportServerSha := flag.String("virt-exportserver-sha", "", shaEnvDeprecationMsg)
	gsSha := flag.String("gs-sha", "", "")
	prHelperSha := flag.String("pr-helper-sha", "", "")
	runbookURLTemplate := flag.String("runbook-url-template", "", "")
	featureGates := flag.String("feature-gates", "", "")
	infraReplicas := flag.Uint("infra-replicas", 0, "")
	virtOperatorImage := flag.String("virt-operator-image", "", "custom image for virt-operator")
	virtApiImage := flag.String("virt-api-image", "", "custom image for virt-api. "+customImageExample)
	virtControllerImage := flag.String("virt-controller-image", "", "custom image for virt-controller. "+customImageExample)
	virtHandlerImage := flag.String("virt-handler-image", "", "custom image for virt-handler. "+customImageExample)
	virtLauncherImage := flag.String("virt-launcher-image", "", "custom image for virt-launcher. "+customImageExample)
	virtExportProxyImage := flag.String("virt-export-proxy-image", "", "custom image for virt-export-proxy. "+customImageExample)
	virtExportServerImage := flag.String("virt-export-server-image", "", "custom image for virt-export-server. "+customImageExample)
	gsImage := flag.String("gs-image", "", "custom image for gs. "+customImageExample)
	prHelperImage := flag.String("pr-helper-image", "", "custom image for pr-helper. "+customImageExample)

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
	pflag.Parse()

	if path := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); path != "" {
		if err := os.Chdir(path); err != nil {
			panic(err)
		}
	}

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
		data.VirtExportProxySha = *virtExportProxySha
		data.VirtExportServerSha = *virtExportServerSha
		data.GsSha = *gsSha
		data.PrHelperSha = *prHelperSha
		data.RunbookURLTemplate = *runbookURLTemplate
		data.OperatorRules = getOperatorRules()
		data.KubeVirtLogo = getKubeVirtLogo(*kubeVirtLogoPath)
		data.PackageName = *packageName
		data.CreatedAt = getTimestamp()
		data.ReplacesCsvVersion = ""
		data.OperatorDeploymentSpec = getOperatorDeploymentSpec(data, 2)
		data.PriorityClassSpec = getPriorityClassSpec(2)
		data.VirtOperatorImage = *virtOperatorImage
		data.VirtApiImage = *virtApiImage
		data.VirtControllerImage = *virtControllerImage
		data.VirtHandlerImage = *virtHandlerImage
		data.VirtLauncherImage = *virtLauncherImage
		data.VirtExportProxyImage = *virtExportProxyImage
		data.VirtExportServerImage = *virtExportServerImage
		data.GsImage = *gsImage
		data.PrHelperImage = *prHelperImage
		if *featureGates != "" {
			data.FeatureGates = strings.Split(*featureGates, ",")
		}
		if *infraReplicas != 0 {
			data.InfraReplicas = uint8(*infraReplicas)
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
		data.VirtExportProxySha = "{{.VirtExportProxySha}}"
		data.VirtExportServerSha = "{{.VirtExportServerSha}}"
		data.ReplacesCsvVersion = "{{.ReplacesCsvVersion}}"
		data.OperatorDeploymentSpec = "{{.OperatorDeploymentSpec}}"
		data.OperatorCsv = "{{.OperatorCsv}}"
		data.OperatorRules = "{{.OperatorRules}}"
		data.KubeVirtLogo = "{{.KubeVirtLogo}}"
		data.PackageName = "{{.PackageName}}"
		data.CreatedAt = "{{.CreatedAt}}"
		data.VirtApiImage = "{{.VirtApiImage}}"
		data.VirtControllerImage = "{{.VirtControllerImage}}"
		data.VirtHandlerImage = "{{.VirtHandlerImage}}"
		data.VirtLauncherImage = "{{.VirtLauncherImage}}"
		data.VirtExportProxyImage = "{{.VirtExportProxyImage}}"
		data.VirtExportServerImage = "{{.VirtExportServerImage}}"
		data.GsImage = "{{.GsImage}}"
		data.PrHelperImage = "{{.PrHelperImage}}"
	}

	if *processFiles {
		manifests, err := os.ReadDir(*genDir)
		if err != nil {
			panic(err)
		}

		for _, manifest := range manifests {
			if manifest.IsDir() {
				continue
			}
			b, err := os.ReadFile(filepath.Join(*genDir, manifest.Name()))
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

func getPriorityClassSpec(indentation int) string {
	priorityClassSpec := components.NewKubeVirtPriorityClassCR()
	writer := strings.Builder{}
	err := util.MarshallObject(priorityClassSpec, &writer)
	if err != nil {
		panic(err)
	}
	return fixResourceString(writer.String(), indentation)
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
		data.Verbosity,
		data.DockerTag,
		data.VirtApiSha,
		data.VirtControllerSha,
		data.VirtHandlerSha,
		data.VirtLauncherSha,
		data.VirtExportProxySha,
		data.VirtExportServerSha,
		data.GsSha,
		data.PrHelperSha,
		data.RunbookURLTemplate,
		data.VirtApiImage,
		data.VirtControllerImage,
		data.VirtHandlerImage,
		data.VirtLauncherImage,
		data.VirtExportProxyImage,
		data.VirtExportServerImage,
		data.GsImage,
		data.PrHelperImage,
		data.VirtOperatorImage,
		v1.PullPolicy(data.ImagePullPolicy))
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
	content, err := io.ReadAll(reader)
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
