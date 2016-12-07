package endpoints

import (
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"kubevirt.io/kubevirt/pkg/precond"
)

type HandlerBuilder interface {
	Build(context.Context) *kithttp.Server
	Post(interface{}) HandlerBuilder
	Put(interface{}) HandlerBuilder
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
	h.encoder = EncodePostResponse
	return h
}

func (h *handlerBuilder) Get() HandlerBuilder {
	h.decoder = NameNamespaceDecodeRequestFunc
	h.encoder = EncodeGetResponse
	return h
}

func (h *handlerBuilder) Delete() HandlerBuilder {
	h.decoder = NameNamespaceDecodeRequestFunc
	h.encoder = EncodeDeleteResponse
	return h
}

func (h *handlerBuilder) Put(payloadTypePtr interface{}) HandlerBuilder {
	h.decoder = NewJsonPutDecodeRequestFunc(payloadTypePtr)
	h.encoder = EncodePutResponse
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
