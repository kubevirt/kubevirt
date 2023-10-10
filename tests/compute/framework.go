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
package compute

import (
	ginkgo "github.com/onsi/ginkgo/v2"

	"kubevirt.io/kubevirt/tests/decorators"
)

func SIGDescribe(text string, args ...interface{}) bool {
	return ginkgo.Describe("[sig-compute] "+text, decorators.SigCompute, args)
}

func FSIGDescribe(text string, args ...interface{}) bool {
	return ginkgo.FDescribe("[sig-compute] "+text, decorators.SigCompute, args)
}
