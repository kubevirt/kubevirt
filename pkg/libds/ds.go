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
 * Copyright The KubeVirt Authors.
 *
 */

package libds

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const (
	dsRandomNameLength = 12
	templateArchLabel  = "template.kubevirt.io/architecture"
)

type Option func(*v1beta1.DataSource)

func New(options ...Option) *v1beta1.DataSource {
	name := randName()
	ds := &v1beta1.DataSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cdi.kubevirt.io/v1beta1",
			Kind:       "DataSource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	for _, option := range options {
		option(ds)
	}
	return ds
}

func randName() string {
	return "test-datasource-" + rand.String(dsRandomNameLength)
}

func WithNamespace(namespace string) Option {
	return func(ds *v1beta1.DataSource) {
		ds.Namespace = namespace
	}
}

func WithName(name string) Option {
	return func(ds *v1beta1.DataSource) {
		ds.ObjectMeta.Name = name
	}
}

func WithTemplateArchLabel(val string) Option {
	return func(ds *v1beta1.DataSource) {
		if ds.ObjectMeta.Labels == nil {
			ds.ObjectMeta.Labels = make(map[string]string)
		}
		ds.ObjectMeta.Labels[templateArchLabel] = val
	}
}

type SourceOption func(*v1beta1.DataSourceSource)

func WithDataSourceSource(options ...SourceOption) Option {
	return func(ds *v1beta1.DataSource) {
		for _, option := range options {
			option(&ds.Spec.Source)
		}
	}
}

func WithDataSource(name, namespace string) SourceOption {
	return func(dss *v1beta1.DataSourceSource) {
		dss.DataSource = &v1beta1.DataSourceRefSourceDataSource{
			Name:      name,
			Namespace: namespace,
		}
	}
}
