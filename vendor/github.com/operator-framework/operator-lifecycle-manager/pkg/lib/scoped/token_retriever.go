package scoped

import (
	"fmt"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BearerTokenRetriever retrieves bearer token from a service account.
type BearerTokenRetriever struct {
	kubeclient operatorclient.ClientInterface
	logger     *logrus.Logger
}

// Retrieve returns the bearer token for API access from a given service account reference.
func (r *BearerTokenRetriever) Retrieve(reference *corev1.ObjectReference) (token string, err error) {
	logger := r.logger.WithFields(logrus.Fields{
		"sa":         reference.Name,
		"namespace":  reference.Namespace,
		logFieldName: logFieldValue,
	})

	sa, err := r.kubeclient.KubernetesInterface().CoreV1().ServiceAccounts(reference.Namespace).Get(reference.Name, metav1.GetOptions{})
	if err != nil {
		return
	}

	secret, err := getAPISecret(logger, r.kubeclient, sa)
	if err != nil {
		err = fmt.Errorf("error occurred while retrieving API secret associated with the service account sa=%s/%s - %v", sa.GetNamespace(), sa.GetName(), err)
		return
	}

	if secret == nil {
		err = fmt.Errorf("the service account does not have any API secret sa=%s/%s", sa.GetNamespace(), sa.GetName())
		return
	}

	token = string(secret.Data[corev1.ServiceAccountTokenKey])
	if token == "" {
		err = fmt.Errorf("the secret does not have any API token sa=%s/%s secret=%s", sa.GetNamespace(), sa.GetName(), secret.GetName())
	}

	return
}

func getAPISecret(logger *logrus.Entry, kubeclient operatorclient.ClientInterface, sa *corev1.ServiceAccount) (APISecret *corev1.Secret, err error) {
	for _, ref := range sa.Secrets {
		// corev1.ObjectReference only has Name populated.
		secret, getErr := kubeclient.KubernetesInterface().CoreV1().Secrets(sa.GetNamespace()).Get(ref.Name, metav1.GetOptions{})
		if getErr != nil {
			if k8serrors.IsNotFound(getErr) {
				logger.Warnf("skipping secret %s - %v", ref.Name, getErr)
				continue
			}

			err = getErr
			break
		}

		// Validate that this is a token for API access.
		if !IsServiceAccountToken(secret, sa) {
			logger.Warnf("skipping secret %s - %v", ref.Name, getErr)
			continue
		}

		// The first eligible secret that has an API access token is returned.
		APISecret = secret
		break
	}

	return
}
