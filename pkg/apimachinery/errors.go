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

package apimachinery

// OptimisticLockErrorMsg is returned by kube-apiserver when trying to update an old version of a resource
// https://github.com/kubernetes/kubernetes/blob/b89f564539fad77cd22de1b155d84638daf8c83f/staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go#L240
const OptimisticLockErrorMsg = "the object has been modified; please apply your changes to the latest version and try again"
