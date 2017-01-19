package endpoints

import (
	"encoding/json"
	"golang.org/x/net/context"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/middleware"
	"net/http"
	"reflect"
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
