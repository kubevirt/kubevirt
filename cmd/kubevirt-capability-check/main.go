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

package main

import (
	goflag "flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/capabilities"
)

func main() {
	hypervisor := pflag.String("hypervisor", "", "Target hypervisor (e.g., 'kvm', 'mshv')")
	arch := pflag.String("arch", "", "Target architecture (e.g., 'amd64', 'arm64', 's390x')")
	supportLevel := pflag.String("support-level", "Unsupported", "Support level to filter by (Unsupported, Experimental, Deprecated, Unregistered)")
	outputFormat := pflag.String("output", "keys", "Output format: 'keys' (capability keys only), 'detailed' (keys with messages), 'json'")
	listAll := pflag.Bool("list-all", false, "List all capabilities regardless of support level")

	pflag.CommandLine.AddGoFlag(goflag.CommandLine.Lookup("v"))
	pflag.Parse()

	log.InitializeLogging("kubevirt-capability-check")

	// Validate required parameters
	if *hypervisor == "" {
		log.Log.Error("--hypervisor parameter is required")
		printUsage()
		os.Exit(1)
	}

	if *arch == "" {
		log.Log.Error("--arch parameter is required")
		printUsage()
		os.Exit(1)
	}

	// Initialize capabilities
	capabilities.Init()

	// Get capabilities support for the specified platform
	platformSupports := capabilities.GetCapabilitiesSupportForPlatform(*hypervisor, *arch)

	// Filter capabilities by support level
	var targetLevel capabilities.SupportLevel
	if !*listAll {
		switch strings.ToLower(*supportLevel) {
		case "unsupported":
			targetLevel = capabilities.Unsupported
		case "experimental":
			targetLevel = capabilities.Experimental
		case "deprecated":
			targetLevel = capabilities.Deprecated
		case "unregistered":
			targetLevel = capabilities.Unregistered
		default:
			log.Log.Errorf("Invalid support level: %s", *supportLevel)
			printUsage()
			os.Exit(1)
		}
	}

	// Collect matching capabilities
	var matchingCapabilities []capabilityInfo

	if *listAll {
		// Include all registered capabilities
		for capKey, capSupport := range platformSupports {
			matchingCapabilities = append(matchingCapabilities, capabilityInfo{
				Key:     string(capKey),
				Level:   capSupport.Level,
				Message: capSupport.Message,
				GatedBy: capSupport.GatedBy,
			})
		}

		// Also include capabilities that are registered but not in platform supports (Unregistered)
		for capKey := range capabilities.CapabilityDefinitions {
			if _, exists := platformSupports[capKey]; !exists {
				matchingCapabilities = append(matchingCapabilities, capabilityInfo{
					Key:     string(capKey),
					Level:   capabilities.Unregistered,
					Message: "Not explicitly registered for this platform",
					GatedBy: "",
				})
			}
		}
	} else {
		// Filter by specific support level
		for capKey, capSupport := range platformSupports {
			if capSupport.Level == targetLevel {
				matchingCapabilities = append(matchingCapabilities, capabilityInfo{
					Key:     string(capKey),
					Level:   capSupport.Level,
					Message: capSupport.Message,
					GatedBy: capSupport.GatedBy,
				})
			}
		}

		// For Unregistered level, also check capabilities not in platform supports
		if targetLevel == capabilities.Unregistered {
			for capKey := range capabilities.CapabilityDefinitions {
				if _, exists := platformSupports[capKey]; !exists {
					matchingCapabilities = append(matchingCapabilities, capabilityInfo{
						Key:     string(capKey),
						Level:   capabilities.Unregistered,
						Message: "Not explicitly registered for this platform",
						GatedBy: "",
					})
				}
			}
		}
	}

	// Sort capabilities by key for consistent output
	sort.Slice(matchingCapabilities, func(i, j int) bool {
		return matchingCapabilities[i].Key < matchingCapabilities[j].Key
	})

	// Output results
	switch strings.ToLower(*outputFormat) {
	case "keys":
		outputKeys(matchingCapabilities)
	case "detailed":
		outputDetailed(matchingCapabilities, *hypervisor, *arch)
	case "json":
		outputJSON(matchingCapabilities, *hypervisor, *arch)
	default:
		log.Log.Errorf("Invalid output format: %s", *outputFormat)
		printUsage()
		os.Exit(1)
	}
}

