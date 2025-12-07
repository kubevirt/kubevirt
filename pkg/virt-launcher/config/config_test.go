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

package config

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virt-Launcher Config Suite")
}

var _ = Describe("Config", func() {
	var originalEnvVars map[string]string

	BeforeEach(func() {
		// Reset global config before each test
		ResetGlobalConfig()

		// Save original environment variables
		originalEnvVars = make(map[string]string)
		envVars := []string{
			EnvVarVirtLauncherLogVerbosity,
			EnvVarLibvirtDebugLogs,
			EnvVarVirtiofsdDebugLogs,
			EnvVarSharedFilesystemPaths,
			EnvVarStandaloneVMI,
			EnvVarPodName,
		}
		for _, envVar := range envVars {
			if val, ok := os.LookupEnv(envVar); ok {
				originalEnvVars[envVar] = val
			}
			os.Unsetenv(envVar)
		}
	})

	AfterEach(func() {
		// Restore original environment variables
		envVars := []string{
			EnvVarVirtLauncherLogVerbosity,
			EnvVarLibvirtDebugLogs,
			EnvVarVirtiofsdDebugLogs,
			EnvVarSharedFilesystemPaths,
			EnvVarStandaloneVMI,
			EnvVarPodName,
		}
		for _, envVar := range envVars {
			os.Unsetenv(envVar)
			if val, ok := originalEnvVars[envVar]; ok {
				os.Setenv(envVar, val)
			}
		}

		// Reset global config after each test
		ResetGlobalConfig()
	})

	Describe("NewConfig", func() {
		It("should return default values when no env vars are set", func() {
			cfg := NewConfig()

			Expect(cfg.LogVerbosity).To(Equal(-1))
			Expect(cfg.LogVerbosityRaw).To(BeEmpty())
			Expect(cfg.LibvirtDebugLogsEnabled).To(BeFalse())
			Expect(cfg.VirtiofsdDebugLogsEnabled).To(BeFalse())
			Expect(cfg.SharedFilesystemPaths).To(BeEmpty())
			Expect(cfg.StandaloneVMI).To(BeEmpty())
			Expect(cfg.PodName).To(BeEmpty())
		})

		It("should parse valid log verbosity", func() {
			os.Setenv(EnvVarVirtLauncherLogVerbosity, "5")

			cfg := NewConfig()

			Expect(cfg.LogVerbosity).To(Equal(5))
			Expect(cfg.LogVerbosityRaw).To(Equal("5"))
			Expect(cfg.IsLogVerbositySet()).To(BeTrue())
		})

		It("should handle invalid log verbosity", func() {
			os.Setenv(EnvVarVirtLauncherLogVerbosity, "invalid")

			cfg := NewConfig()

			Expect(cfg.LogVerbosity).To(Equal(-1))
			Expect(cfg.LogVerbosityRaw).To(Equal("invalid"))
			Expect(cfg.IsLogVerbositySet()).To(BeFalse())
		})

		It("should detect libvirt debug logs when env var is set", func() {
			os.Setenv(EnvVarLibvirtDebugLogs, "1")

			cfg := NewConfig()

			Expect(cfg.LibvirtDebugLogsEnabled).To(BeTrue())
		})

		It("should detect libvirt debug logs even with empty value", func() {
			os.Setenv(EnvVarLibvirtDebugLogs, "")

			cfg := NewConfig()

			Expect(cfg.LibvirtDebugLogsEnabled).To(BeTrue())
		})

		It("should enable virtiofsd debug logs only when set to 1", func() {
			os.Setenv(EnvVarVirtiofsdDebugLogs, "1")

			cfg := NewConfig()

			Expect(cfg.VirtiofsdDebugLogsEnabled).To(BeTrue())
		})

		It("should not enable virtiofsd debug logs when set to other values", func() {
			os.Setenv(EnvVarVirtiofsdDebugLogs, "0")

			cfg := NewConfig()

			Expect(cfg.VirtiofsdDebugLogsEnabled).To(BeFalse())
		})

		It("should read shared filesystem paths", func() {
			os.Setenv(EnvVarSharedFilesystemPaths, "/path1:/path2")

			cfg := NewConfig()

			Expect(cfg.SharedFilesystemPaths).To(Equal("/path1:/path2"))
			Expect(cfg.HasSharedFilesystemPaths()).To(BeTrue())
		})

		It("should read standalone VMI", func() {
			vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance"}`
			os.Setenv(EnvVarStandaloneVMI, vmiJSON)

			cfg := NewConfig()

			Expect(cfg.StandaloneVMI).To(Equal(vmiJSON))
			Expect(cfg.IsStandaloneMode()).To(BeTrue())
		})

		It("should read pod name", func() {
			os.Setenv(EnvVarPodName, "test-pod-123")

			cfg := NewConfig()

			Expect(cfg.PodName).To(Equal("test-pod-123"))
		})
	})

	Describe("Global config", func() {
		It("should return the same instance on multiple calls", func() {
			os.Setenv(EnvVarPodName, "test-pod")

			cfg1 := GetGlobalConfig()
			cfg2 := GetGlobalConfig()

			Expect(cfg1).To(BeIdenticalTo(cfg2))
			Expect(cfg1.PodName).To(Equal("test-pod"))
		})

		It("should allow setting a custom global config", func() {
			customCfg := &Config{
				LogVerbosity: 10,
				PodName:      "custom-pod",
			}

			SetGlobalConfig(customCfg)

			Expect(GetGlobalConfig()).To(BeIdenticalTo(customCfg))
			Expect(GetGlobalConfig().LogVerbosity).To(Equal(10))
		})
	})

	Describe("Helper methods", func() {
		It("IsLogVerbositySet should return false when verbosity is -1", func() {
			cfg := &Config{LogVerbosity: -1}
			Expect(cfg.IsLogVerbositySet()).To(BeFalse())
		})

		It("IsLogVerbositySet should return true when verbosity is 0", func() {
			cfg := &Config{LogVerbosity: 0}
			Expect(cfg.IsLogVerbositySet()).To(BeTrue())
		})

		It("HasSharedFilesystemPaths should return false for empty paths", func() {
			cfg := &Config{SharedFilesystemPaths: ""}
			Expect(cfg.HasSharedFilesystemPaths()).To(BeFalse())
		})

		It("IsStandaloneMode should return false for empty VMI", func() {
			cfg := &Config{StandaloneVMI: ""}
			Expect(cfg.IsStandaloneMode()).To(BeFalse())
		})
	})
})
