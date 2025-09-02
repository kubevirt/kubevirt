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

package util

import (
	// #nosec sha1 used to calculate hash to identify the deployment and not as cryptographic info
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

const (
	// Name of env var containing the operator's image name
	// Deprecated. Use VirtOperatorImageEnvName instead
	OldOperatorImageEnvName                   = "OPERATOR_IMAGE"
	VirtOperatorImageEnvName                  = "VIRT_OPERATOR_IMAGE"
	VirtApiImageEnvName                       = "VIRT_API_IMAGE"
	VirtControllerImageEnvName                = "VIRT_CONTROLLER_IMAGE"
	VirtHandlerImageEnvName                   = "VIRT_HANDLER_IMAGE"
	VirtLauncherImageEnvName                  = "VIRT_LAUNCHER_IMAGE"
	VirtExportProxyImageEnvName               = "VIRT_EXPORTPROXY_IMAGE"
	VirtExportServerImageEnvName              = "VIRT_EXPORTSERVER_IMAGE"
	VirtSynchronizationControllerImageEnvName = "VIRT_SYNCHRONIZATIONCONTROLLER_IMAGE"
	GsImageEnvName                            = "GS_IMAGE"
	PrHelperImageEnvName                      = "PR_HELPER_IMAGE"
	SidecarShimImageEnvName                   = "SIDECAR_SHIM_IMAGE"
	RunbookURLTemplate                        = "RUNBOOK_URL_TEMPLATE"

	KubeVirtVersionEnvName = "KUBEVIRT_VERSION"
	// Deprecated, use TargetDeploymentConfig instead
	TargetInstallNamespace = "TARGET_INSTALL_NAMESPACE"
	// Deprecated, use TargetDeploymentConfig instead
	TargetImagePullPolicy = "TARGET_IMAGE_PULL_POLICY"
	// JSON containing all relevant deployment properties, replaces TargetInstallNamespace and TargetImagePullPolicy
	TargetDeploymentConfig = "TARGET_DEPLOYMENT_CONFIG"

	// these names need to match field names from KubeVirt Spec if they are set from there
	AdditionalPropertiesNamePullPolicy = "ImagePullPolicy"
	AdditionalPropertiesPullSecrets    = "ImagePullSecrets"

	// lookup key in AdditionalProperties
	AdditionalPropertiesMonitorNamespace = "MonitorNamespace"

	// lookup key in AdditionalProperties
	AdditionalPropertiesServiceMonitorNamespace = "ServiceMonitorNamespace"

	// lookup key in AdditionalProperties
	AdditionalPropertiesMonitorServiceAccount = "MonitorAccount"

	// lookup key in AdditionalProperties
	AdditionalPropertiesMigrationNetwork = "MigrationNetwork"

	// lookup key in AdditionalProperties
	AdditionalPropertiesPersistentReservationEnabled = "PersistentReservationEnabled"

	// lookup key in AdditionalProperties
	AdditionalPropertiesSynchronizationPort       = "SynchronizationPort"
	DefaultSynchronizationPort              int32 = 9185

	// account to use if one is not explicitly named
	DefaultMonitorAccount = "prometheus-k8s"

	// lookup keys in AdditionalProperties
	ImagePrefixKey      = "imagePrefix"
	ProductNameKey      = "productName"
	ProductComponentKey = "productComponent"
	ProductVersionKey   = "productVersion"

	// the regex used to parse the operator image
	operatorImageRegex = "^(.*)/(.*)virt-operator([@:].*)?$"

	// #nosec 101, the variable is not holding any credential
	// Prefix for env vars that will be passed along
	PassthroughEnvPrefix = "KV_IO_EXTRA_ENV_"
)

// DefaultMonitorNamespaces holds a set of well known prometheus-operator namespaces.
// Ordering in the list matters. First entries have precedence.
var DefaultMonitorNamespaces = []string{
	"openshift-monitoring", // default namespace in openshift
	"monitoring",           // default namespace of https://github.com/prometheus-operator/kube-prometheus
}

type KubeVirtDeploymentConfig struct {
	ID          string `json:"id,omitempty" optional:"true"`
	Namespace   string `json:"namespace,omitempty" optional:"true"`
	Registry    string `json:"registry,omitempty" optional:"true"`
	ImagePrefix string `json:"imagePrefix,omitempty" optional:"true"`

	// the KubeVirt version
	// matches the image tag, if tags are used, either by the manifest, or by the KubeVirt CR
	// used on the KubeVirt CR status and on annotations, and for determining up-/downgrade paths
	KubeVirtVersion string `json:"kubeVirtVersion,omitempty" optional:"true"`

	// the images names of every image we use
	VirtOperatorImage                  string `json:"virtOperatorImage,omitempty" optional:"true"`
	VirtApiImage                       string `json:"virtApiImage,omitempty" optional:"true"`
	VirtControllerImage                string `json:"virtControllerImage,omitempty" optional:"true"`
	VirtHandlerImage                   string `json:"virtHandlerImage,omitempty" optional:"true"`
	VirtLauncherImage                  string `json:"virtLauncherImage,omitempty" optional:"true"`
	VirtExportProxyImage               string `json:"virtExportProxyImage,omitempty" optional:"true"`
	VirtExportServerImage              string `json:"virtExportServerImage,omitempty" optional:"true"`
	VirtSynchronizationControllerImage string `json:"virtSynchronizationControllerImage,omitempty" optional:"true"`
	GsImage                            string `json:"GsImage,omitempty" optional:"true"`
	PrHelperImage                      string `json:"PrHelperImage,omitempty" optional:"true"`
	SidecarShimImage                   string `json:"SidecarShimImage,omitempty" optional:"true"`

	// everything else, which can e.g. come from KubeVirt CR spec
	AdditionalProperties map[string]string `json:"additionalProperties,omitempty" optional:"true"`

	// environment variables from virt-operator to pass along
	PassthroughEnvVars map[string]string `json:"passthroughEnvVars,omitempty" optional:"true"`
}

var DefaultEnvVarManager EnvVarManager = EnvVarManagerImpl{}

func GetConfigFromEnv() (*KubeVirtDeploymentConfig, error) {
	return GetConfigFromEnvWithEnvVarManager(DefaultEnvVarManager)
}

func GetConfigFromEnvWithEnvVarManager(envVarManager EnvVarManager) (*KubeVirtDeploymentConfig, error) {
	// first check if we have the new deployment config json
	c := envVarManager.Getenv(TargetDeploymentConfig)
	if c != "" {
		config := &KubeVirtDeploymentConfig{}
		if err := json.Unmarshal([]byte(c), config); err != nil {
			return nil, err
		}
		return config, nil
	}

	// for backwards compatibility: check for namespace and pullpolicy from deprecated env vars
	ns := envVarManager.Getenv(TargetInstallNamespace)
	if ns == "" {
		var err error
		ns, err = clientutil.GetNamespace()
		if err != nil {
			return nil, err
		}
	}

	pullPolicy := envVarManager.Getenv(TargetImagePullPolicy)
	additionalProperties := make(map[string]string)
	additionalProperties[AdditionalPropertiesNamePullPolicy] = pullPolicy

	return getConfig("", "", ns, additionalProperties, envVarManager), nil
}

func GetTargetConfigFromKV(kv *v1.KubeVirt) *KubeVirtDeploymentConfig {
	return GetTargetConfigFromKVWithEnvVarManager(kv, DefaultEnvVarManager)
}

func GetTargetConfigFromKVWithEnvVarManager(kv *v1.KubeVirt, envVarManager EnvVarManager) *KubeVirtDeploymentConfig {
	additionalProperties := getKVMapFromSpec(kv.Spec)
	if kv.Spec.Configuration.MigrationConfiguration != nil &&
		kv.Spec.Configuration.MigrationConfiguration.Network != nil {
		additionalProperties[AdditionalPropertiesMigrationNetwork] = *kv.Spec.Configuration.MigrationConfiguration.Network
	}
	if kv.Spec.Configuration.DeveloperConfiguration != nil && len(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates) > 0 {
		for _, v := range kv.Spec.Configuration.DeveloperConfiguration.FeatureGates {
			if v == featuregate.PersistentReservation {
				additionalProperties[AdditionalPropertiesPersistentReservationEnabled] = ""
			}
		}
	}
	// don't use status.target* here, as that is always set, but we need to know if it was set by the spec and with that
	// overriding shasums from env vars
	return getConfig(kv.Spec.ImageRegistry,
		kv.Spec.ImageTag,
		kv.Namespace,
		additionalProperties,
		envVarManager)
}

// retrieve imagePrefix from an existing deployment config (which is stored as JSON)
func getImagePrefixFromDeploymentConfig(deploymentConfig string) (string, bool, error) {
	var obj interface{}
	err := json.Unmarshal([]byte(deploymentConfig), &obj)
	if err != nil {
		return "", false, fmt.Errorf("unable to parse deployment config: %v", err)
	}
	for k, v := range obj.(map[string]interface{}) {
		if k == ImagePrefixKey {
			return v.(string), true, nil
		}
	}
	return "", false, nil
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
		if name == "ImagePullSecrets" {
			value, err := json.Marshal(v.Field(i).Interface())
			if err != nil {
				fmt.Printf("Cannot encode ImagePullsecrets to JSON %v", err)
			} else {
				kvMap[name] = string(value)
			}
			continue
		}
		value := v.Field(i).String()
		kvMap[name] = value
	}
	return kvMap
}

