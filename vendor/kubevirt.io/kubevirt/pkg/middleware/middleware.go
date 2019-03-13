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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/go-kit/kit/endpoint"
	gklog "github.com/go-kit/kit/log"
	"golang.org/x/net/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
)

type AppError interface {
	Cause() error
}

type appError struct {
	err   error
	stack []byte
}

func (e *appError) Error() string {
	return e.err.Error()
}

func (e *appError) Cause() error {
	return e.err
}

// ValidationError indicates that a DTO validations failed. Can be used by http.EncodeResponseFunc implementations.
type ValidationError struct{ appError }

// MappingError indicates  that mapping a DTO to an entity failed. Can be used by endpoint.Endpoint
type MappingError struct{ appError }

// Resource which should be created exists already. Can be used by endpoint.Endpoint or a Service. E.g. lock
type ResourceExistsError struct{ appError }
type ResourceNotFoundError struct{ appError } // Can be thrown before or by a service call
type PreconditionError struct{ appError }     // Precondition not met, most likely a bug in a service (service)
type InternalServerError struct{ appError }   // Unknown internal error, most likely a bug in a service or a library
type UnsupportedMediaTypeError struct{ appError }
type BadRequestError struct{ appError }
type UnprocessableEntityError struct{ appError }

type KubernetesError struct {
	result rest.Result
}

func (k *KubernetesError) Cause() error {
	return k.result.Error()
}

func (k *KubernetesError) Error() string {
	return k.result.Error().Error()
}

func (k *KubernetesError) Status() (*v1.Status, error) {
	b, _ := k.result.Raw()
	status := v1.Status{}
	err := json.Unmarshal(b, &status)
	if err != nil {
		return &status, nil
	}
	return nil, err
}

func (k *KubernetesError) StatusCode() int {
	status, err := k.Status()
	if err != nil {
		return int(status.Code)
	} else {
		var s int
		k.result.StatusCode(&s)
		return s
	}
}

func (k *KubernetesError) Body() []byte {
	body, _ := k.result.Raw()
	return body
}

// InternalErrorMiddleware is a convenience middleware which can be used in combination with panics.
// After data is sanitized and validated, services can expect to get reasonable valid data passed (e.g.
// object not nil, string not empty, ...). With this middleware in place the service can throw an exception with a
// precond.PreconditionError as payload. This middleware will catch that and translate it into an application
// level PreconditionError. All other detected panics will be converted into an InternalServerError. In both cases it
// is most likely that there is an error within the application or a library. Long story short, this is about
// failing early in non recoverable situations.
func InternalErrorMiddleware(logger gklog.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (d interface{}, e error) {
			var data interface{}
			var err error
			defer func() {
				if r := recover(); r != nil {
					switch t := r.(type) {
					case *precond.PreconditionError:
						// Shortcut for failing precondition checks
						d, e = &PreconditionError{appError{err: t, stack: debug.Stack()}}, nil
					default:
						// Other panics should never happen, so map them to InternalServerError
						d = &InternalServerError{
							appError{
								err:   errors.New(fmt.Sprint(t)),
								stack: debug.Stack(),
							}}
						e = nil
					}
					// TODO log it with a logger at the right locations
					log.Log.Criticalf("stacktrace: %v", string(debug.Stack()))
				}
			}()
			data, err = next(ctx, request)
			// From here on all AppErrors returned through the err return value are treated as app
			// payload and returned with the right http return code
			// For instance a service can just return an AppError instance as normal err and this check
			// makes sure that our application error handler handles the response
			if _, ok := err.(AppError); ok {
				log.Log.Criticalf("%s", err)
				return err, nil
			}
			return data, err
		}
	}
}

func NewResourceNotFoundError(msg string) *ResourceNotFoundError {
	return &ResourceNotFoundError{appError{err: fmt.Errorf(msg)}}
}

func NewBadRequestError(msg string) *BadRequestError {
	return &BadRequestError{appError{err: fmt.Errorf(msg)}}
}

func NewResourceExistsError(resource string, name string) *ResourceNotFoundError {
	return NewResourceConflictError(fmt.Sprintf("%s with name %s already exists", resource, name))
}

func NewResourceConflictError(msg string) *ResourceNotFoundError {
	return &ResourceNotFoundError{appError{err: fmt.Errorf(msg)}}
}

func NewInternalServerError(err error) *InternalServerError {
	return &InternalServerError{appError{err: err}}
}

func NewKubernetesError(result rest.Result) *KubernetesError {
	return &KubernetesError{result: result}
}

func NewUnprocessibleEntityError(err error) *UnprocessableEntityError {
	return &UnprocessableEntityError{appError{err: err}}
}

func NewUnsupportedMediaType(mediaType string) *UnsupportedMediaTypeError {
	return &UnsupportedMediaTypeError{appError{err: fmt.Errorf("Media Type(s) '%s' not supported", mediaType)}}
}
