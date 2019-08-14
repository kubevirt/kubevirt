//go:generate counterfeiter deployment_install_client.go InstallStrategyDeploymentInterface
package wrappers

import (
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1beta1rbac "k8s.io/api/rbac/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ErrNilObject = errors.New("Bad object supplied: <nil>")

type InstallStrategyDeploymentInterface interface {
	CreateRole(role *v1beta1rbac.Role) (*v1beta1rbac.Role, error)
	CreateRoleBinding(roleBinding *v1beta1rbac.RoleBinding) (*v1beta1rbac.RoleBinding, error)
	EnsureServiceAccount(serviceAccount *corev1.ServiceAccount, owner ownerutil.Owner) (*corev1.ServiceAccount, error)
	CreateDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error)
	CreateOrUpdateDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error)
	DeleteDeployment(name string) error
	GetServiceAccountByName(serviceAccountName string) (*corev1.ServiceAccount, error)
	FindAnyDeploymentsMatchingNames(depNames []string) ([]*appsv1.Deployment, error)
}

type InstallStrategyDeploymentClientForNamespace struct {
	opClient  operatorclient.ClientInterface
	Namespace string
}

var _ InstallStrategyDeploymentInterface = &InstallStrategyDeploymentClientForNamespace{}

func NewInstallStrategyDeploymentClient(opClient operatorclient.ClientInterface, namespace string) InstallStrategyDeploymentInterface {
	return &InstallStrategyDeploymentClientForNamespace{
		opClient:  opClient,
		Namespace: namespace,
	}
}

func (c *InstallStrategyDeploymentClientForNamespace) CreateRole(role *v1beta1rbac.Role) (*v1beta1rbac.Role, error) {
	return c.opClient.KubernetesInterface().RbacV1beta1().Roles(c.Namespace).Create(role)
}

func (c *InstallStrategyDeploymentClientForNamespace) CreateRoleBinding(roleBinding *v1beta1rbac.RoleBinding) (*v1beta1rbac.RoleBinding, error) {
	return c.opClient.KubernetesInterface().RbacV1beta1().RoleBindings(c.Namespace).Create(roleBinding)
}

func (c *InstallStrategyDeploymentClientForNamespace) EnsureServiceAccount(serviceAccount *corev1.ServiceAccount, owner ownerutil.Owner) (*corev1.ServiceAccount, error) {
	if serviceAccount == nil {
		return nil, ErrNilObject
	}

	foundAccount, err := c.opClient.GetServiceAccount(c.Namespace, serviceAccount.Name)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "checking for existing serviceacccount failed")
	}

	// create if not found
	if err != nil && apierrors.IsNotFound(err) {
		serviceAccount.SetNamespace(c.Namespace)
		createdAccount, err := c.opClient.CreateServiceAccount(serviceAccount)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return nil, errors.Wrap(err, "creating serviceacccount failed")
		}
		if apierrors.IsAlreadyExists(err) {
			return serviceAccount, nil
		}
		return createdAccount, nil
	}

	// if found, ensure ownerreferences
	if ownerutil.IsOwnedBy(foundAccount, owner) {
		return foundAccount, nil
	}
	// set owner if missing
	ownerutil.AddNonBlockingOwner(foundAccount, owner)
	return c.opClient.UpdateServiceAccount(foundAccount)
}

func (c *InstallStrategyDeploymentClientForNamespace) CreateDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return c.opClient.CreateDeployment(deployment)
}

func (c *InstallStrategyDeploymentClientForNamespace) DeleteDeployment(name string) error {
	foregroundDelete := metav1.DeletePropagationForeground // cascading delete
	immediate := int64(0)
	immediateForegroundDelete := &metav1.DeleteOptions{GracePeriodSeconds: &immediate, PropagationPolicy: &foregroundDelete}
	return c.opClient.DeleteDeployment(c.Namespace, name, immediateForegroundDelete)
}

func (c *InstallStrategyDeploymentClientForNamespace) CreateOrUpdateDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	d, _, err := c.opClient.CreateOrRollingUpdateDeployment(deployment)
	return d, err
}

func (c *InstallStrategyDeploymentClientForNamespace) GetServiceAccountByName(serviceAccountName string) (*corev1.ServiceAccount, error) {
	return c.opClient.KubernetesInterface().CoreV1().ServiceAccounts(c.Namespace).Get(serviceAccountName, metav1.GetOptions{})
}

func (c *InstallStrategyDeploymentClientForNamespace) FindAnyDeploymentsMatchingNames(depNames []string) ([]*appsv1.Deployment, error) {
	var deployments []*appsv1.Deployment
	for _, depName := range depNames {
		fetchedDep, err := c.opClient.GetDeployment(c.Namespace, depName)

		if err == nil {
			deployments = append(deployments, fetchedDep)
		} else {
			// Any errors other than !exists are propagated up
			if !apierrors.IsNotFound(err) {
				return deployments, err
			}
		}
	}
	return deployments, nil
}
