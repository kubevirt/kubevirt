/*
 * This file is part of the KubeVirt project
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

package kubecli

import (
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/client-go/api/v1"
)

const vmiserviceSubresourceURL = "/apis/subresources.kubevirt.io/%s/namespaces/%s/vmiservices/%s/%s"

func (k *kubevirt) VMIService(namespace string) VMIServiceInterface {
	return &vmiservice{
		restClient: k.restClient,
		config:     k.config,
		clientSet:  k.Clientset,
		namespace:  namespace,
		resource:   "vmiservices",
	}
}

type vmiservice struct {
	restClient *rest.RESTClient
	config     *rest.Config
	clientSet  *kubernetes.Clientset
	namespace  string
	resource   string
	master     string
	kubeconfig string
}

func (v *vmiservice) Get(name string, options *k8smetav1.GetOptions) (vmi *v1.VMIService, err error) {
	vmiService := &v1.VMIService{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do().
		Into(vmiService)
	vmiService.SetGroupVersionKind(v1.VMIServiceGroupVersionKind)
	return
}

func (v *vmiservice) List(options *k8smetav1.ListOptions) (vmiList *v1.VMIServiceList, err error) {
	vmisList := &v1.VMIServiceList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(options, scheme.ParameterCodec).
		Do().
		Into(vmisList)
	for _, vmiService := range vmisList.Items {
		vmiService.SetGroupVersionKind(v1.VMIServiceGroupVersionKind)
	}

	return
}

func (v *vmiservice) Create(vmi *v1.VMIService) (result *v1.VMIService, err error) {
	result = &v1.VMIService{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vmi).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VMIServiceGroupVersionKind)
	return
}

func (v *vmiservice) Update(vmi *v1.VMIService) (result *v1.VMIService, err error) {
	result = &v1.VMIService{}
	err = v.restClient.Put().
		Name(vmi.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vmi).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VMIServiceGroupVersionKind)
	return
}

func (v *vmiservice) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (v *vmiservice) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VMIService, err error) {
	result = &v1.VMIService{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
