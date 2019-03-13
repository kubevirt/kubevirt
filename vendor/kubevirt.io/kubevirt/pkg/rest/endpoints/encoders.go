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

package endpoints

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ghodss/yaml"
	kithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	ini "gopkg.in/ini.v1"

	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/rest"
)

func encodeApplicationErrors(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", rest.MIME_TEXT)
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
	case *middleware.UnprocessableEntityError:
		w.WriteHeader(422)
		_, err = w.Write([]byte(t.Cause().Error()))
	case *middleware.BadRequestError:
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte(t.Cause().Error()))
	case *middleware.ResourceExistsError:
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

func NewEncodeJsonResponse(returnCode int) kithttp.EncodeResponseFunc {
	return func(context context.Context, w http.ResponseWriter, response interface{}) error {
		if _, ok := response.(middleware.AppError); ok {
			return encodeApplicationErrors(context, w, response)
		}
		return encodeJsonResponse(w, response, returnCode)
	}
}

func NewEncodeINIResponse(returnCode int) kithttp.EncodeResponseFunc {
	return func(context context.Context, w http.ResponseWriter, response interface{}) error {
		if _, ok := response.(middleware.AppError); ok != false {
			return encodeApplicationErrors(context, w, response)
		}
		return encodeINIResponse(w, response, returnCode)
	}
}

func NewEncodeYamlResponse(returnCode int) kithttp.EncodeResponseFunc {
	return func(context context.Context, w http.ResponseWriter, response interface{}) error {
		if _, ok := response.(middleware.AppError); ok != false {
			return encodeApplicationErrors(context, w, response)
		}
		return encodeYamlResponse(w, response, returnCode)
	}
}

func encodeJsonResponse(w http.ResponseWriter, response interface{}, returnCode int) error {
	w.Header().Set("Content-Type", rest.MIME_JSON)
	w.WriteHeader(returnCode)
	return json.NewEncoder(w).Encode(response)
}

func encodeYamlResponse(w http.ResponseWriter, response interface{}, returnCode int) error {
	w.Header().Set("Content-Type", rest.MIME_YAML)
	w.WriteHeader(returnCode)
	b, err := yaml.Marshal(response)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func encodeINIResponse(w http.ResponseWriter, response interface{}, returnCode int) error {
	w.Header().Set("Content-Type", rest.MIME_INI)
	w.WriteHeader(returnCode)
	cfg := ini.Empty()
	err := ini.ReflectFrom(cfg, response)
	if err != nil {
		return err
	}
	_, err = cfg.WriteTo(w)
	return err
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
				mimeType := strings.TrimSpace(m)
				// go-restful adds the content type "*/*" if none was given, if we see that one, use the default encoder
				if mimeType == "*/*" {
					encoder = defaultEncoder
				} else {
					encoder = encoderMapping[mimeType]
				}
				if encoder != nil {
					break
				}
			}
		}
		if encoder == nil {
			return encodeApplicationErrors(context, w, middleware.NewUnsupportedMediaType(contentTypes))
		}

		return encoder(context, w, response)
	}
}