func GetOperatorImageWithEnvVarManager(envVarManager EnvVarManager) string {
	image := envVarManager.Getenv(VirtOperatorImageEnvName)
	if image != "" {
		return image
	}

	return envVarManager.Getenv(OldOperatorImageEnvName)
}

func getConfig(registry, tag, namespace string, additionalProperties map[string]string, envVarManager EnvVarManager) *KubeVirtDeploymentConfig {

	// get registry and tag/shasum from operator image
	imageString := GetOperatorImageWithEnvVarManager(envVarManager)
	imageRegEx := regexp.MustCompile(operatorImageRegex)
	matches := imageRegEx.FindAllStringSubmatch(imageString, 1)
	kubeVirtVersion := envVarManager.Getenv(KubeVirtVersionEnvName)
	if kubeVirtVersion == "" {
		kubeVirtVersion = "latest"
	}

	tagFromOperator := ""
	imagePrefix, useStoredImagePrefix := additionalProperties[ImagePrefixKey]

	if len(matches) == 1 {
		// only use registry from operator image if it was not given yet
		if registry == "" {
			registry = matches[0][1]
		}
		if !useStoredImagePrefix {
			imagePrefix = matches[0][2]
		}

		version := matches[0][3]
		if version == "" {
			tagFromOperator = "latest"
		} else if strings.HasPrefix(version, ":") {
			tagFromOperator = strings.TrimPrefix(version, ":")
		} else {
			// we have a shasum... chances are high that we get the shasums for the other images as well from env vars,
			// but as a fallback use latest tag
			tagFromOperator = kubeVirtVersion
		}

		if tag == "" {
			tag = tagFromOperator
		}
	} else {
		// operator image name has unexpected syntax.
		if tag == "" {
			tag = kubeVirtVersion
		}
	}

	passthroughEnv := GetPassthroughEnv()

	operatorImage := GetOperatorImageWithEnvVarManager(envVarManager)
	apiImage := envVarManager.Getenv(VirtApiImageEnvName)
	controllerImage := envVarManager.Getenv(VirtControllerImageEnvName)
	handlerImage := envVarManager.Getenv(VirtHandlerImageEnvName)
	launcherImage := envVarManager.Getenv(VirtLauncherImageEnvName)
	exportProxyImage := envVarManager.Getenv(VirtExportProxyImageEnvName)
	exportServerImage := envVarManager.Getenv(VirtExportServerImageEnvName)
	synchronizationControllerImage := envVarManager.Getenv(VirtSynchronizationControllerImageEnvName)
	GsImage := envVarManager.Getenv(GsImageEnvName)
	PrHelperImage := envVarManager.Getenv(PrHelperImageEnvName)
	SidecarShimImage := envVarManager.Getenv(SidecarShimImageEnvName)

	return newDeploymentConfigWithTag(registry, imagePrefix, tag, namespace, operatorImage, apiImage, controllerImage, handlerImage, launcherImage, exportProxyImage, exportServerImage, synchronizationControllerImage, GsImage, PrHelperImage, SidecarShimImage, additionalProperties, passthroughEnv)
}

