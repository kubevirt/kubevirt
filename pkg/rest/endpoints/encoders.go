package endpoints

import (
	"encoding/json"
	"golang.org/x/net/context"
	"kubevirt.io/core/pkg/middleware"
	"net/http"
)

func encodeApplicationErrors(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "text/plain")
	switch t := response.(type) {
	// More specific AppErrors  like 404 must be handled before the AppError case
	case middleware.ResourceNotFoundError:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(t.Cause().Error()))
	case middleware.ResourceExistsError:
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(t.Cause().Error()))
	case middleware.AppError:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(t.Cause().Error()))
	default:
		w.WriteHeader(http.StatusInternalServerError)
		// TODO log the error but don't send it along
		w.Write([]byte("Error handling failed, that should never happen."))
	}
	return json.NewEncoder(w).Encode(response)
}

func EncodePostResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	if _, ok := response.(middleware.AppError); ok != false {
		return encodeApplicationErrors(context, w, response)
	}
	return encodeJsonResponse(w, response, http.StatusCreated)
}

func EncodeGetResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	if _, ok := response.(middleware.AppError); ok != false {
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
