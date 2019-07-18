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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/client-go/api/v1"
	kvutil "kubevirt.io/kubevirt/pkg/util"
)

const (
	// Name of env var containing the operator's image name
	OperatorImageEnvName        = "OPERATOR_IMAGE"
	VirtApiShasumEnvName        = "VIRT_API_SHASUM"
	VirtControllerShasumEnvName = "VIRT_CONTROLLER_SHASUM"
	VirtHandlerShasumEnvName    = "VIRT_HANDLER_SHASUM"
	VirtLauncherShasumEnvName   = "VIRT_LAUNCHER_SHASUM"
	KubeVirtVersionEnvName      = "KUBEVIRT_VERSION"
	// Deprecated, use TargetDeploymentConfig instead
	TargetInstallNamespace = "TARGET_INSTALL_NAMESPACE"
	// Deprecated, use TargetDeploymentConfig instead
	TargetImagePullPolicy = "TARGET_IMAGE_PULL_POLICY"
	// JSON containing all relevant deployment properties, replaces TargetInstallNamespace and TargetImagePullPolicy
	TargetDeploymentConfig = "TARGET_DEPLOYMENT_CONFIG"

	// these names need to match field names from KubeVirt Spec if they are set from there
	AdditionalPropertiesNamePullPolicy = "ImagePullPolicy"

	// the regex used to parse the operator image
	operatorImageRegex = "^(.*)/virt-operator([@:].*)?$"
)

type KubeVirtDeploymentConfig struct {
	ID        string `json:"id,omitempty" optional:"true"`
	Namespace string `json:"namespace,omitempty" optional:"true"`
	Registry  string `json:"registry,omitempty" optional:"true"`

	// the KubeVirt version
	// matches the image tag, if tags are used, either by the manifest, or by the KubeVirt CR
	// used on the KubeVirt CR status and on annotations, and for determing up-/downgrade path, even when using shasums for the images
	KubeVirtVersion string `json:"kubeVirtVersion,omitempty" optional:"true"`

	// the shasums of every image we use
	VirtOperatorSha   string `json:"virtOperatorSha,omitempty" optional:"true"`
	VirtApiSha        string `json:"virtApiSha,omitempty" optional:"true"`
	VirtControllerSha string `json:"virtControllerSha,omitempty" optional:"true"`
	VirtHandlerSha    string `json:"virtHandlerSha,omitempty" optional:"true"`
	VirtLauncherSha   string `json:"virtLauncherSha,omitempty" optional:"true"`

	// everything else, which can e.g. come from KubeVirt CR spec
	AdditionalProperties map[string]string `json:"additionalProperties,omitempty" optional:"true"`
}

func GetConfigFromEnv() (*KubeVirtDeploymentConfig, error) {

	// first check if we have the new deployment config json
	c := os.Getenv(TargetDeploymentConfig)
	if c != "" {
		config := &KubeVirtDeploymentConfig{}
		if err := json.Unmarshal([]byte(c), config); err != nil {
			return nil, err
		}
		return config, nil
	}

	// for backwards compatibility: check for namespace and pullpolicy from deprecated env vars
	ns := os.Getenv(TargetInstallNamespace)
	if ns == "" {
		var err error
		ns, err = kvutil.GetNamespace()
		if err != nil {
			return nil, err
		}
	}

	pullPolicy := os.Getenv(TargetImagePullPolicy)
	additionalProperties := make(map[string]string)
	additionalProperties[AdditionalPropertiesNamePullPolicy] = pullPolicy
	return getConfig("", "", ns, additionalProperties), nil

}

func GetTargetConfigFromKV(kv *v1.KubeVirt) *KubeVirtDeploymentConfig {
	// don't use status.target* here, as that is always set, but we need to know if it was set by the spec and with that
	// overriding shasums from env vars
	return getConfig(kv.Spec.ImageRegistry, kv.Spec.ImageTag, kv.Namespace, getKVMapFromSpec(kv.Spec))
}

