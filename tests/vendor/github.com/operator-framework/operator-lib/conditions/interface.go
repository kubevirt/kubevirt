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

package conditions

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition can Get and Set a conditionType in an Operator Condition custom resource
// associated with the operator.
type Condition interface {
	// Get fetches the condition on the operator's
	// OperatorCondition. It returns an error if there are problems getting
	// the OperatorCondition object or if the specific condition type does not
	// exist.
	Get(ctx context.Context) (*metav1.Condition, error)

	// Set sets the specific condition on the operator's
	// OperatorCondition to the provided status. If the condition is not
	// present, it is added to the CR.
	// To set a new condition, the user can call this method and provide optional
	// parameters if required. It returns an error if there are problems getting or
	// updating the OperatorCondition object.
	Set(ctx context.Context, status metav1.ConditionStatus, option ...Option) error
}

// Option is a function that applies a change to a condition.
// This can be used to set optional condition fields, like reasons
// and messages.
type Option func(*metav1.Condition)

// WithReason is an Option, which adds the reason
// to the condition.
func WithReason(reason string) Option {
	return func(c *metav1.Condition) {
		c.Reason = reason
	}
}

// WithMessage is an Option, which adds the reason
// to the condition.
func WithMessage(message string) Option {
	return func(c *metav1.Condition) {
		c.Message = message
	}
}
