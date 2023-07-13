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
	"errors"
	"fmt"
	"net/url"
)

// ValidatedHTTPResponses checks the HTTP responses.
// It expects timeout errors, due to the throttling on the producer side.
// In case of unexpected errors or no errors at all it would fail,
// returning the first unexpected error if any, or a custom error in case
// there were no errors at all.
func ValidatedHTTPResponses(errorsChan chan error, concurrency int) error {
	var expectedErrorsCount = 0
	var unexpectedError error
	for ix := 0; ix < concurrency; ix++ {
		err := <-errorsChan
		if unexpectedError == nil && err != nil {
			var e *url.Error
			if errors.As(err, &e) && e.Timeout() {
				expectedErrorsCount++
			} else {
				unexpectedError = err
			}
		}
	}

	if unexpectedError == nil && expectedErrorsCount == 0 {
		return fmt.Errorf("timeout errors were expected due to throttling")
	}

	return unexpectedError
}
