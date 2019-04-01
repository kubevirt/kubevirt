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
	useEmulationKey        = "debug.useEmulation"
	imagePullPolicyKey     = "dev.imagePullPolicy"
	migrationsConfigKey    = "migrations"

	ParallelOutboundMigrationsPerNodeDefault uint32 = 2
	ParallelMigrationsPerClusterDefault      uint32 = 5
	BandwithPerMigrationDefault                     = "64Mi"
)

// We cannot rely on automatic invocation of 'init' method because this initialization
// code assumes a cluster is available to pull the configmap from
func Init() {
	cfgMap := getConfigMap()
	if val, ok := cfgMap.Data[FeatureGatesKey]; ok {
		os.Setenv(featureGateEnvVar, val)
	}
	if val, ok := cfgMap.Data[emulatedMachinesKey]; ok {
		os.Setenv(emulatedMachinesEnvVar, val)
	}
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

func NewClusterConfig(configMapInformer cache.Store) *ClusterConfig {
	c := &ClusterConfig{
		store: configMapInformer,
	}
	return c
}

type MigrationConfig struct {
	ParallelOutboundMigrationsPerNode *uint32            `json:"parallelOutboundMigrationsPerNode,omitempty"`
	ParallelMigrationsPerCluster      *uint32            `json:"parallelMigrationsPerCluster,omitempty"`
	BandwidthPerMigration             *resource.Quantity `json:"bandwidthPerMigration,omitempty"`
}

type ClusterConfig struct {
	store cache.Store
}

func (c *ClusterConfig) IsUseEmulation() (bool, error) {
	useEmulationValue, err := getConfigMapEntry(c.store, useEmulationKey)
	if err != nil || useEmulationValue == "" {
		return false, err
	}
	if useEmulationValue == "" {
	}
	return (strings.ToLower(useEmulationValue) == "true"), nil
}

func (c *ClusterConfig) GetMigrationConfig() *MigrationConfig {

	parallelOutboundMigrationsPerNodeDefault := ParallelOutboundMigrationsPerNodeDefault
	parallelMigrationsPerClusterDefault := ParallelMigrationsPerClusterDefault
	bandwithPerMigrationDefault := resource.MustParse(BandwithPerMigrationDefault)
	defaultConfig := &MigrationConfig{
		ParallelMigrationsPerCluster:      &parallelMigrationsPerClusterDefault,
		ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNodeDefault,
		BandwidthPerMigration:             &bandwithPerMigrationDefault,
	}
	config, err := getConfigMapEntry(c.store, migrationsConfigKey)
	if err != nil || config == "" {
		return defaultConfig
	}

	_ = yaml.NewYAMLOrJSONDecoder(strings.NewReader(config), 1024).Decode(defaultConfig)
	return defaultConfig
}

func (c *ClusterConfig) GetImagePullPolicy() (policy k8sv1.PullPolicy, err error) {
	var value string
	if value, err = getConfigMapEntry(c.store, imagePullPolicyKey); err != nil || value == "" {
		policy = k8sv1.PullIfNotPresent // Default if not specified
	} else {
		switch value {
		case "Always":
			policy = k8sv1.PullAlways
		case "Never":
			policy = k8sv1.PullNever
		case "IfNotPresent":
			policy = k8sv1.PullIfNotPresent
		default:
			err = fmt.Errorf("Invalid ImagePullPolicy in ConfigMap: %s", value)
		}
	}
	return
}

func getConfigMapEntry(store cache.Store, key string) (string, error) {

	namespace, err := util.GetNamespace()
	if err != nil {
		return "", err
	}

	if obj, exists, err := store.GetByKey(namespace + "/" + configMapName); err != nil {
		return "", err
	} else if !exists {
		return "", nil
	} else {
		return obj.(*k8sv1.ConfigMap).Data[key], nil
	}
}
