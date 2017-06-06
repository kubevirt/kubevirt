/*
 * This file is part of the kubevirt project
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

package rest

import (
	"net/http"

	"github.com/emicklei/go-restful"
	gokithttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-manifest"
)

type manifest struct {
	connection virtwrap.Connection
}

// TODO: this should be a generalized decoder in rest/endpoints
func NewNonNamespacedJsonPostDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := endpoints.NewMimeTypeAwareDecodeRequestFunc(
		endpoints.NewJsonDecodeRequestFunc(payloadTypePtr),
		map[string]gokithttp.DecodeRequestFunc{
			rest.MIME_JSON: endpoints.NewJsonDecodeRequestFunc(payloadTypePtr),
			rest.MIME_YAML: endpoints.NewYamlDecodeRequestFunc(payloadTypePtr),
		},
	)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		payload, err := jsonDecodeRequestFunc(ctx, r)
		if err != nil {
			return nil, err
		}
		return &endpoints.PutObject{Payload: payload}, nil
	}
}

func ManifestService(connection virtwrap.Connection) (*restful.WebService, error) {
	ws := new(restful.WebService)

	m := manifest{
		connection: connection,
	}

	ctx := context.Background()
	aliveEndpoint := endpoints.MakeGoRestfulWrapper(endpoints.NewHandlerBuilder().
		Get().
		Decoder(endpoints.NoopDecoder).
		Endpoint(alive).
		Build(ctx))

	ctx = context.Background()
	mapManifestEndpoint := endpoints.MakeGoRestfulWrapper(endpoints.NewHandlerBuilder().
		Post(&v1.VM{}).
		Decoder(NewNonNamespacedJsonPostDecodeRequestFunc(&v1.VM{})).
		Endpoint(m.mapManifest).
		Build(ctx))

	ws.Path("/").
		Consumes(restful.MIME_JSON, rest.MIME_YAML).
		Produces(restful.MIME_JSON, rest.MIME_YAML)
	ws.Route(ws.GET("/api/v1/status").To(aliveEndpoint))
	ws.Route(ws.POST("/apis/" + v1.GroupVersion.String() + "/manifest").To(mapManifestEndpoint))
	return ws, nil
}

func alive(_ context.Context, request interface{}) (interface{}, error) {
	return map[string]interface{}{"status": "ok"}, nil
}

func (m manifest) mapManifest(_ context.Context, request interface{}) (interface{}, error) {
	vm := request.(*endpoints.PutObject).Payload.(*v1.VM)

	virt_manifest.AddMinimalVMSpec(vm)

	mappedVm, err := virt_manifest.MapVM(m.connection, vm)
	mappedVm.Spec.Domain.Name = mappedVm.ObjectMeta.Name
	if err != nil {
		return nil, middleware.NewInternalServerError(err)
	} else {
		return mappedVm, nil
	}
}
