package scoped

import (
	v1 "k8s.io/api/core/v1"
)

// IsServiceAccountToken returns true if the secret is a valid api token for the service account
// This has been copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/serviceaccount/util.go
func IsServiceAccountToken(secret *v1.Secret, sa *v1.ServiceAccount) bool {
	if secret.Type != v1.SecretTypeServiceAccountToken {
		return false
	}

	name := secret.Annotations[v1.ServiceAccountNameKey]
	uid := secret.Annotations[v1.ServiceAccountUIDKey]
	if name != sa.Name {
		// Name must match
		return false
	}
	if len(uid) > 0 && uid != string(sa.UID) {
		// If UID is specified, it must match
		return false
	}

	return true
}
