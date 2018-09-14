package utils

import (
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	SecretPollPeriod   = defaultPollPeriod
	SecretPollInterval = defaultPollInterval
)

func NewSecretDefinition(labels, stringData map[string]string, data map[string][]byte, ns, prefix string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix,
			Namespace:    ns,
			Labels:       labels,
		},
		StringData: stringData,
		Data:       data,
	}
}

func CreateSecretFromDefinition(c *kubernetes.Clientset, secret *v1.Secret) (*v1.Secret, error) {
	err := wait.PollImmediate(SecretPollInterval, SecretPollPeriod, func() (done bool, err error) {
		secret, err = c.CoreV1().Secrets(secret.Namespace).Create(secret)
		// success
		if err == nil {
			return true, nil
		}
		// fail if secret exists.
		if apierrs.IsAlreadyExists(err) {
			return true, err
		}
		// Log non-fatal errors
		glog.Error(errors.Wrapf(err, "Encountered create error for secret \"%s/%s\"", secret.Namespace, secret.Name))
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return secret, nil
}
