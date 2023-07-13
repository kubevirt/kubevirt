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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libinfra

import (
	"reflect"

	"github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

func ContainsCrt(bundle []byte, containedCrt []byte) bool {
	crts, err := cert.ParseCertsPEM(bundle)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	attached := false
	for _, crt := range crts {
		crtBytes := cert.EncodeCertPEM(crt)
		if reflect.DeepEqual(crtBytes, containedCrt) {
			attached = true
			break
		}
	}
	return attached
}
