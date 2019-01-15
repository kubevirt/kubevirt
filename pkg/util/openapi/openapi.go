package openapi

import (
	"encoding/json"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Validator struct {
	specSchemes   *spec.Schema
	statusSchemes *spec.Schema
	topLevelKeys  map[string]interface{}
}

func CreateOpenAPIConfig(webServices []*restful.WebService) restfulspec.Config {
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
			Title:       "KubeVirt API",
			Description: "This is KubeVirt API an add-on for Kubernetes.",
			Contact: &spec.ContactInfo{
				Name:  "kubevirt-dev",
				Email: "kubevirt-dev@googlegroups.com",
				URL:   "https://github.com/kubevirt/kubevirt",
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
	swo.Swagger = "2.0"
	swo.Security = make([]map[string][]string, 1)
	swo.Security[0] = map[string][]string{"BearerToken": {}}
}

func LoadOpenAPISpec(webServices []*restful.WebService) *spec.Swagger {
	openapispec := restfulspec.BuildSwagger(CreateOpenAPIConfig(webServices))

	// creationTimestamp, lastProbeTime and lastTransitionTime are deserialized as "null"
	// Fix it here until
	// https://github.com/kubernetes/kubernetes/issues/66899 is ready
	// Otherwise CRDs can't use templates which contain metadata and controllers
	// can't set conditions without timestamps
	objectMeta, exists := openapispec.Definitions["v1.ObjectMeta"]
	if exists {
		prop := objectMeta.Properties["creationTimestamp"]
		prop.Type = spec.StringOrArray{"string", "null"}
		objectMeta.Properties["creationTimestamp"] = prop
	}

	for k, s := range openapispec.Definitions {
		if strings.HasSuffix(k, "Condition") {
			prop := s.Properties["lastProbeTime"]
			prop.Type = spec.StringOrArray{"string", "null"}
			s.Properties["lastProbeTime"] = prop
			prop = s.Properties["lastTransitionTime"]
			prop.Type = spec.StringOrArray{"string", "null"}
			s.Properties["lastTransitionTime"] = prop
		}
		if k == "v1.HTTPGetAction" {
			prop := s.Properties["port"]
			prop.Type = spec.StringOrArray{"string", "number"}
			s.Properties["port"] = prop
		}
		if k == "v1.TCPSocketAction" {
			prop := s.Properties["port"]
			prop.Type = spec.StringOrArray{"string", "number"}
			s.Properties["port"] = prop
		}
	}

	return openapispec
}

func CreateOpenAPIValidator(webServices []*restful.WebService) *Validator {
	openapispec := LoadOpenAPISpec(webServices)
	data, err := json.Marshal(openapispec)
	if err != nil {
		glog.Fatal(err)
	}

	specSchema := &spec.Schema{}
	err = json.Unmarshal(data, specSchema)
	if err != nil {
		panic(err)
	}

	// Make sure that no unknown fields are allowed in specs
	for k, v := range specSchema.Definitions {
		v.AdditionalProperties = &spec.SchemaOrBool{Allows: false}
		v.AdditionalItems = &spec.SchemaOrBool{Allows: false}
		specSchema.Definitions[k] = v
	}

	// Expand the specSchemes
	err = spec.ExpandSchema(specSchema, specSchema, nil)
	if err != nil {
		glog.Fatal(err)
	}

	// Load spec once again for status. The status should accept unknown fields
	statusSchema := &spec.Schema{}
	err = json.Unmarshal(data, statusSchema)
	if err != nil {
		panic(err)
	}
	// Expand the statusSchemes
	err = spec.ExpandSchema(statusSchema, statusSchema, nil)
	if err != nil {
		glog.Fatal(err)
	}

	return &Validator{
		specSchemes:   specSchema,
		statusSchemes: statusSchema,
		topLevelKeys: map[string]interface{}{
			"kind":       nil,
			"apiVersion": nil,
			"spec":       nil,
			"status":     nil,
			"metadata":   nil,
		},
	}
}

func (v *Validator) Validate(gvk schema.GroupVersionKind, obj map[string]interface{}) []error {
	errs := []error{}
	for k, _ := range obj {
		if _, exists := v.topLevelKeys[k]; !exists {
			errs = append(errs, errors.PropertyNotAllowed("", "body", k))
		}
	}

	if _, exists := obj["spec"]; !exists {
		errs = append(errs, errors.Required("spec", "body"))
	}

	errs = append(errs, v.ValidateSpec(gvk, obj)...)
	errs = append(errs, v.ValidateStatus(gvk, obj)...)
	return errs
}

func (v *Validator) ValidateSpec(gvk schema.GroupVersionKind, obj map[string]interface{}) []error {
	schema := v.specSchemes.Definitions["v1."+gvk.Kind+"Spec"]
	result := validate.NewSchemaValidator(&schema, nil, "spec", strfmt.Default).Validate(obj["spec"])
	return result.Errors
}

func (v *Validator) ValidateStatus(gvk schema.GroupVersionKind, obj map[string]interface{}) []error {
	schema := v.statusSchemes.Definitions["v1."+gvk.Kind+"Status"]
	result := validate.NewSchemaValidator(&schema, nil, "status", strfmt.Default).Validate(obj["status"])
	return result.Errors
}
