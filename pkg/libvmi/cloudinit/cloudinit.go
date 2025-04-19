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
 * Copyright The KubeVirt Authors.
 *
 */

package cloudinit

import (
	"encoding/base64"

	k8scorev1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

type NoCloudOption func(*v1.CloudInitNoCloudSource)

func WithNoCloudUserData(data string) NoCloudOption {
	return func(source *v1.CloudInitNoCloudSource) {
		source.UserData = data
	}
}

func WithNoCloudEncodedUserData(data string) NoCloudOption {
	return func(source *v1.CloudInitNoCloudSource) {
		source.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(data))
	}
}

func WithNoCloudUserDataSecretName(secretName string) NoCloudOption {
	return func(source *v1.CloudInitNoCloudSource) {
		source.UserDataSecretRef = &k8scorev1.LocalObjectReference{Name: secretName}
	}
}

func WithNoCloudNetworkData(data string) NoCloudOption {
	return func(source *v1.CloudInitNoCloudSource) {
		source.NetworkData = data
	}
}

func WithNoCloudEncodedNetworkData(data string) NoCloudOption {
	return func(source *v1.CloudInitNoCloudSource) {
		source.NetworkDataBase64 = base64.StdEncoding.EncodeToString([]byte(data))
	}
}

func WithNoCloudNetworkDataSecretName(secretName string) NoCloudOption {
	return func(source *v1.CloudInitNoCloudSource) {
		source.NetworkDataSecretRef = &k8scorev1.LocalObjectReference{Name: secretName}
	}
}

type ConfigDriveOption func(*v1.CloudInitConfigDriveSource)

func WithConfigDriveUserData(data string) ConfigDriveOption {
	return func(source *v1.CloudInitConfigDriveSource) {
		source.UserData = data
	}
}

func WithConfigDriveEncodedUserData(data string) ConfigDriveOption {
	return func(source *v1.CloudInitConfigDriveSource) {
		source.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(data))
	}
}

func WithConfigDriveNetworkData(data string) ConfigDriveOption {
	return func(source *v1.CloudInitConfigDriveSource) {
		source.NetworkData = data
	}
}

func WithConfigDriveEncodedNetworkData(data string) ConfigDriveOption {
	return func(source *v1.CloudInitConfigDriveSource) {
		source.NetworkDataBase64 = base64.StdEncoding.EncodeToString([]byte(data))
	}
}

func WithConfigDriveUserDataSecretName(secretName string) ConfigDriveOption {
	return func(source *v1.CloudInitConfigDriveSource) {
		source.UserDataSecretRef = &k8scorev1.LocalObjectReference{Name: secretName}
	}
}

func WithConfigDriveNetworkDataSecretName(secretName string) ConfigDriveOption {
	return func(source *v1.CloudInitConfigDriveSource) {
		source.NetworkDataSecretRef = &k8scorev1.LocalObjectReference{Name: secretName}
	}
}
