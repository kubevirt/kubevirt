package nodeconfig

import (
	"context"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"github.com/go-logr/logr"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/metrics"
)

type kubeletConfig struct {
	gvkClient rest.Interface

	nodeGenerationCache map[string]int64

	KubeletConfig struct {
		NodeStatusMaxImages int `json:"nodeStatusMaxImages"`
	} `json:"kubeletconfig"`
}

func newKubeletConfig(gvkClient rest.Interface) *kubeletConfig {
	return &kubeletConfig{
		gvkClient:           gvkClient,
		nodeGenerationCache: make(map[string]int64),
	}
}

func (kc *kubeletConfig) setNodeNumberOfImagesMetrics(logger logr.Logger, node *corev1.Node) error {
	err := metrics.HcoMetrics.SetHCOMetricNumberOfImages(node.Name, len(node.Status.Images))
	if err != nil {
		logger.Error(err, "Failed to set number of images metric")
		return err
	}

	return nil
}

func (kc *kubeletConfig) setNodeStatusMaxImagesMetrics(ctx context.Context, logger logr.Logger, node *corev1.Node) error {
	if cachedGeneration, ok := kc.nodeGenerationCache[node.Name]; ok && cachedGeneration == node.Generation {
		return nil
	}

	resp, err := kc.gvkClient.Get().
		Resource("nodes").Name(node.Name).
		Suffix("proxy", "configz").
		Do(ctx).Raw()
	if err != nil {
		logger.Error(err, "Failed to get node configz")
		return err
	}

	err = json.Unmarshal(resp, &kc)
	if err != nil {
		logger.Error(err, "Failed to unmarshal kubelet configz")
		return err
	}

	err = metrics.HcoMetrics.SetHCOMetricNodeStatusMaxImages(node.Name, kc.KubeletConfig.NodeStatusMaxImages)
	if err != nil {
		logger.Error(err, "Failed to set node max images metric")
		return err
	}

	kc.nodeGenerationCache[node.Name] = node.Generation

	return nil
}
