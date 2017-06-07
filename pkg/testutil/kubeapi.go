/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package testutil

import (
	"net/http"
	"strings"

	"github.com/onsi/gomega/ghttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubeapi "k8s.io/client-go/pkg/api"
	kubeapiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var TestPersistentVolumeClaimISCSI = kubeapiv1.PersistentVolumeClaim{
	TypeMeta: metav1.TypeMeta{
		Kind:       "PersistentVolumeClaim",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-pvc-iscsi",
		Namespace: kubeapi.NamespaceDefault,
	},
	Spec: kubeapiv1.PersistentVolumeClaimSpec{
		VolumeName: "test-pv-iscsi",
	},
	Status: kubeapiv1.PersistentVolumeClaimStatus{
		Phase: kubeapiv1.ClaimBound,
	},
}

var TestPersistentVolumeISCSI = kubeapiv1.PersistentVolume{
	TypeMeta: metav1.TypeMeta{
		Kind:       "PersistentVolume",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "test-pv-iscsi",
	},
	Spec: kubeapiv1.PersistentVolumeSpec{
		PersistentVolumeSource: kubeapiv1.PersistentVolumeSource{
			ISCSI: &kubeapiv1.ISCSIVolumeSource{
				IQN:          "iqn.2009-02.com.test:for.all",
				Lun:          1,
				TargetPortal: "127.0.0.1:6543",
			},
		},
	},
}

type Resource interface {
	runtime.Object
	metav1.ObjectMetaAccessor
}

func NewKubeServer(resources []Resource) *ghttp.Server {
	server := ghttp.NewServer()

	for _, res := range resources {
		AddServerResource(server, res)
	}

	return server
}

func NewKubeRESTClient(url string) (*rest.RESTClient, error) {
	gv := schema.GroupVersion{Group: "", Version: "v1"}
	restConfig, err := clientcmd.BuildConfigFromFlags(url, "")
	if err != nil {
		return nil, err
	}
	restConfig.GroupVersion = &gv
	restConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: kubeapi.Codecs}
	restConfig.APIPath = "/api"
	restConfig.ContentType = runtime.ContentTypeJSON
	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		return nil, err
	}
	return restClient, nil
}

func AddServerResource(server *ghttp.Server, res Resource) {
	meta := res.GetObjectMeta()
	typ := res.GetObjectKind()
	grp := typ.GroupVersionKind()

	var url string
	url = "/api/" + grp.GroupVersion().Version
	if meta.GetNamespace() != "" {
		url = url + "/namespaces/" + meta.GetNamespace()
	}
	url = url + "/" + strings.ToLower(grp.GroupKind().Kind) + "s/" + meta.GetName()

	server.AppendHandlers(ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", url),
		ghttp.RespondWithJSONEncoded(http.StatusOK, res),
	))
}
