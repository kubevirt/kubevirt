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

package util

import (
	"os"
	"regexp"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	kvutil "kubevirt.io/kubevirt/pkg/util"
)

const (
	// Name of env var containing the operator's image name
	OperatorImageEnvName        = "OPERATOR_IMAGE"
	VirtApiShasumEnvName        = "VIRT_API_SHASUM"
	VirtControllerShasumEnvName = "VIRT_CONTROLLER_SHASUM"
	VirtHandlerShasumEnvName    = "VIRT_HANDLER_SHASUM"
	VirtLauncherShasumEnvName   = "VIRT_LAUNCHER_SHASUM"
	KubeVirtVersionEnvName      = "VERSION"
	TargetInstallNamespace      = "TARGET_INSTALL_NAMESPACE"
	TargetImagePullPolicy       = "TARGET_IMAGE_PULL_POLICY"
)

type KubeVirtDeploymentConfig struct {
	ImageRegistry string
	Versions      *Versions
}

func GetTargetImagePullPolicy() k8sv1.PullPolicy {
	pullPolicy := os.Getenv(TargetImagePullPolicy)
	if pullPolicy == "" {
		return k8sv1.PullIfNotPresent
	}

	return k8sv1.PullPolicy(pullPolicy)
}

func GetTargetInstallNamespace() (string, error) {
	ns := os.Getenv(TargetInstallNamespace)
	if ns == "" {
		return kvutil.GetNamespace()
	}

	return ns, nil
}

func GetConfig() KubeVirtDeploymentConfig {

	// get registry and tag/shasum from operator image
	imageString := os.Getenv(OperatorImageEnvName)
	imageRegEx := regexp.MustCompile(`^(.*)/virt-operator([@:].*)?$`)
	matches := imageRegEx.FindAllStringSubmatch(imageString, 1)
	registry := matches[0][1]
	version := matches[0][2]
	tag := ""
	operatorSha := ""
	if version == "" {
		tag = "latest"
	} else if strings.HasPrefix(version, ":") {
		tag = strings.TrimPrefix(version, ":")
	} else {
		// we have a shasum... chances are high that we get the shasums for the other images as well from env vars,
		// but as a fallback use latest tag
		tag = "latest"
		operatorSha = strings.TrimPrefix(version, "@")
	}

	versions := NewVersionsWithTag(tag)

	// get shasums
	apiSha := os.Getenv(VirtApiShasumEnvName)
	controllerSha := os.Getenv(VirtControllerShasumEnvName)
	handlerSha := os.Getenv(VirtHandlerShasumEnvName)
	launcherSha := os.Getenv(VirtLauncherShasumEnvName)
	kubeVirtVersion := os.Getenv(KubeVirtVersionEnvName)
	if operatorSha != "" && apiSha != "" && controllerSha != "" && handlerSha != "" && launcherSha != "" && kubeVirtVersion != "" {
		versions = NewVersionsWithShasums(kubeVirtVersion, operatorSha, apiSha, controllerSha, handlerSha, launcherSha)
	}

	return KubeVirtDeploymentConfig{
		ImageRegistry: registry,
		Versions:      versions,
	}
}
