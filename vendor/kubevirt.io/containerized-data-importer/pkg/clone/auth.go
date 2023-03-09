/*
 * This file is part of the CDI project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2019 Red Hat, Inc.
 *
 */

package clone

import (
	"fmt"

	authentication "k8s.io/api/authentication/v1"
	authorization "k8s.io/api/authorization/v1"
	"k8s.io/klog/v2"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

// SubjectAccessReviewsProxy proxies calls to work with SubjectAccessReviews
type SubjectAccessReviewsProxy interface {
	Create(*authorization.SubjectAccessReview) (*authorization.SubjectAccessReview, error)
}

// CanUserClonePVC checks if a user has "appropriate" permission to clone from the given PVC
func CanUserClonePVC(client SubjectAccessReviewsProxy, sourceNamespace, pvcName, targetNamespace string,
	userInfo authentication.UserInfo) (bool, string, error) {
	if sourceNamespace == targetNamespace {
		return true, "", nil
	}

	var newExtra map[string]authorization.ExtraValue
	if len(userInfo.Extra) > 0 {
		newExtra = make(map[string]authorization.ExtraValue)
		for k, v := range userInfo.Extra {
			newExtra[k] = authorization.ExtraValue(v)
		}
	}

	sarSpec := authorization.SubjectAccessReviewSpec{
		User:   userInfo.Username,
		Groups: userInfo.Groups,
		Extra:  newExtra,
	}

	return sendSubjectAccessReviews(client, sourceNamespace, pvcName, sarSpec)
}

// CanServiceAccountClonePVC checks if a ServiceAccount has "appropriate" permission to clone from the given PVC
func CanServiceAccountClonePVC(client SubjectAccessReviewsProxy, pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
	if pvcNamespace == saNamespace {
		return true, "", nil
	}

	user := fmt.Sprintf("system:serviceaccount:%s:%s", saNamespace, saName)

	sarSpec := authorization.SubjectAccessReviewSpec{
		User: user,
		Groups: []string{
			"system:serviceaccounts",
			"system:serviceaccounts:" + saNamespace,
			"system:authenticated",
		},
	}

	return sendSubjectAccessReviews(client, pvcNamespace, pvcName, sarSpec)
}

func sendSubjectAccessReviews(client SubjectAccessReviewsProxy, namespace, name string, sarSpec authorization.SubjectAccessReviewSpec) (bool, string, error) {
	allowed := false

	for _, ra := range getResourceAttributes(namespace, name) {
		sar := &authorization.SubjectAccessReview{
			Spec: sarSpec,
		}
		sar.Spec.ResourceAttributes = &ra

		klog.V(3).Infof("Sending SubjectAccessReview %+v", sar)

		response, err := client.Create(sar)
		if err != nil {
			return false, "", err
		}

		klog.V(3).Infof("SubjectAccessReview response %+v", response)

		if response.Status.Allowed {
			allowed = true
			break
		}
	}

	if !allowed {
		return false, fmt.Sprintf("User %s has insufficient permissions in clone source namespace %s", sarSpec.User, namespace), nil
	}

	return true, "", nil
}

func getResourceAttributes(namespace, name string) []authorization.ResourceAttributes {
	return []authorization.ResourceAttributes{
		{
			Namespace:   namespace,
			Verb:        "create",
			Group:       cdiv1.SchemeGroupVersion.Group,
			Resource:    "datavolumes",
			Subresource: cdiv1.DataVolumeCloneSourceSubresource,
			Name:        name,
		},
		{
			Namespace: namespace,
			Verb:      "create",
			Resource:  "pods",
			Name:      name,
		},
	}
}
