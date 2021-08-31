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
	Init(ctx context.Context, cl client.Client, logger logr.Logger) error
	IsOpenshift() bool
	IsRunningLocally() bool
	GetDomain() string
}

type ClusterInfoImp struct {
	runningInOpenshift bool
	runningLocally     bool
	domain             string
}

var clusterInfo ClusterInfo

func GetClusterInfo() ClusterInfo {
	return clusterInfo
}

func (c *ClusterInfoImp) Init(ctx context.Context, cl client.Client, logger logr.Logger) error {
	clusterVersion := &openshiftconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
	}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(clusterVersion), clusterVersion); err != nil {
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			// Not on OpenShift
			c.runningInOpenshift = false
			logger.Info("Cluster type = kubernetes")
		} else {
			logger.Error(err, "Failed to get ClusterVersion")
			return err
		}
	} else {
		c.runningInOpenshift = true
		logger.Info("Cluster type = openshift", "version", clusterVersion.Status.Desired.Version)
		c.domain, err = getClusterDomain(ctx, cl)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c ClusterInfoImp) IsOpenshift() bool {
	return c.runningInOpenshift
}

func (c ClusterInfoImp) IsRunningLocally() bool {
	return c.runningLocally
}

func (c ClusterInfoImp) GetDomain() string {
	return c.domain
}

func getClusterDomain(ctx context.Context, cl client.Client) (string, error) {
	clusterIngress := &openshiftconfigv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(clusterIngress), clusterIngress); err != nil {
		return "", err
	}
	return clusterIngress.Spec.Domain, nil

}

func init() {
	clusterInfo = &ClusterInfoImp{
		runningLocally:     IsRunModeLocal(),
		runningInOpenshift: false,
	}
}
