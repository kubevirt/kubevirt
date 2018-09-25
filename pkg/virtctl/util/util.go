package util

// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package flags implements command-line flag parsing.

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagToEnv converts flag string to upper-case environment variable key string.
func FlagToEnv(prefix, name string) string {
	return prefix + "_" + strings.ToUpper(strings.Replace(name, "-", "_", -1))
}

func LoadEnvVariables(cmd *cobra.Command) {
	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		env := FlagToEnv("KUBECTL_PLUGINS_LOCAL_FLAG", flag.Name)
		if value, ok := os.LookupEnv(env); ok {
			flag.Value.Set(value)
		}
	})
}
