package util

import (
	"context"
	"github.com/go-logr/logr"
	securityv1 "github.com/openshift/api/security/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterInfo interface {
	CheckRunningInOpenshift(ctx context.Context, logger logr.Logger) error
	IsOpenshift() bool
}

type ClusterInfoImp struct {
	client             client.Reader
	firstTime          bool
	runningInOpenshift bool
}

func NewClusterInfo(c client.Reader) ClusterInfo {
	return &ClusterInfoImp{
		client:             c,
		firstTime:          true,
		runningInOpenshift: false,
	}
}

func (c *ClusterInfoImp) CheckRunningInOpenshift(ctx context.Context, logger logr.Logger) error {
	if !c.firstTime {
		return nil
	}

	scc := &securityv1.SecurityContextConstraintsList{}

	err := c.client.List(ctx, scc)
	clusterType := ""
	if err != nil {
		if meta.IsNoMatchError(err) {
			c.runningInOpenshift = false
			clusterType = "kubernetes"
		} else {
			logger.Error(err, "failed to read SecurityContextConstraints")
			return err
		}
	} else {
		c.runningInOpenshift = true
		clusterType = "openshift"
	}

	logger.Info("Cluster type = " + clusterType)
	c.firstTime = false

	return nil
}

func (c ClusterInfoImp) IsOpenshift() bool {
	return c.runningInOpenshift
}
