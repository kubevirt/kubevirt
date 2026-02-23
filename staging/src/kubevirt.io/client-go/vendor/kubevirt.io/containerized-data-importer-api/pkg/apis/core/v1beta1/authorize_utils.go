/*
Copyright 2020 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"fmt"

	authentication "k8s.io/api/authentication/v1"
	authorization "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func newCloneSourceHandler(dataVolume *DataVolume, dsGet dsGetFunc) (CloneSourceHandler, error) {
	var pvcSource *DataVolumeSourcePVC
	var snapshotSource *DataVolumeSourceSnapshot

	if dataVolume.Spec.Source != nil {
		if dataVolume.Spec.Source.PVC != nil {
			pvcSource = dataVolume.Spec.Source.PVC
		} else if dataVolume.Spec.Source.Snapshot != nil {
			snapshotSource = dataVolume.Spec.Source.Snapshot
		}
	} else if dataVolume.Spec.SourceRef != nil && dataVolume.Spec.SourceRef.Kind == DataVolumeDataSource {
		ns := dataVolume.Namespace
		if dataVolume.Spec.SourceRef.Namespace != nil && *dataVolume.Spec.SourceRef.Namespace != "" {
			ns = *dataVolume.Spec.SourceRef.Namespace
		}
		dataSource, err := dsGet(ns, dataVolume.Spec.SourceRef.Name)
		if err != nil {
			return CloneSourceHandler{}, err
		}
		if dataSource.Spec.Source.PVC != nil {
			pvcSource = dataSource.Spec.Source.PVC
		} else if dataSource.Spec.Source.Snapshot != nil {
			snapshotSource = dataSource.Spec.Source.Snapshot
		}
	}

	switch {
	case pvcSource != nil:
		return CloneSourceHandler{
			CloneType:         pvcClone,
			TokenResource:     tokenResourcePvc,
			UserCloneAuthFunc: CanUserClonePVC,
			SACloneAuthFunc:   CanServiceAccountClonePVC,
			SourceName:        pvcSource.Name,
			SourceNamespace:   pvcSource.Namespace,
		}, nil
	case snapshotSource != nil:
		return CloneSourceHandler{
			CloneType:         snapshotClone,
			TokenResource:     tokenResourceSnapshot,
			UserCloneAuthFunc: CanUserCloneSnapshot,
			SACloneAuthFunc:   CanServiceAccountCloneSnapshot,
			SourceName:        snapshotSource.Name,
			SourceNamespace:   snapshotSource.Namespace,
		}, nil
	default:
		return CloneSourceHandler{
			CloneType: noClone,
		}, nil
	}
}

var (
	tokenResourcePvc = metav1.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "persistentvolumeclaims",
	}

	tokenResourceSnapshot = metav1.GroupVersionResource{
		Group:    "snapshot.storage.k8s.io",
		Version:  "v1",
		Resource: "volumesnapshots",
	}
)

type cloneType int

const (
	noClone cloneType = iota
	pvcClone
	snapshotClone
)

// CloneSourceHandler is a helper around determining the
// correct way of authorizing a particular DataVolume
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type CloneSourceHandler struct {
	CloneType         cloneType
	TokenResource     metav1.GroupVersionResource
	UserCloneAuthFunc UserCloneAuthFunc
	SACloneAuthFunc   ServiceAccountCloneAuthFunc
	SourceName        string
	SourceNamespace   string
}

// CloneAuthResponse contains various response details
// regarding authorizing a datavolume
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type CloneAuthResponse struct {
	Handler CloneSourceHandler
	Allowed bool
	Reason  string
}

type createSarFunc func(*authorization.SubjectAccessReview) (*authorization.SubjectAccessReview, error)
type dsGetFunc func(string, string) (*DataSource, error)

// AuthorizationHelperProxy proxies calls to APIs used for DV authorization
type AuthorizationHelperProxy interface {
	CreateSar(*authorization.SubjectAccessReview) (*authorization.SubjectAccessReview, error)
	GetNamespace(string) (*corev1.Namespace, error)
	GetDataSource(string, string) (*DataSource, error)
}

// UserCloneAuthFunc represents a user clone auth func
type UserCloneAuthFunc func(createSar createSarFunc, sourceNamespace, pvcName, targetNamespace string, userInfo authentication.UserInfo) (bool, string, error)

// ServiceAccountCloneAuthFunc represents a serviceaccount clone auth func
type ServiceAccountCloneAuthFunc func(createSar createSarFunc, pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error)

// CanUserClonePVC checks if a user has "appropriate" permission to clone from the given PVC
func CanUserClonePVC(createSar createSarFunc, sourceNamespace, pvcName, targetNamespace string,
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

	return sendSubjectAccessReviewsPvc(createSar, sourceNamespace, pvcName, sarSpec)
}

// CanServiceAccountClonePVC checks if a ServiceAccount has "appropriate" permission to clone from the given PVC
func CanServiceAccountClonePVC(createSar createSarFunc, pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
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

	return sendSubjectAccessReviewsPvc(createSar, pvcNamespace, pvcName, sarSpec)
}

// CanUserCloneSnapshot checks if a user has "appropriate" permission to clone from the given snapshot
func CanUserCloneSnapshot(createSar createSarFunc, sourceNamespace, pvcName, targetNamespace string,
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

	return sendSubjectAccessReviewsSnapshot(createSar, sourceNamespace, pvcName, sarSpec)
}

// CanServiceAccountCloneSnapshot checks if a ServiceAccount has "appropriate" permission to clone from the given snapshot
func CanServiceAccountCloneSnapshot(createSar createSarFunc, pvcNamespace, pvcName, saNamespace, saName string) (bool, string, error) {
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

	return sendSubjectAccessReviewsSnapshot(createSar, pvcNamespace, pvcName, sarSpec)
}

func sendSubjectAccessReviewsPvc(createSar createSarFunc, namespace, name string, sarSpec authorization.SubjectAccessReviewSpec) (bool, string, error) {
	allowed := false

	for _, ra := range getResourceAttributesPvc(namespace, name) {
		sar := &authorization.SubjectAccessReview{
			Spec: sarSpec,
		}
		sar.Spec.ResourceAttributes = &ra

		klog.V(3).Infof("Sending SubjectAccessReview %+v", sar)

		response, err := createSar(sar)
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

func sendSubjectAccessReviewsSnapshot(createSar createSarFunc, namespace, name string, sarSpec authorization.SubjectAccessReviewSpec) (bool, string, error) {
	// Either explicitly allowed
	sar := &authorization.SubjectAccessReview{
		Spec: sarSpec,
	}
	explicitResourceAttr := getExplicitResourceAttributeSnapshot(namespace, name)
	sar.Spec.ResourceAttributes = &explicitResourceAttr

	klog.V(3).Infof("Sending SubjectAccessReview %+v", sar)

	response, err := createSar(sar)
	if err != nil {
		return false, "", err
	}

	klog.V(3).Infof("SubjectAccessReview response %+v", response)

	if response.Status.Allowed {
		return true, "", nil
	}

	// Or both implicit conditions hold
	for _, ra := range getImplicitResourceAttributesSnapshot(namespace, name) {
		sar = &authorization.SubjectAccessReview{
			Spec: sarSpec,
		}
		sar.Spec.ResourceAttributes = &ra

		klog.V(3).Infof("Sending SubjectAccessReview %+v", sar)

		response, err = createSar(sar)
		if err != nil {
			return false, "", err
		}

		klog.V(3).Infof("SubjectAccessReview response %+v", response)

		if !response.Status.Allowed {
			return false, fmt.Sprintf("User %s has insufficient permissions in clone source namespace %s", sarSpec.User, namespace), nil
		}
	}

	return true, "", nil
}

func getResourceAttributesPvc(namespace, name string) []authorization.ResourceAttributes {
	return []authorization.ResourceAttributes{
		{
			Namespace:   namespace,
			Verb:        "create",
			Group:       SchemeGroupVersion.Group,
			Resource:    "datavolumes",
			Subresource: DataVolumeCloneSourceSubresource,
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

func getExplicitResourceAttributeSnapshot(namespace, name string) authorization.ResourceAttributes {
	return authorization.ResourceAttributes{
		Namespace:   namespace,
		Verb:        "create",
		Group:       SchemeGroupVersion.Group,
		Resource:    "datavolumes",
		Subresource: DataVolumeCloneSourceSubresource,
		Name:        name,
	}
}

func getImplicitResourceAttributesSnapshot(namespace, name string) []authorization.ResourceAttributes {
	return []authorization.ResourceAttributes{
		{
			Namespace: namespace,
			Verb:      "create",
			Resource:  "pods",
			Name:      name,
		},
		{
			Namespace: namespace,
			Verb:      "create",
			Resource:  "pvcs",
			Name:      name,
		},
	}
}
