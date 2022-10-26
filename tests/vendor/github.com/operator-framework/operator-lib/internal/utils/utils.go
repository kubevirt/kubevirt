// Copyright 2020 The Operator-SDK Authors
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

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// ErrNoNamespace indicates that a namespace could not be found for the current
// environment
var ErrNoNamespace = fmt.Errorf("namespace not found for current environment")

var readSAFile = func() ([]byte, error) {
	return ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
}

// GetOperatorNamespace returns the namespace the operator should be running in from
// the associated service account secret.
var GetOperatorNamespace = func() (string, error) {
	nsBytes, err := readSAFile()
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoNamespace
		}
		return "", err
	}
	ns := strings.TrimSpace(string(nsBytes))
	return ns, nil
}
