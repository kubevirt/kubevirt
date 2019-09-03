package operator

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ghodss/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	mrv1 "kubevirt.io/machine-remediation-operator/pkg/apis/machineremediation/v1alpha1"
	"kubevirt.io/machine-remediation-operator/pkg/consts"
	"kubevirt.io/machine-remediation-operator/pkg/operator/components"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileMachineRemediationOperator) getDeployment(name string, namespace string) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	if err := r.client.Get(context.TODO(), key, deploy); err != nil {
		return nil, err
	}
	return deploy, nil
}

func (r *ReconcileMachineRemediationOperator) createOrUpdateDeployment(data *components.DeploymentData) error {
	if data.ImageRepository == "" {
		imageRepository, err := r.getOperatorImageRepository()
		if err != nil {
			return err
		}

		data.ImageRepository = imageRepository
	}

	if data.PullPolicy == "" {
		data.PullPolicy = corev1.PullIfNotPresent
	}

	newDeploy := components.NewDeployment(data)

	replicas, err := r.getReplicasCount()
	if err != nil {
		return err
	}
	newDeploy.Spec.Replicas = pointer.Int32Ptr(replicas)

	oldDeploy, err := r.getDeployment(data.Name, data.Namespace)
	if errors.IsNotFound(err) {
		if err := r.client.Create(context.TODO(), newDeploy); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	// do not override some user specific configuration
	newDeploy.Annotations = oldDeploy.Annotations
	newDeploy.Labels = oldDeploy.Labels
	newDeploy.Spec.Replicas = oldDeploy.Spec.Replicas

	// do not update the status, deployment controller one who responsible to update it
	newDeploy.Status = oldDeploy.Status
	return r.client.Update(context.TODO(), newDeploy)
}

func (r *ReconcileMachineRemediationOperator) getReplicasCount() (int32, error) {
	masterNodes := &corev1.NodeList{}
	if err := r.client.List(
		context.TODO(),
		masterNodes,
		client.InNamespace(consts.NamespaceOpenshiftMachineAPI),
		client.MatchingLabels(map[string]string{consts.MasterRoleLabel: ""}),
	); err != nil {
		return 0, err
	}
	if len(masterNodes.Items) < 2 {
		return 1, nil
	}
	return 2, nil
}

func (r *ReconcileMachineRemediationOperator) getOperatorImageRepository() (string, error) {
	ns, err := getOperatorNamespace()
	if err != nil {
		return "", err
	}

	operator, err := r.getDeployment(components.ComponentMachineRemediationOperator, ns)
	if err != nil {
		return "", err
	}

	image := strings.Split(operator.Spec.Template.Spec.Containers[0].Image, "/")
	return strings.Join(image[:len(image)-1], "/"), nil
}

func getOperatorNamespace() (string, error) {
	data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("failed to get operator namespace: %v", err)
	}

	if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
		return ns, nil
	}

	return "", fmt.Errorf("failed to get operator namespace: %v", err)
}

func (r *ReconcileMachineRemediationOperator) deleteDeployment(name string, namespace string) error {
	deploy, err := r.getDeployment(name, namespace)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.client.Delete(context.TODO(), deploy)
}

func (r *ReconcileMachineRemediationOperator) isDeploymentReady(name string, namespace string) (bool, error) {
	d, err := r.getDeployment(name, namespace)
	if err != nil {
		return false, err
	}
	if d.Generation <= d.Status.ObservedGeneration &&
		d.Status.Replicas == *d.Spec.Replicas &&
		d.Status.UpdatedReplicas == d.Status.Replicas &&
		d.Status.UnavailableReplicas == 0 {
		return true, nil
	}
	return false, nil
}

func (r *ReconcileMachineRemediationOperator) getServiceAccount(name string, namespace string) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	if err := r.client.Get(context.TODO(), key, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

func (r *ReconcileMachineRemediationOperator) createOrUpdateServiceAccount(name string, namespace string) error {
	newServiceAccount := components.NewServiceAccount(name, namespace, r.operatorVersion)

	_, err := r.getServiceAccount(name, namespace)
	if errors.IsNotFound(err) {
		if err := r.client.Create(context.TODO(), newServiceAccount); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	return r.client.Update(context.TODO(), newServiceAccount)
}

func (r *ReconcileMachineRemediationOperator) deleteServiceAccount(name string, namespace string) error {
	sa, err := r.getServiceAccount(name, namespace)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.client.Delete(context.TODO(), sa)
}

func (r *ReconcileMachineRemediationOperator) getClusterRole(name string) (*rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceNone,
	}
	if err := r.client.Get(context.TODO(), key, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

func (r *ReconcileMachineRemediationOperator) createOrUpdateClusterRole(name string) error {
	newClusterRole := components.NewClusterRole(name, components.Rules[name], r.operatorVersion)

	_, err := r.getClusterRole(name)
	if errors.IsNotFound(err) {
		if err := r.client.Create(context.TODO(), newClusterRole); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	return r.client.Update(context.TODO(), newClusterRole)
}

func (r *ReconcileMachineRemediationOperator) deleteClusterRole(name string) error {
	cr, err := r.getClusterRole(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.client.Delete(context.TODO(), cr)
}

func (r *ReconcileMachineRemediationOperator) getClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceNone,
	}
	if err := r.client.Get(context.TODO(), key, crb); err != nil {
		return nil, err
	}
	return crb, nil
}

func (r *ReconcileMachineRemediationOperator) createOrUpdateClusterRoleBinding(name string, namespace string) error {
	newClusterRoleBinding := components.NewClusterRoleBinding(name, namespace, r.operatorVersion)

	_, err := r.getClusterRoleBinding(name)
	if errors.IsNotFound(err) {
		if err := r.client.Create(context.TODO(), newClusterRoleBinding); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	return r.client.Update(context.TODO(), newClusterRoleBinding)
}

func (r *ReconcileMachineRemediationOperator) deleteClusterRoleBinding(name string) error {
	crb, err := r.getClusterRoleBinding(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.client.Delete(context.TODO(), crb)
}

func getCustomResourceDefinitionFilePath(name string, dir string) string {
	return fmt.Sprintf("%s/%s_%s_%s.yaml", dir, "machineremediation", mrv1.SchemeGroupVersion.Version, name)
}

func (r *ReconcileMachineRemediationOperator) getCustomResourceDefinition(kind string) (*extv1beta1.CustomResourceDefinition, error) {
	crd := &extv1beta1.CustomResourceDefinition{}
	key := types.NamespacedName{
		Name:      fmt.Sprintf("%ss.%s", kind, mrv1.SchemeGroupVersion.Group),
		Namespace: metav1.NamespaceNone,
	}
	if err := r.client.Get(context.TODO(), key, crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func (r *ReconcileMachineRemediationOperator) createOrUpdateCustomResourceDefinition(kind string) error {
	newCRD := &extv1beta1.CustomResourceDefinition{}
	crdFile, err := ioutil.ReadFile(getCustomResourceDefinitionFilePath(kind, r.crdsManifestsDir))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(crdFile, newCRD); err != nil {
		return err
	}

	oldCRD, err := r.getCustomResourceDefinition(kind)
	if errors.IsNotFound(err) {
		if err := r.client.Create(context.TODO(), newCRD); err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}

	newCRD.ResourceVersion = oldCRD.ResourceVersion
	return r.client.Update(context.TODO(), newCRD)
}

func (r *ReconcileMachineRemediationOperator) deleteCustomResourceDefinition(kind string) error {
	crd, err := r.getCustomResourceDefinition(kind)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.client.Delete(context.TODO(), crd)
}