type capabilityInfo struct {
	Key     string                    `json:"key"`
	Level   capabilities.SupportLevel `json:"level"`
	Message string                    `json:"message,omitempty"`
	GatedBy string                    `json:"gatedBy,omitempty"`
}

func outputKeys(caps []capabilityInfo) {
	for _, cap := range caps {
		fmt.Println(cap.Key)
	}
}

func outputDetailed(caps []capabilityInfo, hypervisor, arch string) {
	fmt.Printf("Capabilities for platform %s/%s:\n", hypervisor, arch)
	fmt.Println()

	for _, cap := range caps {
		fmt.Printf("Key: %s\n", cap.Key)
		fmt.Printf("  Level: %s\n", supportLevelToString(cap.Level))
		if cap.Message != "" {
			fmt.Printf("  Message: %s\n", cap.Message)
		}
		if cap.GatedBy != "" {
			fmt.Printf("  Feature Gate: %s\n", cap.GatedBy)
		}
		fmt.Println()
	}
}

func outputJSON(caps []capabilityInfo, hypervisor, arch string) {
	result := struct {
		Platform     string           `json:"platform"`
		Hypervisor   string           `json:"hypervisor"`
		Architecture string           `json:"architecture"`
		Capabilities []capabilityInfo `json:"capabilities"`
	}{
		Platform:     fmt.Sprintf("%s/%s", hypervisor, arch),
		Hypervisor:   hypervisor,
		Architecture: arch,
		Capabilities: caps,
	}

	// Simple JSON output without external dependencies
	fmt.Printf("{\n")
	fmt.Printf("  \"platform\": \"%s\",\n", result.Platform)
	fmt.Printf("  \"hypervisor\": \"%s\",\n", result.Hypervisor)
	fmt.Printf("  \"architecture\": \"%s\",\n", result.Architecture)
	fmt.Printf("  \"capabilities\": [\n")

	for i, cap := range result.Capabilities {
		fmt.Printf("    {\n")
		fmt.Printf("      \"key\": \"%s\",\n", cap.Key)
		fmt.Printf("      \"level\": %d", cap.Level)
		if cap.Message != "" {
			fmt.Printf(",\n      \"message\": \"%s\"", cap.Message)
		}
		if cap.GatedBy != "" {
			fmt.Printf(",\n      \"gatedBy\": \"%s\"", cap.GatedBy)
		}
		fmt.Printf("\n    }")
		if i < len(result.Capabilities)-1 {
			fmt.Printf(",")
		}
		fmt.Printf("\n")
	}

	fmt.Printf("  ]\n")
	fmt.Printf("}\n")
}

func supportLevelToString(level capabilities.SupportLevel) string {
	switch level {
	case capabilities.Unregistered:
		return "Unregistered"
	case capabilities.Unsupported:
		return "Unsupported"
	case capabilities.Experimental:
		return "Experimental"
	case capabilities.Deprecated:
		return "Deprecated"
	default:
		return fmt.Sprintf("Unknown(%d)", level)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s --hypervisor <hypervisor> --arch <arch> [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "List capabilities for a specific platform (hypervisor + architecture combination).\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  # List unsupported capabilities for KVM on amd64\n")
	fmt.Fprintf(os.Stderr, "  %s --hypervisor kvm --arch amd64\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  # List experimental capabilities with details\n")
	fmt.Fprintf(os.Stderr, "  %s --hypervisor kvm --arch amd64 --support-level experimental --output detailed\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  # List all capabilities in JSON format\n")
	fmt.Fprintf(os.Stderr, "  %s --hypervisor kvm --arch amd64 --list-all --output json\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\n")
	pflag.PrintDefaults()
}
