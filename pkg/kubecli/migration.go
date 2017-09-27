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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

func (k *kubevirt) Migration(namespace string) MigrationInterface {
	return &migrations{k.restClient, namespace, "migrations"}
}

type migrations struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (v *migrations) Get(name string, options k8smetav1.GetOptions) (migration *v1.Migration, err error) {
	migration = &v1.Migration{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(migration)
	migration.SetGroupVersionKind(v1.MigrationGroupVersionKind)
	return
}

func (v *migrations) List(options k8smetav1.ListOptions) (migrationList *v1.MigrationList, err error) {
	migrationList = &v1.MigrationList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(migrationList)
	for _, migration := range migrationList.Items {
		migration.SetGroupVersionKind(v1.MigrationGroupVersionKind)
	}

	return
}

func (v *migrations) Create(migration *v1.Migration) (result *v1.Migration, err error) {
	result = &v1.Migration{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(migration).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.MigrationGroupVersionKind)
	return
}

func (v *migrations) Update(migration *v1.Migration) (result *v1.Migration, err error) {
	result = &v1.Migration{}
	err = v.restClient.Put().
		Name(migration.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(migration).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.MigrationGroupVersionKind)
	return
}

func (v *migrations) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do().
		Error()
}
