/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package definitions

import "kubevirt.io/kubevirt/pkg/util/openapi"

var Validator = openapi.CreateOpenAPIValidator(ComposeAPIDefinitions())
