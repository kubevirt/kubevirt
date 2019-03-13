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
