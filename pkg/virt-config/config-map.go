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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package virtconfig

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	configMapName          = "kubevirt-config"
	featureGateEnvVar      = "FEATURE_GATES"
	FeatureGatesKey        = "feature-gates"
	emulatedMachinesEnvVar = "VIRT_EMULATED_MACHINES"
	emulatedMachinesKey    = "emulated-machines"
	machineTypeKey         = "machine-type"
	useEmulationKey        = "debug.useEmulation"
	imagePullPolicyKey     = "dev.imagePullPolicy"
	migrationsConfigKey    = "migrations"
	cpuModelKey            = "default-cpu-model"
	cpuRequestKey          = "cpu-request"

	ParallelOutboundMigrationsPerNodeDefault uint32 = 2
	ParallelMigrationsPerClusterDefault      uint32 = 5
	BandwithPerMigrationDefault                     = "64Mi"
	MigrationProgressTimeout                 int64  = 150
	MigrationCompletionTimeoutPerGiB         int64  = 800
	DefaultMachineType                              = "q35"
	DefaultCPURequest                               = "100m"

	NodeDrainTaintDefaultKey = "kubevirt.io/drain"
)

// We cannot rely on automatic invocation of 'init' method because this initialization
// code assumes a cluster is available to pull the configmap from
func Init() {
	InitFromConfigMap(getConfigMap())
}

func InitFromConfigMap(cfgMap *k8sv1.ConfigMap) {
	if val, ok := cfgMap.Data[FeatureGatesKey]; ok {
		os.Setenv(featureGateEnvVar, val)
	}
	if val, ok := cfgMap.Data[emulatedMachinesKey]; ok {
		os.Setenv(emulatedMachinesEnvVar, val)
	}
}

func Clear() {
	os.Unsetenv(featureGateEnvVar)
	os.Unsetenv(emulatedMachinesEnvVar)
}

func getConfigMap() *k8sv1.ConfigMap {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	var cfgMap *k8sv1.ConfigMap
	err = wait.PollImmediate(time.Second*1, time.Second*10, func() (bool, error) {

		namespace, curErr := util.GetNamespace()
		if err != nil {
			return false, err
		}

		cfgMap, curErr = virtClient.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})

		if curErr != nil {
			if errors.IsNotFound(curErr) {
				logger := log.DefaultLogger()
				logger.Infof("%s ConfigMap does not exist. Using defaults.", configMapName)
				cfgMap = &k8sv1.ConfigMap{}
				return true, nil
			}
			return false, curErr
		}

		return true, nil
	})

	if err != nil {
		panic(err)
	}

	return cfgMap
}

// NewClusterConfig represents the `kubevirt-config` config map. It can be used to live-update
// values if the config changes. The config update works like this:
// 1. Check if the config exists. If it does not exist, return the default config
// 2. Check if the config got updated. If so, try to parse and return it
// 3. In case of errors or no updates (resource version stays the same), it returns the values from the last good config
func NewClusterConfig(configMapInformer cache.Store, namespace string) *ClusterConfig {

	c := &ClusterConfig{
		store:           configMapInformer,
		lock:            &sync.Mutex{},
		namespace:       namespace,
		lastValidConfig: defaultClusterConfig(),
		defaultConfig:   defaultClusterConfig(),
	}
	return c
}

func defaultClusterConfig() *Config {
	parallelOutboundMigrationsPerNodeDefault := ParallelOutboundMigrationsPerNodeDefault
	parallelMigrationsPerClusterDefault := ParallelMigrationsPerClusterDefault
	bandwithPerMigrationDefault := resource.MustParse(BandwithPerMigrationDefault)
	nodeDrainTaintDefaultKey := NodeDrainTaintDefaultKey
	progressTimeout := MigrationProgressTimeout
	completionTimeoutPerGiB := MigrationCompletionTimeoutPerGiB
	cpuRequestDefault := resource.MustParse(DefaultCPURequest)
	return &Config{
		ResourceVersion: "0",
		ImagePullPolicy: k8sv1.PullIfNotPresent,
		UseEmulation:    false,
		MigrationConfig: &MigrationConfig{
			ParallelMigrationsPerCluster:      &parallelMigrationsPerClusterDefault,
			ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNodeDefault,
			BandwidthPerMigration:             &bandwithPerMigrationDefault,
			NodeDrainTaintKey:                 &nodeDrainTaintDefaultKey,
			ProgressTimeout:                   &progressTimeout,
			CompletionTimeoutPerGiB:           &completionTimeoutPerGiB,
			UnsafeMigrationOverride:           false,
		},
		MachineType: DefaultMachineType,
		CPURequest:  cpuRequestDefault,
	}
}

type Config struct {
	ResourceVersion string
	UseEmulation    bool
	MigrationConfig *MigrationConfig
	ImagePullPolicy k8sv1.PullPolicy
	MachineType     string
	CPUModel        string
	CPURequest      resource.Quantity
}

