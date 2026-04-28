/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
