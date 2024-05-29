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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package libvmi

import (
	"encoding/base64"

	k8scorev1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

const CloudInitDiskName = "disk1"

// WithCloudInitVolume adds cloudInit volume and disk
func WithCloudInitVolume(volumeBuilder CloudInitBuilder) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addCloudInitDiskAndVolume(vmi, CloudInitDiskName, v1.DiskBusVirtio, volumeBuilder.build(CloudInitDiskName))
	}
}

// WithCloudInitNoCloudUserData adds cloud-init no-cloud user data.
func WithCloudInitNoCloudUserData(data string) Option {
	b := NewNoCloudResourceBuilder().WithUserData(data)
	return WithCloudInitVolume(b)
}

// WithCloudInitNoCloudEncodedUserData adds cloud-init no-cloud base64-encoded user data
func WithCloudInitNoCloudEncodedUserData(data string) Option {
	b := NewNoCloudResourceBuilder().WithEncodedData(data)
	return WithCloudInitVolume(b)
}

// WithCloudInitNoCloudNetworkData adds cloud-init no-cloud network data.
func WithCloudInitNoCloudNetworkData(data string) Option {
	b := NewNoCloudResourceBuilder().WithNetworkData(data)
	return WithCloudInitVolume(b)
}

// WithCloudInitNoCloudEncodedNetworkData adds cloud-init no-cloud base64 encoded network data.
func WithCloudInitNoCloudEncodedNetworkData(networkData string) Option {
	b := NewNoCloudResourceBuilder().WithNetworkEncodedData(networkData)
	return WithCloudInitVolume(b)
}

// WithCloudInitNoCloudNetworkDataSecretName adds cloud-init no-cloud network data from secret.
func WithCloudInitNoCloudNetworkDataSecretName(secretName string) Option {
	b := NewNoCloudResourceBuilder().WithNetworkSecretName(secretName)
	return WithCloudInitVolume(b)
}

// WithCloudInitConfigDriveUserData adds cloud-init config-drive user data.
func WithCloudInitConfigDriveUserData(data string) Option {
	b := NewConfigDriveResourceBuilder().WithUserData(data)
	return WithCloudInitVolume(b)
}

func addCloudInitDiskAndVolume(vmi *v1.VirtualMachineInstance, diskName string, bus v1.DiskBus, v v1.Volume) {
	addDisk(vmi, newDisk(diskName, bus))
	addVolume(vmi, v)
}

type CloudInitBuilder interface {
	WithUserData(string) CloudInitBuilder
	WithEncodedData(string) CloudInitBuilder
	WithSecretName(string) CloudInitBuilder
	WithNetworkData(string) CloudInitBuilder
	WithNetworkEncodedData(string) CloudInitBuilder
	WithNetworkSecretName(string) CloudInitBuilder
	build(name string) v1.Volume
}

type NoCloudResource struct {
	src *v1.CloudInitNoCloudSource
}

func NewNoCloudResourceBuilder() *NoCloudResource {
	return &NoCloudResource{src: &v1.CloudInitNoCloudSource{}}
}

func (r *NoCloudResource) WithUserData(data string) CloudInitBuilder {
	r.src.UserData = data
	return r
}

func (r *NoCloudResource) WithEncodedData(data string) CloudInitBuilder {
	r.src.UserDataBase64 = encodeData(data)
	return r
}

func (r *NoCloudResource) WithSecretName(name string) CloudInitBuilder {
	r.src.NetworkDataSecretRef = &k8scorev1.LocalObjectReference{
		Name: name,
	}
	return r
}

func (r *NoCloudResource) WithNetworkData(data string) CloudInitBuilder {
	r.src.NetworkData = data
	return r
}

func (r *NoCloudResource) WithNetworkEncodedData(data string) CloudInitBuilder {
	r.src.UserDataBase64 = encodeData(data)
	return r
}

func (r *NoCloudResource) WithNetworkSecretName(name string) CloudInitBuilder {
	r.src.NetworkDataSecretRef = &k8scorev1.LocalObjectReference{
		Name: name,
	}
	return r
}

func (r *NoCloudResource) build(name string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: r.src,
		},
	}
}

type ConfigDriveResource struct {
	src *v1.CloudInitConfigDriveSource
}

func NewConfigDriveResourceBuilder() *ConfigDriveResource {
	return &ConfigDriveResource{src: &v1.CloudInitConfigDriveSource{}}
}

func (r *ConfigDriveResource) WithUserData(data string) CloudInitBuilder {
	r.src.UserData = data
	return r
}

func (r *ConfigDriveResource) WithEncodedData(data string) CloudInitBuilder {
	r.src.UserDataBase64 = encodeData(data)
	return r
}

func (r *ConfigDriveResource) WithSecretName(name string) CloudInitBuilder {
	r.src.NetworkDataSecretRef = &k8scorev1.LocalObjectReference{
		Name: name,
	}
	return r
}

func (r *ConfigDriveResource) WithNetworkData(data string) CloudInitBuilder {
	r.src.NetworkData = data
	return r
}

func (r *ConfigDriveResource) WithNetworkEncodedData(data string) CloudInitBuilder {
	r.src.UserDataBase64 = encodeData(data)
	return r
}

func (r *ConfigDriveResource) WithNetworkSecretName(name string) CloudInitBuilder {
	r.src.NetworkDataSecretRef = &k8scorev1.LocalObjectReference{
		Name: name,
	}
	return r
}

func (r *ConfigDriveResource) build(name string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			CloudInitConfigDrive: r.src,
		},
	}
}

func encodeData(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}