type MigrationConfig struct {
	ParallelOutboundMigrationsPerNode *uint32            `json:"parallelOutboundMigrationsPerNode,omitempty"`
	ParallelMigrationsPerCluster      *uint32            `json:"parallelMigrationsPerCluster,omitempty"`
	BandwidthPerMigration             *resource.Quantity `json:"bandwidthPerMigration,omitempty"`
	NodeDrainTaintKey                 *string            `json:"nodeDrainTaintKey,omitempty"`
	ProgressTimeout                   *int64             `json:"progressTimeout,omitempty"`
	CompletionTimeoutPerGiB           *int64             `json:"completionTimeoutPerGiB,omitempty"`
	UnsafeMigrationOverride           bool               `json:"unsafeMigrationOverride"`
}

type ClusterConfig struct {
	store                            cache.Store
	namespace                        string
	lock                             *sync.Mutex
	lastValidConfig                  *Config
	defaultConfig                    *Config
	lastInvalidConfigResourceVersion string
}

func (c *ClusterConfig) IsUseEmulation() bool {
	return c.getConfig().UseEmulation
}

func (c *ClusterConfig) GetMigrationConfig() *MigrationConfig {
	return c.getConfig().MigrationConfig
}

func (c *ClusterConfig) GetImagePullPolicy() (policy k8sv1.PullPolicy) {
	return c.getConfig().ImagePullPolicy
}

func (c *ClusterConfig) GetMachineType() string {
	return c.getConfig().MachineType
}

func (c *ClusterConfig) GetCPUModel() string {
	return c.getConfig().CPUModel
}

func (c *ClusterConfig) GetCPURequest() resource.Quantity {
	return c.getConfig().CPURequest
}

// setConfig parses the provided config map and updates the provided config.
// Default values in the provided config stay in tact.
func setConfig(config *Config, configMap *k8sv1.ConfigMap) error {

	// set revision
	config.ResourceVersion = configMap.ResourceVersion

	// set migration options
	rawConfig := strings.TrimSpace(configMap.Data[migrationsConfigKey])
	if rawConfig != "" {
		// only sets values if they were specified, default values stay intact
		err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(rawConfig), 1024).Decode(config.MigrationConfig)
		if err != nil {
			return fmt.Errorf("failed to parse migration config: %v", err)
		}
	}

	// set image pull policy
	policy := strings.TrimSpace(configMap.Data[imagePullPolicyKey])
	switch policy {
	case "":
		// keep the default
	case "Always":
		config.ImagePullPolicy = k8sv1.PullAlways
	case "Never":
		config.ImagePullPolicy = k8sv1.PullNever
	case "IfNotPresent":
		config.ImagePullPolicy = k8sv1.PullIfNotPresent
	default:
		return fmt.Errorf("invalid dev.imagePullPolicy in config: %v", policy)
	}

	// set if emulation is used
	useEmulation := strings.TrimSpace(configMap.Data[useEmulationKey])
	switch useEmulation {
	case "":
		// keep the default
	case "true":
		config.UseEmulation = true
	case "false":
		config.UseEmulation = false
	default:
		return fmt.Errorf("invalid debug.useEmulation in config: %v", useEmulation)
	}

	// set machine type
	if machineType := strings.TrimSpace(configMap.Data[machineTypeKey]); machineType != "" {
		config.MachineType = machineType
	}

	if cpuModel := strings.TrimSpace(configMap.Data[cpuModelKey]); cpuModel != "" {
		config.CPUModel = cpuModel
	}

	if cpuRequest := strings.TrimSpace(configMap.Data[cpuRequestKey]); cpuRequest != "" {
		config.CPURequest = resource.MustParse(cpuRequest)
	}

	return nil
}

// getConfig returns the latest valid parsed config map result, or updates it
// if a newer version is available.
// XXX Rework this, to happen mostly in informer callbacks.
// This will also allow us then to react to config changes and e.g. restart some controllers
func (c *ClusterConfig) getConfig() (config *Config) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if obj, exists, err := c.store.GetByKey(c.namespace + "/" + configMapName); err != nil {
		log.DefaultLogger().Reason(err).Errorf("Error loading the cluster config from cache, falling back to last good resource version '%s'", c.lastValidConfig.ResourceVersion)
		return c.lastValidConfig
	} else if !exists {
		return c.defaultConfig
	} else {
		configMap := obj.(*k8sv1.ConfigMap)
		if c.lastValidConfig.ResourceVersion == configMap.ResourceVersion ||
			c.lastInvalidConfigResourceVersion == configMap.ResourceVersion {
			return c.lastValidConfig
		}
		config := defaultClusterConfig()
		if err := setConfig(config, configMap); err != nil {
			c.lastInvalidConfigResourceVersion = configMap.ResourceVersion
			log.DefaultLogger().Reason(err).Errorf("Invalid cluster config with resource version '%s', falling back to last good resource version '%s'", configMap.ResourceVersion, c.lastValidConfig.ResourceVersion)
			return c.lastValidConfig
		}
		log.DefaultLogger().Infof("Updating cluster config to resource version '%s'", configMap.ResourceVersion)
		c.lastValidConfig = config
		return c.lastValidConfig
	}
}
