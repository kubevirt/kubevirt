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

package vm

import "k8s.io/apimachinery/pkg/api/resource"

type accessCredential struct {
	Type   string `param:"type"`
	Source string `param:"src"`
	Method string `param:"method"`
	User   string `param:"user"`
}

type volumeSource struct {
	Name      string `param:"name"`
	Source    string `param:"src"`
	BootOrder *uint  `param:"bootorder"`
}

type sysprepVolumeSource struct {
	Source string `param:"src"`
	Type   string `param:"type"`
}

type dataVolumeSource struct {
	Name      string             `param:"name"`
	Source    string             `param:"src"`
	Size      *resource.Quantity `param:"size"`
	Type      string             `param:"type"`
	BootOrder *uint              `param:"bootorder"`
}

type dataVolumeSourceBlank struct {
	Size      *resource.Quantity `param:"size"`
	Type      string             `param:"type"`
	Name      string             `param:"name"`
	BootOrder *uint              `param:"bootorder"`
}

type dataVolumeSourceGcs struct {
	SecretRef string             `param:"secretref"`
	URL       string             `param:"url"`
	Size      *resource.Quantity `param:"size"`
	Type      string             `param:"type"`
	Name      string             `param:"name"`
	BootOrder *uint              `param:"bootorder"`
}

type dataVolumeSourceHTTP struct {
	CertConfigMap      string             `param:"certconfigmap"`
	ExtraHeaders       []string           `param:"extraheaders"`
	SecretExtraHeaders []string           `param:"secretextraheaders"`
	SecretRef          string             `param:"secretref"`
	URL                string             `param:"url"`
	Size               *resource.Quantity `param:"size"`
	Type               string             `param:"type"`
	Name               string             `param:"name"`
	BootOrder          *uint              `param:"bootorder"`
}

type dataVolumeSourceImageIO struct {
	CertConfigMap string             `param:"certconfigmap"`
	DiskID        string             `param:"diskid"`
	SecretRef     string             `param:"secretref"`
	URL           string             `param:"url"`
	Size          *resource.Quantity `param:"size"`
	Type          string             `param:"type"`
	Name          string             `param:"name"`
	BootOrder     *uint              `param:"bootorder"`
}

type dataVolumeSourceRegistry struct {
	CertConfigMap string             `param:"certconfigmap"`
	ImageStream   string             `param:"imagestream"`
	PullMethod    string             `param:"pullmethod"`
	SecretRef     string             `param:"secretref"`
	URL           string             `param:"url"`
	Size          *resource.Quantity `param:"size"`
	Type          string             `param:"type"`
	Name          string             `param:"name"`
	BootOrder     *uint              `param:"bootorder"`
}

type dataVolumeSourceS3 struct {
	CertConfigMap string             `param:"certconfigmap"`
	SecretRef     string             `param:"secretref"`
	URL           string             `param:"url"`
	Size          *resource.Quantity `param:"size"`
	Type          string             `param:"type"`
	Name          string             `param:"name"`
	BootOrder     *uint              `param:"bootorder"`
}

type dataVolumeSourceVDDK struct {
	BackingFile  string             `param:"backingfile"`
	InitImageURL string             `param:"initimageurl"`
	SecretRef    string             `param:"secretref"`
	ThumbPrint   string             `param:"thumbprint"`
	URL          string             `param:"url"`
	UUID         string             `param:"uuid"`
	Size         *resource.Quantity `param:"size"`
	Type         string             `param:"type"`
	Name         string             `param:"name"`
	BootOrder    *uint              `param:"bootorder"`
}
