package alerts

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/authorization"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	secretName = "hco-bearer-auth"
)

type secretReconciler struct {
	theSecret *corev1.Secret
}

func newSecretReconciler(namespace string, owner metav1.OwnerReference) *secretReconciler {
	return &secretReconciler{
		theSecret: NewSecret(namespace, owner),
	}
}

func (r *secretReconciler) Kind() string {
	return "Secret"
}

func (r *secretReconciler) ResourceName() string {
	return secretName
}

func (r *secretReconciler) GetFullResource() client.Object {
	return r.theSecret.DeepCopy()
}

func (r *secretReconciler) EmptyObject() client.Object {
	return &corev1.Secret{}
}

func (r *secretReconciler) UpdateExistingResource(ctx context.Context, cl client.Client, resource client.Object, logger logr.Logger) (client.Object, bool, error) {
	found := resource.(*corev1.Secret)
	modified := false

	token, err := authorization.CreateToken()
	if err != nil {
		return nil, false, err
	}

	if found.Data["token"] == nil || string(found.Data["token"]) != token {
		found.StringData = map[string]string{
			"token": token,
		}
		modified = true
	}

	modified = updateCommonDetails(&r.theSecret.ObjectMeta, &found.ObjectMeta) || modified

	if modified {
		err := cl.Update(ctx, found)
		if err != nil {
			logger.Error(err, "failed to update the Secret")
			return nil, false, err
		}
	}

	return found, modified, nil
}

func NewSecret(namespace string, owner metav1.OwnerReference) *corev1.Secret {
	token, err := authorization.CreateToken()
	if err != nil {
		logger.Error(err, "failed to create bearer token")
		return nil
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretName,
			Namespace:       namespace,
			Labels:          hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring),
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		StringData: map[string]string{
			"token": token,
		},
	}
}
