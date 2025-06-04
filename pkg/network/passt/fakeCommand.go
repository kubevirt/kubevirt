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

package passt

import (
	"context"
	"time"
)

type StartError struct{}

func (StartError) Error() string { return "fake command start error" }

type WaitError struct{}

func (WaitError) Error() string { return "fake command wait error" }

type FakeCommand struct {
	ctx context.Context
	isStartError,
	isWaitError,
	isStartCalled,
	isWaitCalled bool
}

func NewFakeCommand(ctx context.Context, isStartError, isWaitError bool) *FakeCommand {
	return &FakeCommand{
		ctx:          ctx,
		isStartError: isStartError,
		isWaitError:  isWaitError,
	}
}

func (f *FakeCommand) Start() error {
	f.isStartCalled = true
	if f.isStartError {
		return StartError{}
	}
	return nil
}

func (f *FakeCommand) Wait() error {
	f.isWaitCalled = true
	select {
	case <-time.After(time.Second):
		if f.isWaitError {
			return WaitError{}
		}
		return nil
	case <-f.ctx.Done():
		return f.ctx.Err()
	}
}

func (f *FakeCommand) String() string {
	return "FakeCommand"
}

func (f *FakeCommand) IsStartCalled() bool {
	return f.isStartCalled
}
func (f *FakeCommand) IsWaitCalled() bool {
	return f.isWaitCalled
}
