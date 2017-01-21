package endpoints

import (
	"encoding/json"
	kithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"gopkg.in/ini.v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/middleware"
	"net/http"
	"reflect"
	"strings"
)

func encodeApplicationErrors(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "text/plain")
	var err error
	switch t := response.(type) {
	// More specific AppErrors  like 404 must be handled before the AppError case
	case *middleware.KubernetesError:
		w.WriteHeader(t.StatusCode())
		_, err = w.Write(t.Body())
	case *middleware.ResourceNotFoundError:
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte(t.Cause().Error()))
	case *middleware.UnsupportedMediaTypeError:
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_, err = w.Write([]byte(t.Cause().Error()))
	case middleware.BadRequestError:
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte(t.Cause().Error()))
	case middleware.ResourceExistsError:
		w.WriteHeader(http.StatusConflict)
		_, err = w.Write([]byte(t.Cause().Error()))
	case middleware.AppError:
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(t.Cause().Error()))
	default:
		w.WriteHeader(http.StatusInternalServerError)
		// TODO log the error but don't send it along
		_, err = w.Write([]byte("Error handling failed, that should never happen."))
	}
	return err
}

func EncodePostResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	logging.DefaultLogger().Info().Msg(reflect.TypeOf(response).Name())
	if _, ok := response.(middleware.AppError); ok {
		return encodeApplicationErrors(context, w, response)
	}
	return encodeJsonResponse(w, response, http.StatusCreated)
}

func EncodeGetResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	if _, ok := response.(middleware.AppError); ok {
		return encodeApplicationErrors(context, w, response)
	}
	return encodeJsonResponse(w, response, http.StatusOK)
}

func EncodePlainTextGetResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	if _, ok := response.(middleware.AppError); ok != false {
		return encodeApplicationErrors(context, w, response)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(response.(string)))
	return err
}

func EncodeINIGetResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	if _, ok := response.(middleware.AppError); ok != false {
		return encodeApplicationErrors(context, w, response)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	cfg := ini.Empty()
	err := ini.ReflectFrom(cfg, response)
	if err != nil {
		return err
	}
	_, err = cfg.WriteTo(w)
	return err
}

func EncodeDeleteResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeGetResponse(context, w, response)
}

func EncodePutResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeGetResponse(context, w, response)
}

func encodeJsonResponse(w http.ResponseWriter, response interface{}, returnCode int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(returnCode)
	return json.NewEncoder(w).Encode(response)
}

func NewMimeTypeAwareEncoder(defaultEncoder kithttp.EncodeResponseFunc, encoderMapping map[string]kithttp.EncodeResponseFunc) kithttp.EncodeResponseFunc {
	return func(context context.Context, w http.ResponseWriter, response interface{}) error {
		requestContext := GetRestfulRequest(context)
		contentTypes := strings.TrimSpace(requestContext.HeaderParameter("Accept"))

		var encoder kithttp.EncodeResponseFunc
		if len(contentTypes) == 0 {
			encoder = defaultEncoder
		} else {
			for _, m := range strings.Split(contentTypes, ",") {
				encoder = encoderMapping[strings.TrimSpace(m)]
				if encoder != nil {
					break
				}
			}
			if encoder == nil {
				return encodeApplicationErrors(context, w, middleware.NewUnsupportedMediaType(contentTypes))
			}
		}

		return encoder(context, w, response)
	}
}
