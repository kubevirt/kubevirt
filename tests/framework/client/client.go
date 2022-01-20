package client

import (
	"k8s.io/client-go/util/flowcontrol"

	"kubevirt.io/client-go/kubecli"
)

func TestClientWithHighRateLimits() (kubecli.KubevirtClient, error) {
	config, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return nil, err
	}
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(40, 80)
	return kubecli.GetKubevirtClientFromRESTConfig(config)
}
