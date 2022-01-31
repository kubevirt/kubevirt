package util

import (
	"context"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

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
	IsManagedByOLM() bool
	IsControlPlaneHighlyAvailable() bool
	IsInfrastructureHighlyAvailable() bool
}

type ClusterInfoImp struct {
	runningInOpenshift            bool
	managedByOLM                  bool
	runningLocally                bool
	controlPlaneHighlyAvailable   bool
	infrastructureHighlyAvailable bool
	domain                        string
}

var clusterInfo ClusterInfo

var GetClusterInfo = func() ClusterInfo {
	return clusterInfo
}

// OperatorConditionNameEnvVar - this Env var is set by OLM, so the Operator can discover it's OperatorCondition.
const OperatorConditionNameEnvVar = "OPERATOR_CONDITION_NAME"

func (c *ClusterInfoImp) Init(ctx context.Context, cl client.Client, logger logr.Logger) error {
	err := c.queryCluster(ctx, cl, logger)
	if err != nil {
		return err
	}

	// We assume that this Operator is managed by OLM when this variable is present.
	_, c.managedByOLM = os.LookupEnv(OperatorConditionNameEnvVar)

	if c.runningInOpenshift {
		err = c.initOpenshift(ctx, cl, logger)
	} else {
		err = c.initKubernetes(cl)
	}

	return err
}

func (c *ClusterInfoImp) initKubernetes(cl client.Client) error {
	masterNodeList := &corev1.NodeList{}
	masterReq, err := labels.NewRequirement("node-role.kubernetes.io/master", selection.Exists, nil)
	if err != nil {
		return err
	}
	masterSelector := labels.NewSelector().Add(*masterReq)
	masterLabelSelector := client.MatchingLabelsSelector{Selector: masterSelector}
	err = cl.List(context.TODO(), masterNodeList, masterLabelSelector)
	if err != nil {
		return err
	}

	workerNodeList := &corev1.NodeList{}
	workerReq, err := labels.NewRequirement("node-role.kubernetes.io/worker", selection.Exists, nil)
	if err != nil {
		return err
	}
	workerSelector := labels.NewSelector().Add(*workerReq)
	workerLabelSelector := client.MatchingLabelsSelector{Selector: workerSelector}
	err = cl.List(context.TODO(), workerNodeList, workerLabelSelector)
	if err != nil {
		return err
	}

	c.controlPlaneHighlyAvailable = len(masterNodeList.Items) >= 3
	c.infrastructureHighlyAvailable = len(workerNodeList.Items) >= 2
	return nil
}

func (c *ClusterInfoImp) initOpenshift(ctx context.Context, cl client.Client, logger logr.Logger) error {
	clusterInfrastructure := &openshiftconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}
	err := cl.Get(ctx, client.ObjectKeyFromObject(clusterInfrastructure), clusterInfrastructure)
	if err != nil {
		return err
	}

	logger.Info("Cluster Infrastructure",
		"platform", clusterInfrastructure.Status.PlatformStatus.Type,
		"controlPlaneTopology", clusterInfrastructure.Status.ControlPlaneTopology,
		"infrastructureTopology", clusterInfrastructure.Status.InfrastructureTopology,
	)

	c.controlPlaneHighlyAvailable = clusterInfrastructure.Status.ControlPlaneTopology == openshiftconfigv1.HighlyAvailableTopologyMode
	c.infrastructureHighlyAvailable = clusterInfrastructure.Status.InfrastructureTopology == openshiftconfigv1.HighlyAvailableTopologyMode
	return nil
}

func (c ClusterInfoImp) IsManagedByOLM() bool {
	return c.managedByOLM
}

func (c ClusterInfoImp) IsOpenshift() bool {
	return c.runningInOpenshift
}

func (c ClusterInfoImp) IsRunningLocally() bool {
	return c.runningLocally
}

func (c ClusterInfoImp) IsControlPlaneHighlyAvailable() bool {
	return c.controlPlaneHighlyAvailable
}

func (c ClusterInfoImp) IsInfrastructureHighlyAvailable() bool {
	return c.infrastructureHighlyAvailable
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

func (c *ClusterInfoImp) queryCluster(ctx context.Context, cl client.Client, logger logr.Logger) error {
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
