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

// Package config provides centralized configuration management for virt-launcher.
// All environment variables should be read at the application's top level and
// stored in the Config struct, which is then passed to components that need
// access to these values.
//
// This package serves as a single source of truth for all environment variables
// that virt-launcher supports. By reading all environment variables at startup
// and storing them in the Config struct, we gain:
//
//   - Clear visibility of all supported configuration options
//   - Easier testing through dependency injection
//   - Consistent behavior across the application
//   - Single point of documentation for environment variables
//
// Usage:
//
//	cfg := config.NewConfig()
//	// Pass cfg to components that need it
package config

import (
	"os"
	"strconv"
	"sync"

	"kubevirt.io/client-go/log"
)

// Environment variable names used by virt-launcher.
// These constants define all supported environment variables for easy reference.
const (
	// EnvVarVirtLauncherLogVerbosity controls the log verbosity level for virt-launcher.
	// When set to a numeric value, it adjusts the logging verbosity.
	// Values above EXT_LOG_VERBOSITY_THRESHOLD (5) enable extended debug logging.
	EnvVarVirtLauncherLogVerbosity = "VIRT_LAUNCHER_LOG_VERBOSITY"

	// EnvVarLibvirtDebugLogs enables libvirt debug logging when set (any value).
	// This is typically set to "1" but the presence of the variable is what matters.
	EnvVarLibvirtDebugLogs = "LIBVIRT_DEBUG_LOGS"

	// EnvVarVirtiofsdDebugLogs enables virtiofsd debug logging when set to "1".
	// Unlike LIBVIRT_DEBUG_LOGS, this requires the specific value "1".
	EnvVarVirtiofsdDebugLogs = "VIRTIOFSD_DEBUG_LOGS"

	// EnvVarSharedFilesystemPaths contains colon-separated paths for shared filesystems.
	// These paths are used to configure QEMU's shared filesystem feature.
	EnvVarSharedFilesystemPaths = "SHARED_FILESYSTEM_PATHS"

	// EnvVarStandaloneVMI contains the VMI object in JSON/YAML format for standalone mode.
	// When set, virt-launcher will sync the VMI directly without waiting for virt-handler.
	EnvVarStandaloneVMI = "STANDALONE_VMI"

	// EnvVarTargetPodExitSignal is set internally to signal that the target pod
	// should perform cleanup during migration. This is not meant to be set externally.
	EnvVarTargetPodExitSignal = "VIRT_LAUNCHER_TARGET_POD_EXIT_SIGNAL"

	// EnvVarPodName contains the name of the current pod.
	// This is typically set by Kubernetes downward API.
	EnvVarPodName = "POD_NAME"
)

// Config holds all configuration values read from environment variables.
// This struct should be instantiated once at application startup and passed
// to components that need access to these configuration values.
type Config struct {
	// LogVerbosity is the parsed log verbosity level.
	// -1 indicates the environment variable was not set or invalid.
	LogVerbosity int

	// LogVerbosityRaw is the raw string value from the environment variable.
	// Empty string if not set.
	LogVerbosityRaw string

	// LibvirtDebugLogsEnabled indicates whether LIBVIRT_DEBUG_LOGS is set.
	LibvirtDebugLogsEnabled bool

	// VirtiofsdDebugLogsEnabled indicates whether VIRTIOFSD_DEBUG_LOGS is set to "1".
	VirtiofsdDebugLogsEnabled bool

	// SharedFilesystemPaths contains the raw value of SHARED_FILESYSTEM_PATHS.
	// Empty string if not set.
	SharedFilesystemPaths string

	// StandaloneVMI contains the VMI object string for standalone mode.
	// Empty string if not set.
	StandaloneVMI string

	// PodName contains the name of the current pod.
	// Empty string if not set.
	PodName string
}

var (
	// globalConfig holds the singleton config instance
	globalConfig *Config
	configOnce   sync.Once
)

// NewConfig reads all supported environment variables and returns a populated Config struct.
// This function should be called once at the application's top level (main function).
func NewConfig() *Config {
	cfg := &Config{
		LogVerbosity: -1,
	}

	// Read VIRT_LAUNCHER_LOG_VERBOSITY
	if verbosityStr, ok := os.LookupEnv(EnvVarVirtLauncherLogVerbosity); ok {
		cfg.LogVerbosityRaw = verbosityStr
		if verbosity, err := strconv.Atoi(verbosityStr); err == nil {
			cfg.LogVerbosity = verbosity
		} else {
			log.Log.Warningf("failed to parse %s value %q - must be an integer",
				EnvVarVirtLauncherLogVerbosity, verbosityStr)
		}
	}

	// Read LIBVIRT_DEBUG_LOGS
	_, cfg.LibvirtDebugLogsEnabled = os.LookupEnv(EnvVarLibvirtDebugLogs)

	// Read VIRTIOFSD_DEBUG_LOGS
	if debugLogsStr, ok := os.LookupEnv(EnvVarVirtiofsdDebugLogs); ok && debugLogsStr == "1" {
		cfg.VirtiofsdDebugLogsEnabled = true
	}

	// Read SHARED_FILESYSTEM_PATHS
	cfg.SharedFilesystemPaths, _ = os.LookupEnv(EnvVarSharedFilesystemPaths)

	// Read STANDALONE_VMI
	cfg.StandaloneVMI, _ = os.LookupEnv(EnvVarStandaloneVMI)

	// Read POD_NAME
	cfg.PodName, _ = os.LookupEnv(EnvVarPodName)

	return cfg
}

// GetGlobalConfig returns the singleton config instance, initializing it if necessary.
// This provides backward compatibility for code that cannot easily receive the config
// as a parameter. New code should prefer receiving Config as a dependency.
func GetGlobalConfig() *Config {
	configOnce.Do(func() {
		globalConfig = NewConfig()
	})
	return globalConfig
}

// SetGlobalConfig sets the global config instance. This is primarily useful for testing.
// In production code, prefer using NewConfig() at startup and passing the config explicitly.
func SetGlobalConfig(cfg *Config) {
	globalConfig = cfg
	// Reset the once so GetGlobalConfig will use the new value
	configOnce = sync.Once{}
	configOnce.Do(func() {}) // Mark as done
}

// ResetGlobalConfig resets the global config to nil, primarily for testing purposes.
func ResetGlobalConfig() {
	globalConfig = nil
	configOnce = sync.Once{}
}

// IsLogVerbositySet returns true if the log verbosity environment variable was set and valid.
func (c *Config) IsLogVerbositySet() bool {
	return c.LogVerbosity >= 0
}

// HasSharedFilesystemPaths returns true if shared filesystem paths are configured.
func (c *Config) HasSharedFilesystemPaths() bool {
	return c.SharedFilesystemPaths != ""
}

// IsStandaloneMode returns true if virt-launcher is running in standalone mode.
func (c *Config) IsStandaloneMode() bool {
	return c.StandaloneVMI != ""
}
