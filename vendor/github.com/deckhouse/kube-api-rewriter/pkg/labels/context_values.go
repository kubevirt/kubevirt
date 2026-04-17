/*
Copyright 2024 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package labels

import (
	"context"
	"strconv"
)

func ContextWithCommon(ctx context.Context, name, resource, method, watch, toTargetAction, fromTargetAction string) context.Context {
	ctx = context.WithValue(ctx, resourceKey{}, resource)
	ctx = context.WithValue(ctx, methodKey{}, method)
	ctx = context.WithValue(ctx, watchKey{}, watch)
	ctx = context.WithValue(ctx, toTargetActionKey{}, toTargetAction)
	ctx = context.WithValue(ctx, toTargetActionKey{}, fromTargetAction)
	return context.WithValue(ctx, nameKey{}, name)
}

func ContextWithDecision(ctx context.Context, decision string) context.Context {
	return context.WithValue(ctx, decisionKey{}, decision)
}

func ContextWithStatus(ctx context.Context, status int) context.Context {
	return context.WithValue(ctx, statusKey{}, strconv.Itoa(status))
}

type nameKey struct{}
type resourceKey struct{}
type methodKey struct{}
type watchKey struct{}
type decisionKey struct{}
type toTargetActionKey struct{}
type fromTargetActionKey struct{}
type statusKey struct{}

func NameFromContext(ctx context.Context) string {
	if method, ok := ctx.Value(nameKey{}).(string); ok {
		return method
	}
	return ""
}

func ResourceFromContext(ctx context.Context) string {
	if method, ok := ctx.Value(resourceKey{}).(string); ok {
		return method
	}
	return ""
}

func MethodFromContext(ctx context.Context) string {
	if method, ok := ctx.Value(methodKey{}).(string); ok {
		return method
	}
	return ""
}

func WatchFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(watchKey{}).(string); ok {
		return value
	}
	return ""
}

func ToTargetActionFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(toTargetActionKey{}).(string); ok {
		return value
	}
	return ""
}

func FromTargetActionFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(fromTargetActionKey{}).(string); ok {
		return value
	}
	return ""
}

func DecisionFromContext(ctx context.Context) string {
	if decision, ok := ctx.Value(decisionKey{}).(string); ok {
		return decision
	}
	return ""
}

func StatusFromContext(ctx context.Context) string {
	if decision, ok := ctx.Value(statusKey{}).(string); ok {
		return decision
	}
	return ""
}
