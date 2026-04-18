/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package domainstats

const (
	nanosecondsPerSecond float64 = 1_000_000_000
	bytesPerKibibyte     float64 = 1024
)

func nanosecondsToSeconds(ns uint64) float64 {
	return float64(ns) / nanosecondsPerSecond
}

func kibibytesToBytes(kibibytes uint64) float64 {
	return float64(kibibytes) * bytesPerKibibyte
}
