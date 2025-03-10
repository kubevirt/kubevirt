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
	DefaultMinCPUModel = "Penryn"
	RequirePolicy      = "require"
	KVMPath            = "/dev/kvm"
	VmxFeature         = "vmx"
)

var DefaultObsoleteCPUModels = map[string]bool{
	"486":           true,
	"486-v1":        true,
	"pentium":       true,
	"pentium-v1":    true,
	"pentium2":      true,
	"pentium2-v1":   true,
	"pentium3":      true,
	"pentium3-v1":   true,
	"pentiumpro":    true,
	"pentiumpro-v1": true,
	"coreduo":       true,
	"coreduo-v1":    true,
	"n270":          true,
	"n270-v1":       true,
	"core2duo":      true,
	"core2duo-v1":   true,
	"Conroe":        true,
	"Conroe-v1":     true,
	"athlon":        true,
	"athlon-v1":     true,
	"phenom":        true,
	"phenom-v1":     true,
	"qemu64":        true,
	"qemu64-v1":     true,
	"qemu32":        true,
	"qemu32-v1":     true,
	"kvm64":         true,
	"kvm64-v1":      true,
	"kvm32":         true,
	"kvm32-v1":      true,
	"Opteron_G1":    true,
	"Opteron_G1-v1": true,
	"Opteron_G2":    true,
	"Opteron_G2-v1": true,
}

var DefaultArchitecturePrefix = map[string]string{
	"amd64": "x86_",
	"arm64": "arm_",
	"s390x": "s390x_",
}
