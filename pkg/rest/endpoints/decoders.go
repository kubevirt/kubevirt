package endpoints

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/asaskevich/govalidator"
	"github.com/emicklei/go-restful"
	"github.com/ghodss/yaml"
	gokithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"kubevirt.io/kubevirt/pkg/middleware"
	"net/http"
	"reflect"
	"strings"
)

type PutObject struct {
	Metadata Metadata
	Payload  interface{}
}

type Metadata struct {
	Name      string
	Namespace string
}

const (
	ReqKey  string = "restful_req__"
	RespKey string = "restful_resp__"
)

func GetRestfulRequest(ctx context.Context) *restful.Request {
	return ctx.Value(ReqKey).(*restful.Request)
}

func GetRestfulResponse(ctx context.Context) *restful.Response {
	return ctx.Value(RespKey).(*restful.Response)
}

func MakeGoRestfulWrapper(server *gokithttp.Server) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		requestFunc := func(ctx context.Context, _ *http.Request) context.Context {
			ctx = context.WithValue(ctx, ReqKey, request)
			ctx = context.WithValue(ctx, RespKey, response)
			return ctx
		}
		gokithttp.ServerBefore(requestFunc)(server)
		server.ServeHTTP(response.ResponseWriter, request.Request)
	}
}

func nameDecodeRequestFunc(ctx context.Context, r *http.Request) (interface{}, error) {
	rest := GetRestfulRequest(ctx)
	name := rest.PathParameter("name")
	if name == "" {
		return nil, errors.New("Could not find a 'name' variable.")
	}

	if !govalidator.IsAlphanumeric(name) {
		return nil, errors.New("Variable 'name' does not validate as alphanumeric.")
	}
	return name, nil
}

func namespaceDecodeRequestFunc(ctx context.Context, r *http.Request) (interface{}, error) {
	rest := GetRestfulRequest(ctx)

	namespace := rest.PathParameter("namespace")
	if namespace == "" {
		return nil, errors.New("Could not find a 'namespace' variable.")
	}

	if !govalidator.IsAlphanumeric(namespace) {
		return nil, errors.New("Variable 'name' does not validate as alphanumeric.")
	}
	return namespace, nil
}

func NameNamespaceDecodeRequestFunc(ctx context.Context, r *http.Request) (interface{}, error) {
	name, err := nameDecodeRequestFunc(ctx, r)
	if err != nil {
		return nil, err
	}
	namespace, err := namespaceDecodeRequestFunc(ctx, r)
	if err != nil {
		return nil, err
	}

	return &Metadata{Name: name.(string), Namespace: namespace.(string)}, nil
}

func NewJsonDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	payloadType := reflect.TypeOf(payloadTypePtr).Elem()
	return func(_ context.Context, r *http.Request) (interface{}, error) {
		obj := reflect.New(payloadType).Interface()
		if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
			return nil, err
		}
		if _, err := govalidator.ValidateStruct(obj); err != nil {
			return nil, err
		}
		return obj, nil
	}
}

func NewYamlDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	payloadType := reflect.TypeOf(payloadTypePtr).Elem()
	return func(_ context.Context, r *http.Request) (interface{}, error) {
		obj := reflect.New(payloadType).Interface()
		var b []byte
		buf := bytes.NewBuffer(b)
		_, err := buf.ReadFrom(r.Body)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(buf.Bytes(), obj); err != nil {
			return nil, err
		}
		if _, err := govalidator.ValidateStruct(obj); err != nil {
			return nil, err
		}
		return obj, nil
	}
}

func NewJsonPostDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := NewMimeTypeAwareDecodeRequestFunc(
		NewJsonDecodeRequestFunc(payloadTypePtr),
		map[string]gokithttp.DecodeRequestFunc{
			"application/json": NewJsonDecodeRequestFunc(payloadTypePtr),
			"application/yaml": NewYamlDecodeRequestFunc(payloadTypePtr),
		},
	)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		namespace, err := namespaceDecodeRequestFunc(ctx, r)
		if err != nil {
			return nil, err
		}
		payload, err := jsonDecodeRequestFunc(ctx, r)
		if err != nil {
			return nil, err
		}
		return &PutObject{Metadata: Metadata{Namespace: namespace.(string)}, Payload: payload}, nil
	}
}

func NewJsonPutDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := NewMimeTypeAwareDecodeRequestFunc(
		NewJsonDecodeRequestFunc(payloadTypePtr),
		map[string]gokithttp.DecodeRequestFunc{
			"application/json": NewJsonDecodeRequestFunc(payloadTypePtr),
			"application/yaml": NewYamlDecodeRequestFunc(payloadTypePtr),
		},
	)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		metadata, err := NameNamespaceDecodeRequestFunc(ctx, r)
		if err != nil {
			return nil, err
		}
		payload, err := jsonDecodeRequestFunc(ctx, r)
		if err != nil {
			return nil, err
		}
		return &PutObject{Metadata: *metadata.(*Metadata), Payload: payload}, nil
	}
}

func NewMimeTypeAwareDecodeRequestFunc(defaultDecoder gokithttp.DecodeRequestFunc, decoderMapping map[string]gokithttp.DecodeRequestFunc) gokithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		requestContext := GetRestfulRequest(ctx)
		contentType := strings.TrimSpace(requestContext.HeaderParameter("Content-Type"))
		// Use default encoder in case no Content-Type is specified
		decoder := defaultDecoder
		if len(contentType) > 0 {
			// Content-Type is given, check if we have a decoder and fail if we don't
			decoder = decoderMapping[contentType]
			if decoder == nil {
				return nil, middleware.NewUnsupportedMediaType(contentType)
			}
		}
		return decoder(ctx, r)
	}
}