func GetObservedConfigFromKV(kv *v1.KubeVirt) *KubeVirtDeploymentConfig {
	return getConfig(kv.Status.ObservedKubeVirtRegistry, kv.Status.ObservedKubeVirtVersion, kv.Namespace, getKVMapFromSpec(kv.Spec))
}

func getKVMapFromSpec(spec v1.KubeVirtSpec) map[string]string {
	kvMap := make(map[string]string)
	v := reflect.ValueOf(spec)
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		if name == "ImageTag" || name == "ImageRegistry" {
			// these are handled in the root deployment config already
			continue
		}
		value := v.Field(i).String()
		kvMap[name] = value
	}
	return kvMap
}

func getConfig(registry, tag, namespace string, additionalProperties map[string]string) *KubeVirtDeploymentConfig {

	// get registry and tag/shasum from operator image
	imageString := os.Getenv(OperatorImageEnvName)
	imageRegEx := regexp.MustCompile(operatorImageRegex)
	matches := imageRegEx.FindAllStringSubmatch(imageString, 1)

	tagFromOperator := ""
	operatorSha := ""
	skipShasums := false

	if len(matches) == 1 {
		// only use registry from operator image if it was not given yet
		if registry == "" {
			registry = matches[0][1]
		}

		version := matches[0][2]
		if version == "" {
			tagFromOperator = "latest"
		} else if strings.HasPrefix(version, ":") {
			tagFromOperator = strings.TrimPrefix(version, ":")
		} else {
			// we have a shasum... chances are high that we get the shasums for the other images as well from env vars,
			// but as a fallback use latest tag
			tagFromOperator = "latest"
			operatorSha = strings.TrimPrefix(version, "@")
		}

		// only use tag from operator image if it was not given yet
		// and if it was given, don't look for shasums
		if tag == "" {
			tag = tagFromOperator
		} else {
			skipShasums = true
		}
	}

	config := newDeploymentConfigWithTag(registry, tag, namespace, additionalProperties)
	if skipShasums {
		return config
	}

	// get shasums
	apiSha := os.Getenv(VirtApiShasumEnvName)
	controllerSha := os.Getenv(VirtControllerShasumEnvName)
	handlerSha := os.Getenv(VirtHandlerShasumEnvName)
	launcherSha := os.Getenv(VirtLauncherShasumEnvName)
	kubeVirtVersion := os.Getenv(KubeVirtVersionEnvName)
	if operatorSha != "" && apiSha != "" && controllerSha != "" && handlerSha != "" && launcherSha != "" && kubeVirtVersion != "" {
		config = newDeploymentConfigWithShasums(registry, kubeVirtVersion, operatorSha, apiSha, controllerSha, handlerSha, launcherSha, namespace, additionalProperties)
	}

	return config
}

func VerifyEnv() error {
	// ensure the operator image is valid
	imageString := os.Getenv(OperatorImageEnvName)
	if imageString == "" {
		return fmt.Errorf("empty env var %s for operator image", OperatorImageEnvName)
	}
	imageRegEx := regexp.MustCompile(operatorImageRegex)
	matches := imageRegEx.FindAllStringSubmatch(imageString, 1)
	if len(matches) != 1 || len(matches[0]) != 3 {
		return fmt.Errorf("can not parse operator image env var %s", imageString)
	}

	// ensure that all or no shasums are given
	missingShas := make([]string, 0)
	count := 0
	for _, name := range []string{VirtApiShasumEnvName, VirtControllerShasumEnvName, VirtHandlerShasumEnvName, VirtLauncherShasumEnvName, KubeVirtVersionEnvName} {
		count++
		sha := os.Getenv(name)
		if sha == "" {
			missingShas = append(missingShas, name)
		}
	}
	if len(missingShas) > 0 && len(missingShas) < count {
		return fmt.Errorf("incomplete configuration, missing env vars %v", missingShas)
	}

	return nil
}

func newDeploymentConfigWithTag(registry, tag, namespace string, kvSpec map[string]string) *KubeVirtDeploymentConfig {
	c := &KubeVirtDeploymentConfig{
		Registry:             registry,
		KubeVirtVersion:      tag,
		Namespace:            namespace,
		AdditionalProperties: kvSpec,
	}
	c.generateInstallStrategyID()
	return c
}

