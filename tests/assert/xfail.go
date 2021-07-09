/*
 * This file is part of the kubevirt project
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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package assert

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// XFail replaces the default Fail handler with a XFail for one occurrence.
// XFail skips the test once it fails and leaves a XFAIL label on the message.
// It is useful to mark tests as XFail when one wants them to run even though they fail,
// monitoring and collecting information.
// 'condition' can be used to specify whether the test should be marked as XFail
func XFail(reason string, f func(), condition ...bool) {
	if len(condition) == 0 || condition[0] {
		defer RegisterFailHandler(Fail)
		RegisterFailHandler(func(m string, offset ...int) {
			depth := 0
			if len(offset) > 0 {
				depth = offset[0]
			}
			Skip(fmt.Sprintf("[XFAIL] %s, failure: %s", reason, m), depth+1)
		})
	}
	f()
}