func VerifyEnv() error {
	return VerifyEnvWithEnvVarManager(DefaultEnvVarManager)
}

func VerifyEnvWithEnvVarManager(envVarManager EnvVarManager) error {
	// ensure the operator image is valid
	imageString := GetOperatorImageWithEnvVarManager(envVarManager)
	if imageString == "" {
		return fmt.Errorf("cannot find virt-operator's image")
	}

	return nil
}

func GetPassthroughEnv() map[string]string {
	return GetPassthroughEnvWithEnvVarManager(DefaultEnvVarManager)
}

func GetPassthroughEnvWithEnvVarManager(envVarManager EnvVarManager) map[string]string {
	passthroughEnv := map[string]string{}

	for _, env := range envVarManager.Environ() {
		if strings.HasPrefix(env, PassthroughEnvPrefix) {
			split := strings.Split(env, "=")
			passthroughEnv[strings.TrimPrefix(split[0], PassthroughEnvPrefix)] = split[1]
		}
	}

	return passthroughEnv
}

func newDeploymentConfigWithTag(registry, imagePrefix, tag, namespace, operatorImage, apiImage, controllerImage, handlerImage, launcherImage, exportProxyImage, exportServerImage, synchronizationControllerImage, gsImage, prHelperImage, sidecarShimImage string, kvSpec, passthroughEnv map[string]string) *KubeVirtDeploymentConfig {
	c := &KubeVirtDeploymentConfig{
		Registry:                           registry,
		ImagePrefix:                        imagePrefix,
		KubeVirtVersion:                    tag,
		VirtOperatorImage:                  operatorImage,
		VirtApiImage:                       apiImage,
		VirtControllerImage:                controllerImage,
		VirtHandlerImage:                   handlerImage,
		VirtLauncherImage:                  launcherImage,
		VirtExportProxyImage:               exportProxyImage,
		VirtExportServerImage:              exportServerImage,
		VirtSynchronizationControllerImage: synchronizationControllerImage,
		GsImage:                            gsImage,
		PrHelperImage:                      prHelperImage,
		SidecarShimImage:                   sidecarShimImage,
		Namespace:                          namespace,
		AdditionalProperties:               kvSpec,
		PassthroughEnvVars:                 passthroughEnv,
	}
	c.generateInstallStrategyID()
	return c
}

