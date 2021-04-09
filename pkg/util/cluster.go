package util

import (
	"context"
	"github.com/go-logr/logr"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterInfo interface {
	CheckRunningInOpenshift(creader client.Reader, ctx context.Context, logger logr.Logger, runningLocally bool) error
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

func (c *ClusterInfoImp) CheckRunningInOpenshift(creader client.Reader, ctx context.Context, logger logr.Logger, runningLocally bool) error {
	c.runningLocally = runningLocally
	isOpenShift := false
	version := ""

	clusterVersion := &openshiftconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
	}
	if err := creader.Get(ctx, client.ObjectKeyFromObject(clusterVersion), clusterVersion); err != nil {
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			// Not on OpenShift
			isOpenShift = false
		} else {
			logger.Error(err, "Failed to get ClusterVersion")
			return err
		}
	} else {
		isOpenShift = true
		version = clusterVersion.Status.Desired.Version
	}

	c.runningInOpenshift = isOpenShift
	if isOpenShift {
		logger.Info("Cluster type = openshift", "version", version)
	} else {
		logger.Info("Cluster type = kubernetes")
	}

	return nil
}

func (c ClusterInfoImp) IsOpenshift() bool {
	return c.runningInOpenshift
}

func (c ClusterInfoImp) IsRunningLocally() bool {
	return c.runningLocally
}

func init() {
	clusterInfo = &ClusterInfoImp{
		runningInOpenshift: false,
	}
}
