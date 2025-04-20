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
 */

package util

import (
	"fmt"
	"os"
)

type EnvVarManager interface {
	Getenv(key string) string
	Setenv(key, value string) error
	Unsetenv(key string) error
	Environ() []string
}

type EnvVarManagerImpl struct{}

func (e EnvVarManagerImpl) Getenv(key string) string {
	return os.Getenv(key)
}

func (e EnvVarManagerImpl) Setenv(key, value string) error {
	return os.Setenv(key, value)
}

func (e EnvVarManagerImpl) Unsetenv(key string) error {
	return os.Unsetenv(key)
}

func (e EnvVarManagerImpl) Environ() []string {
	return os.Environ()
}

type EnvVarManagerMock struct {
	envVars map[string]string
}

func (e *EnvVarManagerMock) Getenv(key string) string {
	return e.envVars[key]
}

func (e *EnvVarManagerMock) Setenv(key, value string) error {
	if e.envVars == nil {
		e.envVars = make(map[string]string)
	}

	e.envVars[key] = value
	return nil
}

func (e *EnvVarManagerMock) Unsetenv(key string) error {
	delete(e.envVars, key)
	return nil
}

func (e *EnvVarManagerMock) Environ() (ret []string) {
	for key, value := range e.envVars {
		ret = append(ret, fmt.Sprintf("%s=%s", key, value))
	}
	return ret
}
