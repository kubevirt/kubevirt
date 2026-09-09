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

package apiserver

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	kubevirtv1 "kubevirt.io/api/core/v1"
	virtclientgoscheme "kubevirt.io/client-go/kubevirt/scheme"
)

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(virtclientgoscheme.AddToScheme(scheme))

	subresourceSchemeBuilder := runtime.NewSchemeBuilder(
		kubevirtv1.AddKnownTypesGenerator(kubevirtv1.SubresourceGroupVersions),
	)
	utilruntime.Must(subresourceSchemeBuilder.AddToScheme(scheme))

	return scheme
}
