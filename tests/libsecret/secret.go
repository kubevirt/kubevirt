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

package libsecret

import (
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type secretData interface {
	Data() map[string][]byte
	StringData() map[string]string
}

// New return Secret of Opaque type with "kubevirt.io/secret" label
func New(name string, data secretData) *kubev1.Secret {
	// secretLabel set this label to make the test suite namespace clean-up delete the secret on teardown
	const secretLabel = "kubevirt.io/secret" // #nosec G101
	return &kubev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				secretLabel: name,
			},
		},
		Data:       data.Data(),
		StringData: data.StringData(),
	}
}

type DataBytes map[string][]byte

func (s DataBytes) Data() map[string][]byte {
	return s
}

func (DataBytes) StringData() map[string]string {
	return nil
}

type DataString map[string]string

func (DataString) Data() map[string][]byte {
	return nil
}

func (s DataString) StringData() map[string]string {
	return s
}
