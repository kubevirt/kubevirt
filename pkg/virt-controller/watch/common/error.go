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
package common

type SyncError interface {
	error
	Reason() string
	// RequiresRequeue indicates if the sync error should trigger a requeue, or
	// if information should just be added to the sync condition and a regular controller
	// wakeup will resolve the situation.
	RequiresRequeue() bool
}

func NewSyncError(err error, reason string) *syncErrorImpl {
	return &syncErrorImpl{err, reason}
}

type syncErrorImpl struct {
	err    error
	reason string
}

func (e *syncErrorImpl) Error() string {
	return e.err.Error()
}

func (e *syncErrorImpl) Reason() string {
	return e.reason
}

func (e *syncErrorImpl) RequiresRequeue() bool {
	return true
}
