package util

import (
	"errors"
	"github.com/go-logr/logr"
	secv1 "github.com/openshift/api/security/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/util/cluster"
	"os"
	"strings"
)

type ClusterInfo interface {
	CheckRunningInOpenshift(logger logr.Logger, runningLocally bool) error
	IsOpenshift() bool
	IsRunningLocally() bool
}

type ClusterInfoImp struct {
	runningInOpenshift bool
	runningLocally     bool
}

var clusterInfo ClusterInfo

func GetClusterInfo() ClusterInfo {
	return clusterInfo
}

func (c *ClusterInfoImp) CheckRunningInOpenshift(logger logr.Logger, runningLocally bool) error {
	c.runningLocally = runningLocally

	virtClient, err := c.getKubevirtClient(logger, runningLocally)
	if err != nil {
		return err
	}

	isOpenShift, err := cluster.IsOnOpenShift(virtClient)
	if err != nil {
		return err
	}

	c.runningInOpenshift = isOpenShift
	if isOpenShift {
		logger.Info("Cluster type = openshift")
	} else {
		logger.Info("Cluster type = kubernetes")
	}

	return nil
}

func (c *ClusterInfoImp) getKubevirtClient(logger logr.Logger, runningLocally bool) (kubecli.KubevirtClient, error) {
	var (
		virtCli kubecli.KubevirtClient
		err     error
	)

	kubecli.Init()
	if runningLocally {
		kubeconfig, ok := os.LookupEnv("KUBECONFIG")
		if ok {
			virtCli, err = kubecli.GetKubevirtClientFromFlags("", kubeconfig)
			if err != nil {
				logger.Error(err, "failed to get KubevirtClient From Flags", "kubeconfig", kubeconfig)
				return nil, err
			}
		} else {
			const errMsg = "KUBECONFIG environment variable is not defined"
			err = errors.New(errMsg)
			logger.Error(err, errMsg)
			return nil, err
		}
	} else {
		virtCli, err = kubecli.GetKubevirtClient()
		if err != nil {
			logger.Error(err, "failed to get KubevirtClient")
			return nil, err
		}
	}
	return virtCli, nil
}

func (c ClusterInfoImp) IsOpenshift() bool {
	return c.runningInOpenshift
}

func (c ClusterInfoImp) IsRunningLocally() bool {
	return c.runningLocally
}

func (c ClusterInfoImp) findApi(apis []*metav1.APIResourceList, resourceName string) bool {
	for _, api := range apis {
		if api.GroupVersion == secv1.GroupVersion.String() {
			for _, resource := range api.APIResources {
				if strings.ToLower(resource.Name) == resourceName {
					return true
				}
			}
		}
	}

	return false
}

func init() {
	clusterInfo = &ClusterInfoImp{
		runningInOpenshift: false,
	}
}
