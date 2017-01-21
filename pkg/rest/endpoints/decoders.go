package endpoints

import (
	"encoding/json"
	"errors"
	"github.com/asaskevich/govalidator"
	"github.com/emicklei/go-restful"
	gokithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"net/http"
	"reflect"
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

func NewJsonPostDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := NewJsonDecodeRequestFunc(payloadTypePtr)
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
	jsonDecodeRequestFunc := NewJsonDecodeRequestFunc(payloadTypePtr)
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
