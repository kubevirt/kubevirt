// Copyright 2021 The Operator-SDK Authors
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

package conditions

import (
	"fmt"
	"os"

	apiv2 "github.com/operator-framework/api/pkg/operators/v2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/operator-lib/internal/utils"
)

// Factory define the interface for building Conditions.
type Factory interface {
	NewCondition(apiv2.ConditionType) (Condition, error)
	GetNamespacedName() (*types.NamespacedName, error)
}

// InClusterFactory is a conditions factory that can build conditions and get
// the namespaced name of the operator's condition based on an in-cluster
// configuration.
type InClusterFactory struct {
	Client client.Client
}

// NewCondition creates a new Condition using the provided client and condition
// type. The condition's name and namespace are determined by the Factory's GetName
// and GetNamespace functions.
func (f InClusterFactory) NewCondition(condType apiv2.ConditionType) (Condition, error) {
	objKey, err := f.GetNamespacedName()
	if err != nil {
		return nil, err
	}
	return &condition{
		namespacedName: *objKey,
		condType:       condType,
		client:         f.Client,
	}, nil
}

// GetNamespacedName returns the NamespacedName of the CR. It returns an error
// when the name of the CR cannot be found from the environment variable set by
// OLM. Hence, GetNamespacedName() can provide the NamespacedName when the operator
// is running on cluster and is being managed by OLM.
func (f InClusterFactory) GetNamespacedName() (*types.NamespacedName, error) {
	conditionName, err := f.getConditionName()
	if err != nil {
		return nil, fmt.Errorf("get operator condition name: %v", err)
	}
	conditionNamespace, err := f.getConditionNamespace()
	if err != nil {
		return nil, fmt.Errorf("get operator condition namespace: %v", err)
	}

	return &types.NamespacedName{Name: conditionName, Namespace: conditionNamespace}, nil
}

const (
	// operatorCondEnvVar is the env variable which
	// contains the name of the Condition CR associated to the operator,
	// set by OLM.
	operatorCondEnvVar = "OPERATOR_CONDITION_NAME"
)

// getConditionName reads and returns the OPERATOR_CONDITION_NAME environment
// variable. If the variable is unset or empty, it returns an error.
func (f InClusterFactory) getConditionName() (string, error) {
	name := os.Getenv(operatorCondEnvVar)
	if name == "" {
		return "", fmt.Errorf("could not determine operator condition name: environment variable %s not set", operatorCondEnvVar)
	}
	return name, nil
}

// readNamespace gets the namespacedName of the operator.
var readNamespace = utils.GetOperatorNamespace

// getConditionNamespace reads the namespace file mounted into a pod in a
// cluster via its service account volume. If the file is not found or cannot be
// read, this function returns an error.
func (f InClusterFactory) getConditionNamespace() (string, error) {
	return readNamespace()
}

// NewCondition returns a new Condition interface using the provided client
// for the specified conditionType. The condition will internally fetch the namespacedName
// of the operatorConditionCRD.
//
// Deprecated: Use InClusterFactory{cl}.NewCondition() instead.
func NewCondition(cl client.Client, condType apiv2.ConditionType) (Condition, error) {
	return InClusterFactory{cl}.NewCondition(condType)
}

// GetNamespacedName returns the NamespacedName of the CR. It returns an error
// when the name of the CR cannot be found from the environment variable set by
// OLM. Hence, GetNamespacedName() can provide the NamespacedName when the operator
// is running on cluster and is being managed by OLM. If running locally, operator
// writers are encouraged to skip this method or gracefully handle the errors by logging
// a message.
//
// Deprecated: InClusterFactory{}.GetNamespacedName().
func GetNamespacedName() (*types.NamespacedName, error) {
	return InClusterFactory{}.GetNamespacedName()
}
