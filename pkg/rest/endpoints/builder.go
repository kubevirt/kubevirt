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
	"net/http"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/rest"
)

type HandlerBuilder interface {
	Build(context.Context) *kithttp.Server
	Post(interface{}) HandlerBuilder
	Put(interface{}) HandlerBuilder
	Patch() HandlerBuilder
	Get() HandlerBuilder
	Delete() HandlerBuilder
	Middleware([]endpoint.Middleware) HandlerBuilder
	Encoder(kithttp.EncodeResponseFunc) HandlerBuilder
	Decoder(kithttp.DecodeRequestFunc) HandlerBuilder
	Endpoint(endpoint.Endpoint) HandlerBuilder
}

type handlerBuilder struct {
	middleware []endpoint.Middleware
	endpoint   endpoint.Endpoint
	encoder    kithttp.EncodeResponseFunc
	decoder    kithttp.DecodeRequestFunc
}

func (h *handlerBuilder) Build(ctx context.Context) *kithttp.Server {
	precond.MustNotBeNil(h.endpoint)
	precond.MustNotBeNil(h.encoder)
	precond.MustNotBeNil(h.decoder)

	// wrap endpoint with registered middleware
	endpoint := h.endpoint
	for _, mw := range h.middleware {
		endpoint = mw(endpoint)
	}

	return kithttp.NewServer(
		ctx,
		endpoint,
		h.decoder,
		h.encoder,
	)
}

func (h *handlerBuilder) Post(payloadTypePtr interface{}) HandlerBuilder {
	h.decoder = NewJsonPostDecodeRequestFunc(payloadTypePtr)
	h.encoder = NewMimeTypeAwareEncoder(NewEncodeJsonResponse(http.StatusCreated),
		map[string]kithttp.EncodeResponseFunc{
			rest.MIME_JSON: NewEncodeJsonResponse(http.StatusCreated),
			rest.MIME_YAML: NewEncodeYamlResponse(http.StatusCreated),
		},
	)
	return h
}

func (h *handlerBuilder) Get() HandlerBuilder {
	h.decoder = NameNamespaceDecodeRequestFunc
	h.encoder = NewMimeTypeAwareEncoder(NewEncodeJsonResponse(http.StatusOK),
		map[string]kithttp.EncodeResponseFunc{
			rest.MIME_JSON: NewEncodeJsonResponse(http.StatusOK),
			rest.MIME_YAML: NewEncodeYamlResponse(http.StatusOK),
		},
	)
	return h
}

func (h *handlerBuilder) Delete() HandlerBuilder {
	h.decoder = NewJsonDeleteDecodeRequestFunc(&v1.DeleteOptions{})
	h.encoder = NewMimeTypeAwareEncoder(NewEncodeJsonResponse(http.StatusOK),
		map[string]kithttp.EncodeResponseFunc{
			rest.MIME_JSON: NewEncodeJsonResponse(http.StatusOK),
			rest.MIME_YAML: NewEncodeYamlResponse(http.StatusOK),
		},
	)
	return h
}

func (h *handlerBuilder) Put(payloadTypePtr interface{}) HandlerBuilder {
	h.decoder = NewJsonPutDecodeRequestFunc(payloadTypePtr)
	h.encoder = NewMimeTypeAwareEncoder(NewEncodeJsonResponse(http.StatusOK),
		map[string]kithttp.EncodeResponseFunc{
			rest.MIME_JSON: NewEncodeJsonResponse(http.StatusOK),
			rest.MIME_YAML: NewEncodeYamlResponse(http.StatusOK),
		},
	)
	return h
}

func (h *handlerBuilder) Patch() HandlerBuilder {
	h.decoder = NewJsonPatchDecodeRequestFunc()
	h.encoder = NewMimeTypeAwareEncoder(NewEncodeJsonResponse(http.StatusOK),
		map[string]kithttp.EncodeResponseFunc{
			rest.MIME_JSON: NewEncodeJsonResponse(http.StatusOK),
			rest.MIME_YAML: NewEncodeYamlResponse(http.StatusOK),
		},
	)
	return h
}

func (h *handlerBuilder) Middleware(middleware []endpoint.Middleware) HandlerBuilder {
	h.middleware = middleware
	return h
}

func (h *handlerBuilder) Decoder(decoder kithttp.DecodeRequestFunc) HandlerBuilder {
	h.decoder = decoder
	return h
}

func (h *handlerBuilder) Encoder(encoder kithttp.EncodeResponseFunc) HandlerBuilder {
	h.encoder = encoder
	return h
}

func NewHandlerBuilder() HandlerBuilder {
	return &handlerBuilder{}
}

func (h *handlerBuilder) Endpoint(endpoint endpoint.Endpoint) HandlerBuilder {
	h.endpoint = endpoint
	return h
}
