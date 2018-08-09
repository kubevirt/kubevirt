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

func NewClusterConfig(configMapInformer cache.Store) *ClusterConfig {
	c := &ClusterConfig{
		store: configMapInformer,
	}
	return c
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

func (c *ClusterConfig) GetImagePullPolicy() (policy v1.PullPolicy, err error) {
	var value string
	if value, err = getConfigMapEntry(c.store, imagePullPolicyKey); err != nil || value == "" {
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

func getConfigMapEntry(store cache.Store, key string) (string, error) {

	if obj, exists, err := store.GetByKey(configMapName); err != nil {
		return "", err
	} else if !exists {
		return "", nil
	} else {
		return obj.(*v1.ConfigMap).Data[key], nil
	}
}
