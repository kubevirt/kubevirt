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

package instancetype

type IgnoreableInferenceError struct {
	err error
}

func (e *IgnoreableInferenceError) Error() string {
	return e.err.Error()
}

func (e *IgnoreableInferenceError) Unwrap() error {
	return e.err
}

func NewIgnoreableInferenceError(err error) error {
	return &IgnoreableInferenceError{err: err}
}
