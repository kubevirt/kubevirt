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
