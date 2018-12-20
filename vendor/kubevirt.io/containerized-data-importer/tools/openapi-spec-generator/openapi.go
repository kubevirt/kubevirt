package main

import (
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
)

// code stolen/adapted from https://github.com/kubevirt/kubevirt/blob/master/pkg/util/openapi/openapi.go

func createOpenAPIConfig(webServices []*restful.WebService) restfulspec.Config {
	return restfulspec.Config{
		WebServices:    webServices,
		WebServicesURL: "",
		APIPath:        "/swaggerapi",
		PostBuildSwaggerObjectHandler: addInfoToSwaggerObject,
	}
}

func addInfoToSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "KubeVirt Containerized Data Importer API",
			Description: "Containerized Data Importer for KubeVirt.",
			Contact: &spec.ContactInfo{
				Name:  "kubevirt-dev",
				Email: "kubevirt-dev@googlegroups.com",
				URL:   "https://github.com/kubevirt/containerized-data-importer",
			},
			License: &spec.License{
				Name: "Apache 2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0",
			},
		},
	}
	swo.SecurityDefinitions = spec.SecurityDefinitions{
		"BearerToken": &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type:        "apiKey",
				Name:        "authorization",
				In:          "header",
				Description: "Bearer Token authentication",
			},
		},
	}
	swo.Security = make([]map[string][]string, 1)
	swo.Security[0] = map[string][]string{"BearerToken": {}}
}

func loadOpenAPISpec(webServices []*restful.WebService) *spec.Swagger {
	openapispec := restfulspec.BuildSwagger(createOpenAPIConfig(webServices))

	// creationTimestamp, lastProbeTime and lastTransitionTime are deserialized as "null"
	// Fix it here until
	// https://github.com/kubernetes/kubernetes/issues/66899 is ready
	// Otherwise CRDs can't use templates which contain metadata and controllers
	// can't set conditions without timestamps
	objectMeta := openapispec.Definitions["v1.ObjectMeta"]
	prop := objectMeta.Properties["creationTimestamp"]
	prop.Type = spec.StringOrArray{"string", "null"}
	objectMeta.Properties["creationTimestamp"] = prop

	for k, s := range openapispec.Definitions {
		if strings.HasSuffix(k, "Condition") {
			prop := s.Properties["lastProbeTime"]
			prop.Type = spec.StringOrArray{"string", "null"}
			s.Properties["lastProbeTime"] = prop
			prop = s.Properties["lastTransitionTime"]
			prop.Type = spec.StringOrArray{"string", "null"}
			s.Properties["lastTransitionTime"] = prop
		}
	}

	return openapispec
}
