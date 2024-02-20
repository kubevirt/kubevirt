package authorization

import (
	"context"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
)

func CanPatchNode(authClient authorizationv1.AuthorizationV1Interface, nodeName string) (bool, error) {
	review := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Resource: "nodes",
				Verb:     "patch",
				Name:     nodeName,
			},
		},
	}

	result, err := authClient.SelfSubjectAccessReviews().Create(context.Background(), review, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}

	return result.Status.Allowed, nil
}
