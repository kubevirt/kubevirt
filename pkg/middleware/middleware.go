package middleware

import (
	"errors"
	"fmt"
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
	"runtime/debug"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"kubevirt/core/pkg/precond"
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

// InternalErrorMiddleware is a convenience middleware which can be used in combination with panics.
// After data is sanitized and validated, services can expect to get reasonable valid data passed (e.g.
// object not nil, string not empty, ...). With this middleware in place the service can throw an exception with a
// precond.PreconditionError as payload. This middleware will catch that and translate it into an application
// level PreconditionError. All other detected panics will be converted into an InternalServerError. In both cases it
// is most likely that there is an error withing the application or a library. Long story short, this is about
// failing early in non recoverable situations.
func InternalErrorMiddleware(logger log.Logger) endpoint.Middleware {
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
					levels.New(logger).Crit().Log("stacktrace", debug.Stack())
				}
			}()
			data, err = next(ctx, request)
			return data, err
		}
	}
}
