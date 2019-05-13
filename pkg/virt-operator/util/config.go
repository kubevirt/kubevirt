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

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

const (
	// Name of env var containing the operator's version
	OperatorVersionEnvName = "OPERATOR_VERSION"
	// Name of env var containing the operator's image name
	OperatorImageEnvName   = "OPERATOR_IMAGE"
	APIImageEnvName        = "API_IMAGE"
	ControllerImageEnvName = "CONTROLLER_IMAGE"
	LauncherImageEnvName   = "LAUNCHER_IMAGE"
	HandlerImageEnvName    = "HANDLER_IMAGE"
)

type KubeVirtDeploymentConfig struct {
	ImageRegistry string
	ImageTag      string

	Images v1.KubeVirtImages
}

func GetConfig() KubeVirtDeploymentConfig {
	registry := ""
	tag := os.Getenv(OperatorVersionEnvName)

	images := v1.KubeVirtImages{
		OperatorImage:   os.Getenv(OperatorImageEnvName),
		APIImage:        os.Getenv(APIImageEnvName),
		ControllerImage: os.Getenv(ControllerImageEnvName),
		LauncherImage:   os.Getenv(LauncherImageEnvName),
		HandlerImage:    os.Getenv(HandlerImageEnvName),
	}

	// FIXME: error if OPERATOR_IMAGE is not set

	if tag == "" || images.APIImage == "" || images.ControllerImage == "" || images.LauncherImage == "" || images.HandlerImage == "" {
		// FIXME: error if OPERATOR_IMAGE doesn't match the pattern
		imageRegEx := regexp.MustCompile(`^(.*)/virt-operator(:.*)?$`)
		matches := imageRegEx.FindAllStringSubmatch(images.OperatorImage, 1)
		registry = matches[0][1]
		if tag == "" {
			tag = strings.TrimPrefix(matches[0][2], ":")
			if tag == "" {
				tag = "latest"
			}
		}
	}

	// FIXME: require a tag to be found

	return KubeVirtDeploymentConfig{
		ImageRegistry: registry,
		ImageTag:      tag,
		Images:        images,
	}
}

func getConfigFromStatus(version string, registry string, images *v1.KubeVirtImages) *KubeVirtDeploymentConfig {
	return &KubeVirtDeploymentConfig{
		ImageRegistry: registry,
		ImageTag:      version,
		Images:        *images,
	}
}

func GetTargetConfigFromStatus(status *v1.KubeVirtStatus) *KubeVirtDeploymentConfig {
	return getConfigFromStatus(status.TargetKubeVirtVersion, status.TargetKubeVirtRegistry, &status.TargetKubeVirtImages)
}

func GetObservedConfigFromStatus(status *v1.KubeVirtStatus) *KubeVirtDeploymentConfig {
	return getConfigFromStatus(status.ObservedKubeVirtVersion, status.ObservedKubeVirtRegistry, &status.ObservedKubeVirtImages)
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

	operatorImage := spec.Images.OperatorImage
	if operatorImage == "" {
		operatorImage = fallback.Images.OperatorImage
	}

	APIImage := spec.Images.APIImage
	if APIImage == "" {
		APIImage = fallback.Images.APIImage
	}

	controllerImage := spec.Images.ControllerImage
	if controllerImage == "" {
		controllerImage = fallback.Images.ControllerImage
	}

	launcherImage := spec.Images.LauncherImage
	if launcherImage == "" {
		launcherImage = fallback.Images.LauncherImage
	}

	handlerImage := spec.Images.HandlerImage
	if handlerImage == "" {
		handlerImage = fallback.Images.HandlerImage
	}

	return &KubeVirtDeploymentConfig{
		ImageRegistry: registry,
		ImageTag:      tag,
		Images: v1.KubeVirtImages{
			OperatorImage:   operatorImage,
			APIImage:        APIImage,
			ControllerImage: controllerImage,
			LauncherImage:   launcherImage,
			HandlerImage:    handlerImage,
		},
	}
}

func SyncTargetStatus(status *v1.KubeVirtStatus, config *KubeVirtDeploymentConfig) {
	status.TargetKubeVirtVersion = config.ImageTag
	status.TargetKubeVirtRegistry = config.ImageRegistry
	status.TargetKubeVirtImages = config.Images
}

func SyncObservedStatus(status *v1.KubeVirtStatus, config *KubeVirtDeploymentConfig) {
	status.ObservedKubeVirtVersion = config.ImageTag
	status.ObservedKubeVirtRegistry = config.ImageRegistry
	status.ObservedKubeVirtImages = config.Images
}

func (conf *KubeVirtDeploymentConfig) GetOperatorImage() string {
	return conf.Images.OperatorImage
}

func (conf *KubeVirtDeploymentConfig) GetAPIImage() string {
	if conf.Images.APIImage == "" {
		return fmt.Sprintf("%s/virt-api:%s", conf.ImageRegistry, conf.ImageTag)
	}
	return conf.Images.APIImage
}

func (conf *KubeVirtDeploymentConfig) GetControllerImage() string {
	if conf.Images.ControllerImage == "" {
		return fmt.Sprintf("%s/virt-controller:%s", conf.ImageRegistry, conf.ImageTag)
	}
	return conf.Images.ControllerImage
}

func (conf *KubeVirtDeploymentConfig) GetLauncherImage() string {
	if conf.Images.LauncherImage == "" {
		return fmt.Sprintf("%s/virt-launcher:%s", conf.ImageRegistry, conf.ImageTag)
	}
	return conf.Images.LauncherImage
}

func (conf *KubeVirtDeploymentConfig) GetHandlerImage() string {
	if conf.Images.HandlerImage == "" {
		return fmt.Sprintf("%s/virt-handler:%s", conf.ImageRegistry, conf.ImageTag)
	}
	return conf.Images.HandlerImage
}

func (conf *KubeVirtDeploymentConfig) GetEnvVars() []k8sv1.EnvVar {
	return []k8sv1.EnvVar{
		{
			Name:  OperatorVersionEnvName,
			Value: conf.ImageTag,
		},
		{
			Name:  OperatorImageEnvName,
			Value: conf.Images.OperatorImage,
		},
		{
			Name:  APIImageEnvName,
			Value: conf.Images.APIImage,
		},
		{
			Name:  ControllerImageEnvName,
			Value: conf.Images.ControllerImage,
		},
		{
			Name:  LauncherImageEnvName,
			Value: conf.Images.LauncherImage,
		},
		{
			Name:  HandlerImageEnvName,
			Value: conf.Images.HandlerImage,
		},
	}
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
