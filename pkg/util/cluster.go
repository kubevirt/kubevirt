package util

import (
	"errors"
	"github.com/go-logr/logr"
	secv1 "github.com/openshift/api/security/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"kubevirt.io/client-go/kubecli"
	"os"
	"strings"
)

type ClusterInfo interface {
	CheckRunningInOpenshift(logger logr.Logger, runningLocally bool) error
	IsOpenshift() bool
	IsRunningLocally() bool
}

type ClusterInfoImp struct {
	firstTime          bool
	runningInOpenshift bool
	runningLocally     bool
}

var clusterInfo ClusterInfo

func GetClusterInfo() ClusterInfo {
	return clusterInfo
}

func (c *ClusterInfoImp) CheckRunningInOpenshift(logger logr.Logger, runningLocally bool) error {

	if !c.firstTime {
		return nil
	}

	c.runningLocally = runningLocally

	virtCli, err := c.getKubevirtClient(logger, runningLocally)
	if err != nil {
		return err
	}

	c.runningInOpenshift = false
	clusterType := "kubernetes"

	_, apis, err := virtCli.DiscoveryClient().ServerGroupsAndResources()
	if err != nil {
		if !discovery.IsGroupDiscoveryFailedError(err) {
			logger.Error(err, "failed to get ServerGroupsAndResources")
			return err
		} else if discovery.IsGroupDiscoveryFailedError(err) {
			// In case of an error, check if security.openshift.io is the reason (unlikely).
			// If it is, we are obviously on an openshift cluster.
			// Otherwise we can do a positive check.
			e := err.(*discovery.ErrGroupDiscoveryFailed)
			if _, exists := e.Groups[secv1.GroupVersion]; exists {
				c.runningInOpenshift = true
				clusterType = "openshift"
			}
		}
	} else if c.findApi(apis, "securitycontextconstraints") {
		c.runningInOpenshift = true
		clusterType = "openshift"
	}

	logger.Info("Cluster type = " + clusterType)
	c.firstTime = false

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
		firstTime:          true,
		runningInOpenshift: false,
	}
}
