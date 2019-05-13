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
	"fmt"
	"os"
	"regexp"
	"strings"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

const (
	// Name of env var containing the operator's image name
	OperatorImageEnvName = "OPERATOR_IMAGE"
)

type KubeVirtDeploymentConfig struct {
	ImageRegistry string
	ImageTag      string
}

func GetConfig() KubeVirtDeploymentConfig {
	imageString := os.Getenv(OperatorImageEnvName)
	imageRegEx := regexp.MustCompile(`^(.*)/virt-operator(:.*)?$`)
	matches := imageRegEx.FindAllStringSubmatch(imageString, 1)
	registry := matches[0][1]
	tag := strings.TrimPrefix(matches[0][2], ":")
	if tag == "" {
		tag = "latest"
	}
	return KubeVirtDeploymentConfig{
		ImageRegistry: registry,
		ImageTag:      tag,
	}
}

func getConfigFromStatus(version string, registry string) *KubeVirtDeploymentConfig {
	return &KubeVirtDeploymentConfig{
		ImageRegistry: registry,
		ImageTag:      version,
	}
}

func GetTargetConfigFromStatus(status *v1.KubeVirtStatus) *KubeVirtDeploymentConfig {
	return getConfigFromStatus(status.TargetKubeVirtVersion, status.TargetKubeVirtRegistry)
}

func GetObservedConfigFromStatus(status *v1.KubeVirtStatus) *KubeVirtDeploymentConfig {
	return getConfigFromStatus(status.ObservedKubeVirtVersion, status.ObservedKubeVirtRegistry)
}

func GetConfigFromSpec(spec *v1.KubeVirtSpec, fallback *KubeVirtDeploymentConfig) *KubeVirtDeploymentConfig {
	registry := spec.ImageRegistry
	if registry == "" {
		registry = fallback.ImageRegistry
	}

	tag := spec.ImageTag
	if tag == "" {
		tag = fallback.ImageTag
	}
	return &KubeVirtDeploymentConfig{
		ImageRegistry: registry,
		ImageTag:      tag,
	}
}

func SyncTargetStatus(status *v1.KubeVirtStatus, config *KubeVirtDeploymentConfig) {
	status.TargetKubeVirtVersion = config.ImageTag
	status.TargetKubeVirtRegistry = config.ImageRegistry
}

func SyncObservedStatus(status *v1.KubeVirtStatus, config *KubeVirtDeploymentConfig) {
	status.ObservedKubeVirtVersion = config.ImageTag
	status.ObservedKubeVirtRegistry = config.ImageRegistry
}

func (conf *KubeVirtDeploymentConfig) GetOperatorImage() string {
	return fmt.Sprintf("%s/virt-operator:%s", conf.ImageRegistry, conf.ImageTag)
}

func (conf *KubeVirtDeploymentConfig) GetAPIImage() string {
	return fmt.Sprintf("%s/virt-api:%s", conf.ImageRegistry, conf.ImageTag)
}

func (conf *KubeVirtDeploymentConfig) GetControllerImage() string {
	return fmt.Sprintf("%s/virt-controller:%s", conf.ImageRegistry, conf.ImageTag)
}

func (conf *KubeVirtDeploymentConfig) GetLauncherImage() string {
	return fmt.Sprintf("%s/virt-launcher:%s", conf.ImageRegistry, conf.ImageTag)
}

func (conf *KubeVirtDeploymentConfig) GetHandlerImage() string {
	return fmt.Sprintf("%s/virt-handler:%s", conf.ImageRegistry, conf.ImageTag)
}

func (conf *KubeVirtDeploymentConfig) String() string {
	return fmt.Sprintf("%s/%s", conf.ImageRegistry, conf.ImageTag)
}

func (conf *KubeVirtDeploymentConfig) GetMapKey() string {
	return conf.String()
}

func (conf *KubeVirtDeploymentConfig) AddAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[v1.InstallStrategyVersionAnnotation] = conf.ImageTag
	annotations[v1.InstallStrategyRegistryAnnotation] = conf.ImageRegistry
	return annotations
}

func (conf *KubeVirtDeploymentConfig) MatchesAnnotations(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}

	tagAnno, ok := annotations[v1.InstallStrategyVersionAnnotation]
	if !ok {
		return false
	}

	registryAnno, ok := annotations[v1.InstallStrategyRegistryAnnotation]
	if !ok {
		return false
	}

	return tagAnno == conf.ImageTag && registryAnno == conf.ImageRegistry
}