func newDeploymentConfigWithShasums(registry, kubeVirtVersion, operatorSha, apiSha, controllerSha, handlerSha, launcherSha, namespace string, additionalProperties map[string]string) *KubeVirtDeploymentConfig {
	c := &KubeVirtDeploymentConfig{
		Registry:             registry,
		KubeVirtVersion:      kubeVirtVersion,
		VirtOperatorSha:      operatorSha,
		VirtApiSha:           apiSha,
		VirtControllerSha:    controllerSha,
		VirtHandlerSha:       handlerSha,
		VirtLauncherSha:      launcherSha,
		Namespace:            namespace,
		AdditionalProperties: additionalProperties,
	}
	c.generateInstallStrategyID()
	return c
}

func (c *KubeVirtDeploymentConfig) GetOperatorVersion() string {
	if c.UseShasums() {
		return c.VirtOperatorSha
	}
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetApiVersion() string {
	if c.UseShasums() {
		return c.VirtApiSha
	}
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetControllerVersion() string {
	if c.UseShasums() {
		return c.VirtControllerSha
	}
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetHandlerVersion() string {
	if c.UseShasums() {
		return c.VirtHandlerSha
	}
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetLauncherVersion() string {
	if c.UseShasums() {
		return c.VirtLauncherSha
	}
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetKubeVirtVersion() string {
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetImageRegistry() string {
	return c.Registry
}

func (c *KubeVirtDeploymentConfig) UseShasums() bool {
	return c.VirtOperatorSha != "" && c.VirtApiSha != "" && c.VirtControllerSha != "" && c.VirtHandlerSha != "" && c.VirtLauncherSha != ""
}

func (c *KubeVirtDeploymentConfig) SetTargetDeploymentConfig(kv *v1.KubeVirt) error {
	kv.Status.TargetKubeVirtVersion = c.GetKubeVirtVersion()
	kv.Status.TargetKubeVirtRegistry = c.GetImageRegistry()
	kv.Status.TargetDeploymentID = c.GetDeploymentID()
	json, err := c.GetJson()
	kv.Status.TargetDeploymentConfig = json
	return err
}

func (c *KubeVirtDeploymentConfig) SetObservedDeploymentConfig(kv *v1.KubeVirt) error {
	kv.Status.ObservedKubeVirtVersion = c.GetKubeVirtVersion()
	kv.Status.ObservedKubeVirtRegistry = c.GetImageRegistry()
	kv.Status.ObservedDeploymentID = c.GetDeploymentID()
	json, err := c.GetJson()
	kv.Status.ObservedDeploymentConfig = json
	return err
}

func (c *KubeVirtDeploymentConfig) GetImagePullPolicy() k8sv1.PullPolicy {
	p := c.AdditionalProperties[AdditionalPropertiesNamePullPolicy]
	if p != "" {
		return k8sv1.PullPolicy(p)
	}
	return k8sv1.PullIfNotPresent
}

func (c *KubeVirtDeploymentConfig) GetNamespace() string {
	return c.Namespace
}

func (c *KubeVirtDeploymentConfig) GetVerbosity() string {
	// not configurable yet
	return "2"
}

func (c *KubeVirtDeploymentConfig) generateInstallStrategyID() {
	// We need an id, which identifies a KubeVirt deployment based on version, shasums, registry, namespace, and other
	// changeable properties from the KubeVirt CR. This will be used for identifying the correct install strategy job
	// and configmap
	// Calculate a sha over all those properties

	hasher := sha1.New()

	version := fmt.Sprintf("%+v", c)
	hasher.Write([]byte(version))

	c.ID = hex.EncodeToString(hasher.Sum(nil))
}

func (c *KubeVirtDeploymentConfig) GetDeploymentID() string {
	return c.ID
}

func (c *KubeVirtDeploymentConfig) GetJson() (string, error) {
	json, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(json), nil
}
