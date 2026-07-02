package util

import "regexp"

// Pattern: virt-launcher-<vmi-name>-<pod-hash>-<template-name>-<claim-hash>
// Captures: <vmi-name> and <template-name> (excluding the two 5-char hashes).
// Pod hash and claim hash are both exactly 5 lowercase alphanumeric characters.
var stableClaimNameRegex = regexp.MustCompile(`^virt-launcher-(.+?)-[a-z0-9]{5}-(.+)-[a-z0-9]{5}$`)

// ExtractStableClaimName extracts the migration-stable portion of a ResourceClaim name.
// For KubeVirt claims, the format is: virt-launcher-<vmi-name>-<pod-hash>-<template-name>-<claim-hash>
// We extract: <vmi-name>-<template-name> (removing both 5-char hashes).
// Example: "virt-launcher-vm-a-drz4j-dummy-gpu-fngjv" -> "vm-a-dummy-gpu".
func ExtractStableClaimName(fullClaimName string) string {
	matches := stableClaimNameRegex.FindStringSubmatch(fullClaimName)
	if matches == nil {
		// Not a virt-launcher claim or doesn't match expected pattern, use full name
		return fullClaimName
	}
	// matches[1] = <vmi-name>, matches[2] = <template-name>
	return matches[1] + "-" + matches[2]
}
