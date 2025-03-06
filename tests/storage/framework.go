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
 * Copyright the KubeVirt Authors.
 *
 */

package storage

import (
	. "github.com/onsi/ginkgo/v2"

	"kubevirt.io/kubevirt/tests/decorators"
)

func SIGDescribe(text string, args ...interface{}) bool {
	return Describe(SIG(text, args))
}

func FSIGDescribe(text string, args ...interface{}) bool {
	return FDescribe(SIG(text, args))
}

func PSIGDescribe(text string, args ...interface{}) bool {
	return PDescribe(SIG(text, args))
}

func SIG(text string, args ...interface{}) (extendedText string, newArgs []interface{}) {
	return decorators.SIG("[sig-storage]", decorators.SigStorage, text, args)
}
