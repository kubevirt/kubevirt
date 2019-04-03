package operatorsource

import (
	"context"

	marketplace "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"

	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SetupAppRegistryOptions generates an Options object based on the OperatorSource spec. It passes along
// the opsrc endpoint and, if defined, retrieves the authorization token from the specified Secret
// object.
func SetupAppRegistryOptions(client k8sclient.Client, spec *marketplace.OperatorSourceSpec, namespace string) (appregistry.Options, error) {
	options := appregistry.Options{
		Source: spec.Endpoint,
	}

	auth := spec.AuthorizationToken
	if auth.SecretName != "" {
		secret := corev1.Secret{}
		key := k8sclient.ObjectKey{
			Name:      auth.SecretName,
			Namespace: namespace,
		}
		err := client.Get(context.TODO(), key, &secret)
		if err != nil {
			return options, err
		}

		options.AuthToken = string(secret.Data["token"])
	}

	return options, nil
}
