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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package util

import (
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("Operator Config", func() {
	var envVarManager EnvVarManager

	BeforeEach(func() {
		envVarManager = &EnvVarManagerMock{}
		DefaultEnvVarManager = envVarManager
	})

	getConfig := func(registry, version string) *KubeVirtDeploymentConfig {
		return &KubeVirtDeploymentConfig{
			Registry:        registry,
			KubeVirtVersion: version,
		}
	}

	DescribeTable("Parse image", func(image string, config *KubeVirtDeploymentConfig) {
		envVarManager.Setenv(OldOperatorImageEnvName, image)

		err := VerifyEnv()
		Expect(err).ToNot(HaveOccurred())

		parsedConfig, err := GetConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedConfig.GetImageRegistry()).To(Equal(config.GetImageRegistry()), "registry should match")
		Expect(parsedConfig.GetKubeVirtVersion()).To(Equal(config.GetKubeVirtVersion()), "tag should match")
	},
		Entry("without registry", "kubevirt/virt-operator:v123", getConfig("kubevirt", "v123")),
		Entry("with registry", "reg/kubevirt/virt-operator:v123", getConfig("reg/kubevirt", "v123")),
		Entry("with registry with port", "reg:1234/kubevirt/virt-operator:latest", getConfig("reg:1234/kubevirt", "latest")),
		Entry("without tag", "kubevirt/virt-operator", getConfig("kubevirt", "latest")),
		Entry("with shasum", "kubevirt/virt-operator@sha256:abcdef", getConfig("kubevirt", "latest")),
	)

	getConfigWithShas := func(apiSha, controllerSha, handlerSha, launcherSha, version string) *KubeVirtDeploymentConfig {
		return &KubeVirtDeploymentConfig{
			KubeVirtVersion:   version,
			VirtApiSha:        apiSha,
			VirtControllerSha: controllerSha,
			VirtHandlerSha:    handlerSha,
			VirtLauncherSha:   launcherSha,
		}
	}

	getFullConfig := func(registry, operatorSha, apiSha, controllerSha, handlerSha, launcherSha, version string) *KubeVirtDeploymentConfig {
		return &KubeVirtDeploymentConfig{
			Registry:          registry,
			KubeVirtVersion:   version,
			VirtOperatorSha:   operatorSha,
			VirtApiSha:        apiSha,
			VirtControllerSha: controllerSha,
			VirtHandlerSha:    handlerSha,
			VirtLauncherSha:   launcherSha,
		}
	}

	DescribeTable("Read shasums", func(image string, envVersions *KubeVirtDeploymentConfig, expectedConfig *KubeVirtDeploymentConfig, useShasums bool) {
		envVarManager.Setenv(OldOperatorImageEnvName, image)

		envVarManager.Setenv(VirtApiShasumEnvName, envVersions.VirtApiSha)
		envVarManager.Setenv(VirtControllerShasumEnvName, envVersions.VirtControllerSha)
		envVarManager.Setenv(VirtHandlerShasumEnvName, envVersions.VirtHandlerSha)
		envVarManager.Setenv(VirtLauncherShasumEnvName, envVersions.VirtLauncherSha)
		envVarManager.Setenv(KubeVirtVersionEnvName, envVersions.KubeVirtVersion)

		err := VerifyEnv()
		Expect(err).ToNot(HaveOccurred())

		parsedConfig, err := GetConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedConfig.GetImageRegistry()).To(Equal(expectedConfig.GetImageRegistry()), "registry should match")
		Expect(parsedConfig.GetKubeVirtVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "tag / shasums should match")

		if useShasums {
			Expect(parsedConfig.GetApiVersion()).To(Equal(expectedConfig.GetApiVersion()), "api shasums should match")
			Expect(parsedConfig.GetControllerVersion()).To(Equal(expectedConfig.GetControllerVersion()), "controller shasums should match")
			Expect(parsedConfig.GetHandlerVersion()).To(Equal(expectedConfig.GetHandlerVersion()), "handler shasums should match")
			Expect(parsedConfig.GetLauncherVersion()).To(Equal(expectedConfig.GetLauncherVersion()), "launcher shasums should match")
		} else {
			Expect(parsedConfig.GetApiVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "api version should be tag")
			Expect(parsedConfig.GetControllerVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "controller version should be tag")
			Expect(parsedConfig.GetHandlerVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "handler version should be tag")
			Expect(parsedConfig.GetLauncherVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "launcher version should be tag")
		}
	},
		Entry("with no shasum given", "kubevirt/virt-operator:v123",
			&KubeVirtDeploymentConfig{},
			getConfig("kubevirt", "v123"),
			false),
		Entry("with all shasums given", "kubevirt/virt-operator@sha256:operator",
			getConfigWithShas("sha256:api", "sha256:controller", "sha256:handler", "sha256:launcher", "v234"),
			getFullConfig("kubevirt", "sha256:operator", "sha256:api", "sha256:controller", "sha256:handler", "sha256:launcher", "v234"),
			true),
	)

	It("should be able to extract shasum from image names", func() {
		envVarManager.Setenv(VirtOperatorImageEnvName,
			"acme.com/kubevirt/virt-operator@sha256:virt-operator-sha")
		envVarManager.Setenv(VirtApiImageEnvName,
			"acme.com/kubevirt/virt-api@sha256:virt-api-sha")
		envVarManager.Setenv(VirtControllerImageEnvName,
			"acme.com/kubevirt/virt-controller@sha256:virt-controller-sha")
		envVarManager.Setenv(VirtHandlerImageEnvName,
			"acme.com/kubevirt/virt-handler@sha256:virt-handler-sha")
		envVarManager.Setenv(VirtLauncherImageEnvName,
			"acme.com/kubevirt/virt-launcher@sha256:virt-launcher-sha")
		envVarManager.Setenv(VirtExportProxyImageEnvName,
			"acme.com/kubevirt/virt-exportproxy@sha256:virt-exportproxy-sha")
		envVarManager.Setenv(VirtExportServerImageEnvName,
			"acme.com/kubevirt/virt-exportserver@sha256:virt-exportserver-sha")

		err := VerifyEnv()
		Expect(err).ToNot(HaveOccurred())

		parsedConfig, err := GetConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())
		Expect("virt-operator-sha").To(Equal(parsedConfig.GetOperatorVersion()))
		Expect("virt-api-sha").To(Equal(parsedConfig.GetApiVersion()))
		Expect("virt-controller-sha").To(Equal(parsedConfig.GetControllerVersion()))
		Expect("virt-handler-sha").To(Equal(parsedConfig.GetHandlerVersion()))
		Expect("virt-launcher-sha").To(Equal(parsedConfig.GetLauncherVersion()))
		Expect("virt-exportproxy-sha").To(Equal(parsedConfig.GetExportProxyVersion()))
		Expect("virt-exportserver-sha").To(Equal(parsedConfig.GetExportServerVersion()))
	})

	Describe("GetPassthroughEnv()", func() {
		It("should eturn environment variables matching the passthrough prefix (and only those vars)", func() {
			realKey := rand.String(10)
			passthroughKey := fmt.Sprintf("%s%s", PassthroughEnvPrefix, realKey)
			otherKey := rand.String(10)
			val := rand.String(10)

			err := envVarManager.Setenv(passthroughKey, val)
			Expect(err).ToNot(HaveOccurred())

			err = envVarManager.Setenv(otherKey, val)
			Expect(err).ToNot(HaveOccurred())

			envMap := GetPassthroughEnv()

			err = os.Unsetenv(passthroughKey)
			Expect(err).ToNot(HaveOccurred())

			err = os.Unsetenv(otherKey)
			Expect(err).ToNot(HaveOccurred())

			Expect(envMap).To(Equal(map[string]string{realKey: val}))
		})
	})

	Describe("NewEnvVarMap()", func() {
		It("Should convert a map to a list of EnvVar objects", func() {
			key1 := rand.String(10)
			val1 := rand.String(10)
			key2 := rand.String(10)
			val2 := rand.String(10)

			envMap := map[string]string{key1: val1, key2: val2}

			envObjects := NewEnvVarMap(envMap)
			expected := []k8sv1.EnvVar{
				{Name: key1, Value: val1},
				{Name: key2, Value: val2},
			}

			Expect(*envObjects).To(ConsistOf(expected))
		})
	})

	Describe("Config json from env var", func() {
		It("should be parsed", func() {
			json := `{"id":"9ca7273e4d5f1bee842f64a8baabc15cbbf1ce59","namespace":"kubevirt","registry":"registry:5000/kubevirt","imagePrefix":"somePrefix","kubeVirtVersion":"devel","additionalProperties":{"ImagePullPolicy":"IfNotPresent", "MonitorNamespace":"non-default-monitor-namespace", "MonitorAccount":"non-default-prometheus-k8s"}}`
			envVarManager.Setenv(TargetDeploymentConfig, json)
			parsedConfig, err := GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedConfig.GetDeploymentID()).To(Equal("9ca7273e4d5f1bee842f64a8baabc15cbbf1ce59"))
			Expect(parsedConfig.GetNamespace()).To(Equal("kubevirt"))
			Expect(parsedConfig.GetImageRegistry()).To(Equal("registry:5000/kubevirt"))
			Expect(parsedConfig.GetImagePrefix()).To(Equal("somePrefix"))
			Expect(parsedConfig.GetKubeVirtVersion()).To(Equal("devel"))
			Expect(parsedConfig.GetImagePullPolicy()).To(Equal(k8sv1.PullIfNotPresent))
			Expect(parsedConfig.GetPotentialMonitorNamespaces()).To(ConsistOf("non-default-monitor-namespace"))
			Expect(parsedConfig.GetMonitorServiceAccountName()).To(Equal("non-default-prometheus-k8s"))
		})
	})

	Describe("Config json with default value", func() {
		It("should be parsed", func() {
			json := `{"id":"9ca7273e4d5f1bee842f64a8baabc15cbbf1ce59","additionalProperties":{"ImagePullPolicy":"IfNotPresent"}}`
			envVarManager.Setenv(TargetDeploymentConfig, json)
			parsedConfig, err := GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedConfig.GetPotentialMonitorNamespaces()).To(ConsistOf("openshift-monitoring", "monitoring"))
			Expect(parsedConfig.GetMonitorServiceAccountName()).To(Equal("prometheus-k8s"))
		})
	})

	Describe("parsing ObservedDeploymentConfig", func() {
		It("should retrieve imagePrefix if present", func() {
			prefix := "test-prefix-"
			deploymentConfig := KubeVirtDeploymentConfig{}
			deploymentConfig.ImagePrefix = prefix

			blob, err := deploymentConfig.GetJson()
			Expect(err).ToNot(HaveOccurred())

			result, found, err := getImagePrefixFromDeploymentConfig(blob)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(result).To(Equal(prefix))
		})

		It("should not error if imagePrefix is not present", func() {
			deploymentConfig := KubeVirtDeploymentConfig{}

			blob, err := deploymentConfig.GetJson()
			Expect(err).ToNot(HaveOccurred())

			result, found, err := getImagePrefixFromDeploymentConfig(blob)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
			Expect(result).To(Equal(""))
		})
	})

	Describe("creating config ID", func() {
		var idMissing, idEmpty, idFilled string

		BeforeEach(func() {
			cfgMissing := &KubeVirtDeploymentConfig{}
			cfgMissing.AdditionalProperties = make(map[string]string)
			cfgMissing.generateInstallStrategyID()
			idMissing = cfgMissing.ID

			cfgEmpty := &KubeVirtDeploymentConfig{}
			cfgEmpty.AdditionalProperties = make(map[string]string)
			cfgEmpty.AdditionalProperties[ImagePrefixKey] = ""
			cfgEmpty.generateInstallStrategyID()
			idEmpty = cfgEmpty.ID

			cfgFilled := &KubeVirtDeploymentConfig{}
			cfgFilled.AdditionalProperties = make(map[string]string)
			cfgFilled.AdditionalProperties[ImagePrefixKey] = "something"
			cfgFilled.generateInstallStrategyID()
			idFilled = cfgFilled.ID
		})

		It("should result in same ID with missing or empty image prefix", func() {
			Expect(idMissing).ToNot(BeEmpty())
			Expect(idMissing).To(Equal(idEmpty))
		})

		It("should result in different ID with filled image prefix", func() {
			Expect(idFilled).ToNot(BeEmpty())
			Expect(idFilled).ToNot(Equal(idEmpty))
		})
	})

	Context("Product Names and Versions", func() {
		DescribeTable("label validation", func(testVector string, expectedResult bool) {
			Expect(IsValidLabel(testVector)).To(Equal(expectedResult))
		},
			Entry("should allow 1 character strings", "a", true),
			Entry("should allow 2 character strings", "aa", true),
			Entry("should allow 3 character strings", "aaa", true),
			Entry("should allow 63 character strings", strings.Repeat("a", 63), true),
			Entry("should reject 64 character strings", strings.Repeat("a", 64), false),
			Entry("should reject strings that begin with .", ".a", false),
			Entry("should reject strings that end with .", "a.", false),
			Entry("should reject strings that contain junk characters", `a\a`, false),
			Entry("should allow strings that contain dots", "a.a", true),
			Entry("should allow strings that contain dashes", "a-a", true),
			Entry("should allow strings that contain underscores", "a_a", true),
			Entry("should allow empty strings", "", true),
		)
	})

	Context("custom component images", func() {
		var definedEnvVars []string

		setCustomImageForComponent := func(component string) string {
			customImage := "a/kubevirt:" + component

			// defining a different SHA so we make sure the custom image has precedence
			customSha := "sha256:" + component + "fake-suffix"

			envVarNameBase := strings.ToUpper(component)
			envVarNameBase = strings.ReplaceAll(envVarNameBase, "-", "_") + "_"
			imageEnvVarName := envVarNameBase + "IMAGE"
			shaEnvVarName := envVarNameBase + "SHASUM"

			ExpectWithOffset(1, envVarManager.Setenv(imageEnvVarName, customImage)).To(Succeed())
			definedEnvVars = append(definedEnvVars, imageEnvVarName)

			ExpectWithOffset(1, envVarManager.Setenv(shaEnvVarName, customSha)).To(Succeed())
			definedEnvVars = append(definedEnvVars, shaEnvVarName)

			return customImage
		}

		BeforeEach(func() {
			definedEnvVars = nil

			config := getConfig("kubevirt", "v123")

			ExpectWithOffset(1, envVarManager.Setenv(KubeVirtVersionEnvName, config.KubeVirtVersion)).To(Succeed())
		})

		AfterEach(func() {
			for _, envVar := range definedEnvVars {
				_ = os.Unsetenv(envVar)
			}

			_ = os.Unsetenv(KubeVirtVersionEnvName)
		})

		It("should pull the right image", func() {
			operatorImage := setCustomImageForComponent("virt-operator")
			apiImage := setCustomImageForComponent("virt-api")
			controllerImage := setCustomImageForComponent("virt-controller")
			handlerImage := setCustomImageForComponent("virt-handler")
			launcherImage := setCustomImageForComponent("virt-launcher")
			exportProxyImage := setCustomImageForComponent("virt-exportproxy")
			exportServerImage := setCustomImageForComponent("virt-exportserver")
			gsImage := setCustomImageForComponent("gs")

			err := VerifyEnv()
			Expect(err).ToNot(HaveOccurred())

			parsedConfig, err := GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())

			const errMsg = "image is not set as expected"
			Expect(parsedConfig.VirtOperatorImage).To(Equal(operatorImage), errMsg)
			Expect(parsedConfig.VirtApiImage).To(Equal(apiImage), errMsg)
			Expect(parsedConfig.VirtControllerImage).To(Equal(controllerImage), errMsg)
			Expect(parsedConfig.VirtHandlerImage).To(Equal(handlerImage), errMsg)
			Expect(parsedConfig.VirtLauncherImage).To(Equal(launcherImage), errMsg)
			Expect(parsedConfig.VirtExportProxyImage).To(Equal(exportProxyImage), errMsg)
			Expect(parsedConfig.VirtExportServerImage).To(Equal(exportServerImage), errMsg)
			Expect(parsedConfig.GsImage).To(Equal(gsImage), errMsg)
		})

		DescribeTable("when virt-operator image is", func(envVarName string, isValid bool) {
			const image = "some.registry.io/virt-operator@sha256:abcdefg"

			if envVarName != "" {
				err := envVarManager.Setenv(envVarName, image)
				Expect(err).ToNot(HaveOccurred())
			}

			err := VerifyEnv()
			if isValid {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
			Entry(fmt.Sprintf("provided via new %s env variable - expected to pass", VirtOperatorImageEnvName), VirtOperatorImageEnvName, true),
			Entry(fmt.Sprintf("provided via old %s env variable - expected to pass", OldOperatorImageEnvName), OldOperatorImageEnvName, true),
			Entry("not provided at all - expected to fail", "", false),
		)
	})

	Context("kubevirt version", func() {
		type testInput struct {
			imageName         string
			kubevirtVerEnvVar string
			version           string
		}

		BeforeEach(func() {
			ExpectWithOffset(1, envVarManager.Unsetenv(KubeVirtVersionEnvName)).To(Succeed())
			ExpectWithOffset(1, envVarManager.Unsetenv(VirtOperatorImageEnvName)).To(Succeed())
		})

		DescribeTable("is read from", func(input *testInput) {
			Expect(envVarManager.Setenv(VirtOperatorImageEnvName, input.imageName)).To(Succeed())

			if input.kubevirtVerEnvVar != "" {
				Expect(envVarManager.Setenv(KubeVirtVersionEnvName, input.kubevirtVerEnvVar)).To(Succeed())
			}

			err := VerifyEnv()
			Expect(err).ToNot(HaveOccurred())

			parsedConfig, err := GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())

			kubevirtVersion := parsedConfig.GetKubeVirtVersion()
			Expect(kubevirtVersion).To(Equal(input.version))
		},
			Entry("virt-operator image tag when both KUBEVIRT_VERSION is set and virt-operator provided with tag",
				&testInput{
					kubevirtVerEnvVar: "v3.0.0-env.var",
					imageName:         "acme.com/kubevirt/my-virt-operator:v3.0.0",
					version:           "v3.0.0",
				}),

			Entry("KUBEVIRT_VERSION variable when virt-operator provided with digest",
				&testInput{
					kubevirtVerEnvVar: "v3.0.0",
					imageName:         "acme.com/kubevirt/my-virt-operator@sha256:trivebuk",
					version:           "v3.0.0",
				}),
			Entry("operator tag when no KUBEVIRT_VERSION provided and operator image is with a tag",
				&testInput{
					imageName: "acme.com/kubevirt/my-virt-operator:v3.0.0",
					version:   "v3.0.0",
				}),
			Entry("hardcoded \"latest\" string when no KUBEVIRT_VERSION provided and operator image is with a digest",
				&testInput{
					version:   "latest",
					imageName: "acme.com/kubevirt/my-virt-operator@sha256:trivebuk",
				}),
			Entry("KUBEVIRT_VERSION variable when virt-operator image name is corrupted",
				&testInput{
					kubevirtVerEnvVar: "v3.0.0",
					imageName:         "blablabla",
					version:           "v3.0.0",
				}),
			Entry("hardcoded \"latest\" string when no KUBEVIRT_VERSION provided and operator image is corrupted",
				&testInput{
					imageName: "blablabla",
					version:   "latest",
				}))
	})
})
