package config

import (
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apimachinery/pkg/labels"

	"fmt"

	v12 "kubevirt.io/kubevirt/pkg/api/v1"
)

const configName = "kube-system/kubevirt-config"

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
	config, err := c.getConfig()
	if err != nil {
		return false, err
	}
	return config.IsUseEmulation(), nil
}

func (c *ClusterConfig) GetLabelNetworksForNode(node *v1.Node) (map[string]*v12.LabelNetwork, error) {
	config, err := c.getConfig()
	if err != nil {
		return nil, err
	}
	res := map[string]*v12.LabelNetwork{}

	for n, networks := range config.Networks {
		if networks.LabelNetwork != nil && matchNode(node, networks.LabelNetwork.Definitions) {
			res[n] = networks.LabelNetwork
		}
	}
	return res, nil
}

func (c *ClusterConfig) getConfig() (*v12.KubeVirtConfig, error) {
	obj, exists, err := c.store.GetByKey(configName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return &v12.KubeVirtConfig{}, nil
	}
	return obj.(*v12.KubeVirtConfig), nil
}

func matchNode(node *v1.Node, definitions []v12.LabelNetworkDefinition) bool {
	for _, def := range definitions {
		if labels.SelectorFromSet(labels.Set(def.NodeSelector)).Matches(labels.Set(node.Labels)) {
			return true
		}
	}
	return false
}

func (c *ClusterConfig) GetImagePullPolicy() (policy v1.PullPolicy, err error) {
	policy = v1.PullIfNotPresent // Default if not specified
	config, err := c.getConfig()
	if err != nil {
		return
	}
	if config.Developer == nil || config.Developer.ImagePullPolicy == "" {
		return
	}
	value := config.Developer.ImagePullPolicy
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
	return
}
