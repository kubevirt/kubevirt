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

package api

import (
	"fmt"

	authentication "k8s.io/api/authentication/v1"
	authorization "k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

// CanClonePVC checks if a user has "appropriate" permission to clone from the given PVC
func CanClonePVC(client kubernetes.Interface, namespace, name string, userInfo authentication.UserInfo) (bool, string, error) {
	var newExtra map[string]authorization.ExtraValue
	if len(userInfo.Extra) > 0 {
		newExtra = make(map[string]authorization.ExtraValue)
		for k, v := range userInfo.Extra {
			newExtra[k] = authorization.ExtraValue(v)
		}
	}

	sar := &authorization.SubjectAccessReview{
		Spec: authorization.SubjectAccessReviewSpec{
			User:   userInfo.Username,
			Groups: userInfo.Groups,
			Extra:  newExtra,
			ResourceAttributes: &authorization.ResourceAttributes{
				Namespace:   namespace,
				Verb:        "create",
				Group:       cdiv1alpha1.SchemeGroupVersion.Group,
				Version:     cdiv1alpha1.SchemeGroupVersion.Version,
				Resource:    "datavolumes",
				Subresource: cdiv1alpha1.DataVolumeCloneSourceSubresource,
				Name:        name,
			},
		},
	}

	klog.V(3).Infof("Sending SubjectAccessReview %+v", sar)

	response, err := client.AuthorizationV1().SubjectAccessReviews().Create(sar)
	if err != nil {
		return false, "", err
	}

	klog.V(3).Infof("SubjectAccessReview response %+v", response)

	if !response.Status.Allowed {
		return false, fmt.Sprintf("User %s has insufficient permissions in clone source namespace %s", userInfo.Username, namespace), nil
	}

	return true, "", nil
}
