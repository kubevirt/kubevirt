package config

import (
	"strings"

	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

const configMapName = "kube-system/kubevirt-config"
const useEmulationKey = "debug.useEmulation"
const imagePullPolicyKey = "dev.imagePullPolicy"
const featureGatesKey = "feature-gates"

const (
	cpuManagerFeatureGate = "CPUManager"
)

func NewClusterConfig(configMapInformer cache.Store) *ClusterConfig {
	c := &ClusterConfig{
		store: configMapInformer,
	}
	return c
}

type ClusterConfig struct {
	store cache.Store
}

func (c *ClusterConfig) IsUseEmulation() bool {
	useEmulationValue := getConfigMapEntry(c.store, useEmulationKey)
	if useEmulationValue == "" {
		return false
	}
	if useEmulationValue == "" {
	}
	return (strings.ToLower(useEmulationValue) == "true")
}

func (c *ClusterConfig) GetImagePullPolicy() (policy v1.PullPolicy, err error) {
	var value string
	if value = getConfigMapEntry(c.store, imagePullPolicyKey); value == "" {
		policy = v1.PullIfNotPresent // Default if not specified
	} else {
		switch value {
		case "Always":
			policy = v1.PullAlways
		case "Never":
			policy = v1.PullNever
		case "IfNotPresent":
			policy = v1.PullIfNotPresent
		default:
			err = fmt.Errorf("Invalid ImagePullPolicy in ConfigMap: %s", value)
		}
	}
	return
}

func (c *ClusterConfig) GetFeatureGates() string {
	return getConfigMapEntry(c.store, featureGatesKey)
}

func getConfigMapEntry(store cache.Store, key string) string {
	if obj, exists, err := store.GetByKey(configMapName); err != nil {
		panic(fmt.Sprintf("Caches can't return errors, but got %v", err))
	} else if !exists {
		return ""
	} else {
		return obj.(*v1.ConfigMap).Data[key]
	}
}

func (c *ClusterConfig) CPUManagerEnabled() bool {
	featureGates := c.GetFeatureGates()
	return strings.Contains(featureGates, cpuManagerFeatureGate)
}