func (c *KubeVirtDeploymentConfig) GetOperatorVersion() string {
	if digest := DigestFromImageName(c.VirtOperatorImage); digest != "" {
		return digest
	}

	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetApiVersion() string {
	if digest := DigestFromImageName(c.VirtApiImage); digest != "" {
		return digest
	}

	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetControllerVersion() string {
	if digest := DigestFromImageName(c.VirtControllerImage); digest != "" {
		return digest
	}

	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetHandlerVersion() string {
	if digest := DigestFromImageName(c.VirtHandlerImage); digest != "" {
		return digest
	}

	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetLauncherVersion() string {
	if digest := DigestFromImageName(c.VirtLauncherImage); digest != "" {
		return digest
	}

	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetExportProxyVersion() string {
	if digest := DigestFromImageName(c.VirtExportProxyImage); digest != "" {
		return digest
	}

	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetSynchronizationControllerVersion() string {
	if digest := DigestFromImageName(c.VirtSynchronizationControllerImage); digest != "" {
		return digest
	}

	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetExportServerVersion() string {
	if digest := DigestFromImageName(c.VirtExportServerImage); digest != "" {
		return digest
	}

	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetPrHelperVersion() string {
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetSidecarShimVersion() string {
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetKubeVirtVersion() string {
	return c.KubeVirtVersion
}

func (c *KubeVirtDeploymentConfig) GetImageRegistry() string {
	return c.Registry
}

func (c *KubeVirtDeploymentConfig) GetImagePrefix() string {
	return c.ImagePrefix
}

func (c *KubeVirtDeploymentConfig) GetExtraEnv() map[string]string {
	return c.PassthroughEnvVars
}

func (c *KubeVirtDeploymentConfig) SetTargetDeploymentConfig(kv *v1.KubeVirt) error {
	kv.Status.TargetKubeVirtVersion = c.GetKubeVirtVersion()
	kv.Status.TargetKubeVirtRegistry = c.GetImageRegistry()
	kv.Status.TargetDeploymentID = c.GetDeploymentID()
	json, err := c.GetJson()
	kv.Status.TargetDeploymentConfig = json
	return err
}

func (c *KubeVirtDeploymentConfig) SetDefaultArchitecture(kv *v1.KubeVirt) error {
	if kv.Spec.Configuration.ArchitectureConfiguration != nil && kv.Spec.Configuration.ArchitectureConfiguration.DefaultArchitecture != "" {
		kv.Status.DefaultArchitecture = kv.Spec.Configuration.ArchitectureConfiguration.DefaultArchitecture
	} else {
		// only set default architecture in status in the event that it has not been already set previously
		if kv.Status.DefaultArchitecture == "" {
			kv.Status.DefaultArchitecture = runtime.GOARCH
		}
	}

	return nil
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

func (c *KubeVirtDeploymentConfig) GetImagePullSecrets() []k8sv1.LocalObjectReference {
	var data []k8sv1.LocalObjectReference
	s, ok := c.AdditionalProperties[AdditionalPropertiesPullSecrets]
	if !ok {
		return data
	}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		fmt.Printf("Unable to parse imagePullSecrets: %v\n", err)
		if e, ok := err.(*json.SyntaxError); ok {
			fmt.Printf("syntax error at byte offset %d\n", e.Offset)
		}
		return data
	}
	return data
}

func (c *KubeVirtDeploymentConfig) PersistentReservationEnabled() bool {
	_, enabled := c.AdditionalProperties[AdditionalPropertiesPersistentReservationEnabled]
	return enabled
}

func (c *KubeVirtDeploymentConfig) GetMigrationNetwork() *string {
	value, enabled := c.AdditionalProperties[AdditionalPropertiesMigrationNetwork]
	if enabled {
		return &value
	} else {
		return nil
	}
}

func (c *KubeVirtDeploymentConfig) GetSynchronizationPort() int32 {
	value, enabled := c.AdditionalProperties[AdditionalPropertiesSynchronizationPort]
	if enabled {
		port, err := strconv.Atoi(value)
		if err != nil {
			log.Log.Errorf("Unable to convert %s to integer", value)
		} else {
			return int32(port)
		}

	}
	return DefaultSynchronizationPort
}

/*
if the monitoring namespace field is defiend in kubevirtCR than return it
otherwise we return common monitoring namespaces.
*/
func (c *KubeVirtDeploymentConfig) GetPotentialMonitorNamespaces() []string {
	p := c.AdditionalProperties[AdditionalPropertiesMonitorNamespace]
	if p == "" {
		return DefaultMonitorNamespaces
	}
	return []string{p}
}

func (c *KubeVirtDeploymentConfig) GetServiceMonitorNamespace() string {
	svcMonitorNs := c.AdditionalProperties[AdditionalPropertiesServiceMonitorNamespace]
	return svcMonitorNs
}

func (c *KubeVirtDeploymentConfig) GetMonitorServiceAccountName() string {
	p := c.AdditionalProperties[AdditionalPropertiesMonitorServiceAccount]
	if p == "" {
		return DefaultMonitorAccount
	}
	return p
}

func (c *KubeVirtDeploymentConfig) GetNamespace() string {
	return c.Namespace
}

func (c *KubeVirtDeploymentConfig) GetVerbosity() string {
	// not configurable yet
	return "2"
}

func (c *KubeVirtDeploymentConfig) GetProductComponent() string {
	return c.AdditionalProperties[ProductComponentKey]
}

func (c *KubeVirtDeploymentConfig) GetProductName() string {
	return c.AdditionalProperties[ProductNameKey]
}

func (c *KubeVirtDeploymentConfig) GetProductVersion() string {
	productVersion, ok := c.AdditionalProperties[ProductVersionKey]
	if !ok {
		return c.GetKubeVirtVersion()
	}
	return productVersion
}

func (c *KubeVirtDeploymentConfig) generateInstallStrategyID() {
	// We need an id, which identifies a KubeVirt deployment based on version, registry, namespace, and other
	// changeable properties from the KubeVirt CR. This will be used for identifying the correct install strategy job
	// and configmap
	// Calculate a sha over all those properties
	// #nosec CWE: 326 - Use of weak cryptographic primitive (http://cwe.mitre.org/data/definitions/326.html)
	// reason: sha1 is not used for encryption but for creating a hash value
	hasher := sha1.New()
	values := getStringFromFields(*c)
	hasher.Write([]byte(values))

	c.ID = hex.EncodeToString(hasher.Sum(nil))
}

// use KubeVirtDeploymentConfig by value because we modify sth just for the ID
func getStringFromFields(c KubeVirtDeploymentConfig) string {
	result := ""

	// image prefix might be empty. In order to get the same ID for missing and empty, remove an empty one
	if prefix, ok := c.AdditionalProperties[ImagePrefixKey]; ok && prefix == "" {
		delete(c.AdditionalProperties, ImagePrefixKey)
	}

	v := reflect.ValueOf(c)
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		result += fieldName
		field := v.Field(i)
		if field.Type().Kind() == reflect.Map {
			keys := field.MapKeys()
			nameKeys := make(map[string]reflect.Value, len(keys))
			names := make([]string, 0, len(keys))
			for _, key := range keys {
				name := key.String()
				if name == "" {
					continue
				}
				nameKeys[name] = key
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				key := nameKeys[name]
				val := field.MapIndex(key).String()
				result += name
				result += val
			}
		} else {
			value := v.Field(i).String()
			result += value
		}
	}
	return result
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

func NewEnvVarMap(envMap map[string]string) *[]k8sv1.EnvVar {
	env := []k8sv1.EnvVar{}

	for k, v := range envMap {
		env = append(env, k8sv1.EnvVar{Name: k, Value: v})
	}

	return &env
}

func IsValidLabel(label string) bool {
	// First and last character must be alphanumeric
	// middle chars can be alphanumeric, or dot hyphen or dash
	// entire string must not exceed 63 chars
	r := regexp.MustCompile(`^([a-z0-9A-Z]([a-z0-9A-Z\-\_\.]{0,61}[a-z0-9A-Z])?)?$`)
	return r.Match([]byte(label))
}

func DigestFromImageName(name string) (digest string) {
	if name != "" && strings.LastIndex(name, "@sha256:") != -1 {
		digest = strings.Split(name, "@sha256:")[1]
	}

	return
}
