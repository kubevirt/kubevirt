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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package util

const (
	DeprecatedLabelNamespace              string = "feature.node.kubernetes.io"
	DeprecatedLabellerNamespaceAnnotation        = "node-labeller-feature.node.kubernetes.io"
	DeprecatedcpuFeaturePrefix                   = "/cpu-feature-"
	DeprecatedcpuModelPrefix                     = "/cpu-model-"
	DeprecatedHyperPrefix                        = "/kvm-info-cap-hyperv-"
	DefaultMinCPUModel                           = "Penryn"
	RequirePolicy                                = "require"
	KVMPath                                      = "/dev/kvm"
	VmxFeature                                   = "vmx"
)

var DefaultObsoleteCPUModels = map[string]bool{
	"486":        true,
	"pentium":    true,
	"pentium2":   true,
	"pentium3":   true,
	"pentiumpro": true,
	"coreduo":    true,
	"n270":       true,
	"core2duo":   true,
	"Conroe":     true,
	"athlon":     true,
	"phenom":     true,
	"qemu64":     true,
	"qemu32":     true,
	"kvm64":      true,
	"kvm32":      true,
}

var DefaultArchitecturePrefix = map[string]string{
	"amd64": "x86_",
	"arm64": "arm_",
}
