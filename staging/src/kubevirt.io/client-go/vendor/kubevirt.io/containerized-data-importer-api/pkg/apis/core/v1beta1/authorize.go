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
	"errors"

	authentication "k8s.io/api/authentication/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"

	"kubevirt.io/containerized-data-importer-api/pkg/apis/core"
)

const (
	// AnnPrePopulated is a PVC annotation telling the datavolume controller that the PVC is already populated
	AnnPrePopulated = core.GroupName + "/storage.prePopulated"
	// AnnCheckStaticVolume checks if a statically allocated PV exists before creating the target PVC.
	// If so, PVC is still created but population is skipped
	AnnCheckStaticVolume = core.GroupName + "/storage.checkStaticVolume"
)

// ErrNoTokenOkay indicates proceeding without token is allowed
// This error should only be of interest to entities that give out DataVolume tokens
var ErrNoTokenOkay = errors.New("proceeding without token is okay under the circumstances")

// AuthorizeUser indicates if the creating user is authorized to create the data volume
// For sources other than clone (import/upload/etc), this is a no-op
func (dv *DataVolume) AuthorizeUser(requestNamespace, requestName string, proxy AuthorizationHelperProxy, userInfo authentication.UserInfo) (CloneAuthResponse, error) {
	_, prePopulated := dv.Annotations[AnnPrePopulated]
	_, checkStaticVolume := dv.Annotations[AnnCheckStaticVolume]
	noTokenOkay := prePopulated || checkStaticVolume

	targetNamespace, targetName := dv.Namespace, dv.Name
	if targetNamespace == "" {
		targetNamespace = requestNamespace
	}
	if targetName == "" {
		targetName = requestName
	}

	cloneSourceHandler, err := newCloneSourceHandler(dv, proxy.GetDataSource)
	if err != nil {
		if k8serrors.IsNotFound(err) && noTokenOkay {
			// no token needed, likely since no datasource
			klog.V(3).Infof("DataVolume %s/%s is pre/static populated, not adding token, no datasource", targetNamespace, targetName)
			return CloneAuthResponse{Allowed: true, Reason: "", Handler: cloneSourceHandler}, ErrNoTokenOkay
		}
		return CloneAuthResponse{Allowed: false, Reason: "", Handler: cloneSourceHandler}, err
	}

	if cloneSourceHandler.CloneType == noClone {
		klog.V(3).Infof("DataVolume %s/%s not cloning", targetNamespace, targetName)
		return CloneAuthResponse{Allowed: true, Reason: "", Handler: cloneSourceHandler}, ErrNoTokenOkay
	}

	sourceName, sourceNamespace := cloneSourceHandler.SourceName, cloneSourceHandler.SourceNamespace
	if sourceNamespace == "" {
		sourceNamespace = targetNamespace
	}

	_, err = proxy.GetNamespace(sourceNamespace)
	if err != nil {
		if k8serrors.IsNotFound(err) && noTokenOkay {
			// no token needed, likely since no source namespace
			klog.V(3).Infof("DataVolume %s/%s is pre/static populated, not adding token, no source namespace", targetNamespace, targetName)
			return CloneAuthResponse{Allowed: true, Reason: "", Handler: cloneSourceHandler}, ErrNoTokenOkay
		}
		return CloneAuthResponse{Allowed: false, Reason: "", Handler: cloneSourceHandler}, err
	}

	ok, reason, err := cloneSourceHandler.UserCloneAuthFunc(proxy.CreateSar, sourceNamespace, sourceName, targetNamespace, userInfo)
	if err != nil {
		return CloneAuthResponse{Allowed: false, Reason: reason, Handler: cloneSourceHandler}, err
	}

	if !ok {
		if noTokenOkay {
			klog.V(3).Infof("DataVolume %s/%s is pre/static populated, not adding token, auth failed", targetNamespace, targetName)
			return CloneAuthResponse{Allowed: true, Reason: "", Handler: cloneSourceHandler}, ErrNoTokenOkay
		}
	}

	return CloneAuthResponse{Allowed: ok, Reason: reason, Handler: cloneSourceHandler}, err
}

// AuthorizeSA indicates if the creating ServiceAccount is authorized to create the data volume
// For sources other than clone (import/upload/etc), this is a no-op
func (dv *DataVolume) AuthorizeSA(requestNamespace, requestName string, proxy AuthorizationHelperProxy, saNamespace, saName string) (CloneAuthResponse, error) {
	_, prePopulated := dv.Annotations[AnnPrePopulated]
	_, checkStaticVolume := dv.Annotations[AnnCheckStaticVolume]
	noTokenOkay := prePopulated || checkStaticVolume

	targetNamespace, targetName := dv.Namespace, dv.Name
	if targetNamespace == "" {
		targetNamespace = requestNamespace
	}
	if saNamespace == "" {
		saNamespace = targetNamespace
	}
	if targetName == "" {
		targetName = requestName
	}

	cloneSourceHandler, err := newCloneSourceHandler(dv, proxy.GetDataSource)
	if err != nil {
		if k8serrors.IsNotFound(err) && noTokenOkay {
			// no token needed, likely since no datasource
			klog.V(3).Infof("DataVolume %s/%s is pre/static populated, not adding token, no datasource", targetNamespace, targetName)
			return CloneAuthResponse{Allowed: true, Reason: "", Handler: cloneSourceHandler}, ErrNoTokenOkay
		}
		return CloneAuthResponse{Allowed: false, Reason: "", Handler: cloneSourceHandler}, err
	}

	if cloneSourceHandler.CloneType == noClone {
		klog.V(3).Infof("DataVolume %s/%s not cloning", targetNamespace, targetName)
		return CloneAuthResponse{Allowed: true, Reason: "", Handler: cloneSourceHandler}, ErrNoTokenOkay
	}

	sourceName, sourceNamespace := cloneSourceHandler.SourceName, cloneSourceHandler.SourceNamespace
	if sourceNamespace == "" {
		sourceNamespace = targetNamespace
	}

	_, err = proxy.GetNamespace(sourceNamespace)
	if err != nil {
		if k8serrors.IsNotFound(err) && noTokenOkay {
			// no token needed, likely since no source namespace
			klog.V(3).Infof("DataVolume %s/%s is pre/static populated, not adding token, no source namespace", targetNamespace, targetName)
			return CloneAuthResponse{Allowed: true, Reason: "", Handler: cloneSourceHandler}, ErrNoTokenOkay
		}
		return CloneAuthResponse{Allowed: false, Reason: "", Handler: cloneSourceHandler}, err
	}

	ok, reason, err := cloneSourceHandler.SACloneAuthFunc(proxy.CreateSar, sourceNamespace, sourceName, saNamespace, saName)
	if err != nil {
		return CloneAuthResponse{Allowed: false, Reason: reason, Handler: cloneSourceHandler}, err
	}

	if !ok {
		if noTokenOkay {
			klog.V(3).Infof("DataVolume %s/%s is pre/static populated, not adding token, auth failed", targetNamespace, targetName)
			return CloneAuthResponse{Allowed: true, Reason: "", Handler: cloneSourceHandler}, ErrNoTokenOkay
		}
	}

	return CloneAuthResponse{Allowed: ok, Reason: reason, Handler: cloneSourceHandler}, err
}
