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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	restful "github.com/emicklei/go-restful"
	"github.com/ghodss/yaml"
	gokithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/rest"
)

type PutObject struct {
	Metadata Metadata
	Payload  interface{}
}

type PatchObject struct {
	Metadata  Metadata
	Patch     interface{}
	PatchType types.PatchType
}

type Metadata struct {
	Name      string
	Namespace string
	Headers   MetadataHeader
}

type MetadataHeader struct {
	Pretty          bool
	Export          bool
	Exact           bool
	LabelSelector   string
	FieldSelector   string
	ResourceVersion string
	TimeoutSeconds  int64
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

var nameIdentifierRegex = regexp.MustCompile("^[a-z0-9\\-]+$")
var namespaceIdentifierRegex = regexp.MustCompile("^[a-z0-9\\-]+$")

func nameDecodeRequestFunc(ctx context.Context, r *http.Request) (interface{}, error) {
	rest := GetRestfulRequest(ctx)
	name := rest.PathParameter("name")
	if name == "" {
		return nil, errors.New("Could not find a 'name' variable.")
	}

	if !nameIdentifierRegex.MatchString(name) {
		return nil, errors.New("Variable 'name' is invalid.")
	}
	return name, nil
}

func queryExtractor(ctx context.Context, r *http.Request) (*MetadataHeader, error) {
	rest := GetRestfulRequest(ctx)
	meta := &MetadataHeader{}

	if err := extractBool(rest.QueryParameter("pretty"), &(meta.Pretty)); err != nil {
		return nil, err
	}

	if err := extractBool(rest.QueryParameter("export"), &(meta.Export)); err != nil {
		return nil, err
	}
	if err := extractBool(rest.QueryParameter("exact"), &(meta.Exact)); err != nil {
		return nil, err
	}
	meta.FieldSelector = rest.QueryParameter("fieldSelector")
	meta.LabelSelector = rest.QueryParameter("labelSelector")
	meta.ResourceVersion = rest.QueryParameter("resourceVersion")

	if err := extractDuration(rest.QueryParameter("timeoutSeconds"), &(meta.TimeoutSeconds)); err != nil {
		return nil, err
	}
	return meta, nil
}

func extractBool(header string, target *bool) error {
	if header != "" {
		f, err := strconv.ParseBool(header)
		if err != nil {
			return err
		}
		target = &f
	}
	return nil
}

func extractDuration(header string, target *int64) error {
	if header != "" {
		f, err := strconv.Atoi(header)
		if err != nil {
			return err
		}
		f64 := int64(f)
		target = &f64
	}
	return nil
}

func namespaceDecodeRequestFunc(ctx context.Context, r *http.Request) (interface{}, error) {
	rest := GetRestfulRequest(ctx)

	namespace := rest.PathParameter("namespace")
	if namespace == "" {
		return nil, errors.New("Could not find a 'namespace' variable.")
	}

	if !namespaceIdentifierRegex.MatchString(namespace) {
		return nil, errors.New("Variable 'namespace' is invalid.")
	}
	return namespace, nil
}

func NoopDecoder(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func NotNamespacedDecodeRequestFunc(ctx context.Context, r *http.Request) (interface{}, error) {
	headers, err := queryExtractor(ctx, r)
	if err != nil {
		return nil, err
	}
	return &Metadata{Headers: *headers}, nil
}

func NamespaceDecodeRequestFunc(ctx context.Context, r *http.Request) (interface{}, error) {
	namespace, err := namespaceDecodeRequestFunc(ctx, r)
	if err != nil {
		return nil, err
	}
	headers, err := queryExtractor(ctx, r)
	if err != nil {
		return nil, err
	}
	return &Metadata{Namespace: namespace.(string), Headers: *headers}, nil
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
	headers, err := queryExtractor(ctx, r)
	if err != nil {
		return nil, err
	}
	return &Metadata{Name: name.(string), Namespace: namespace.(string), Headers: *headers}, nil
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

func JsonPatchDecodeRequestFunc(_ context.Context, r *http.Request) (interface{}, error) {
	var patch interface{}
	err := json.NewDecoder(r.Body).Decode(&patch)
	if err != nil {
		return nil, err
	}
	return patch, nil
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

func NewJsonDeleteDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := NewMimeTypeAwareDecodeRequestFunc(
		NewJsonDecodeRequestFunc(payloadTypePtr),
		map[string]gokithttp.DecodeRequestFunc{
			rest.MIME_JSON: NewJsonDecodeRequestFunc(payloadTypePtr),
			rest.MIME_YAML: NewYamlDecodeRequestFunc(payloadTypePtr),
		},
	)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		metadata, err := NameNamespaceDecodeRequestFunc(ctx, r)
		if err != nil {
			return nil, err
		}
		var payload interface{}

		if r.Body != nil {
			payload, err = jsonDecodeRequestFunc(ctx, r)
			// payload is optional, so continue if we get EOF
			if err != nil && err != io.EOF {
				return nil, err
			}
		}
		return &PutObject{Metadata: *metadata.(*Metadata), Payload: payload}, nil
	}
}

func NewJsonPostDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := NewMimeTypeAwareDecodeRequestFunc(
		NewJsonDecodeRequestFunc(payloadTypePtr),
		map[string]gokithttp.DecodeRequestFunc{
			rest.MIME_JSON: NewJsonDecodeRequestFunc(payloadTypePtr),
			rest.MIME_YAML: NewYamlDecodeRequestFunc(payloadTypePtr),
		},
	)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		metadata, err := NamespaceDecodeRequestFunc(ctx, r)
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

func NewJsonPutDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := NewMimeTypeAwareDecodeRequestFunc(
		NewJsonDecodeRequestFunc(payloadTypePtr),
		map[string]gokithttp.DecodeRequestFunc{
			rest.MIME_JSON: NewJsonDecodeRequestFunc(payloadTypePtr),
			rest.MIME_YAML: NewYamlDecodeRequestFunc(payloadTypePtr),
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

func NewJsonPatchDecodeRequestFunc() gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := NewMimeTypeAwareDecodeRequestFunc(
		nil,
		map[string]gokithttp.DecodeRequestFunc{
			string(types.JSONPatchType):  JsonPatchDecodeRequestFunc,
			string(types.MergePatchType): JsonPatchDecodeRequestFunc,
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
		patchType := types.PatchType(r.Header.Get("Content-Type"))
		return &PatchObject{Metadata: *metadata.(*Metadata), Patch: payload, PatchType: patchType}, nil
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
