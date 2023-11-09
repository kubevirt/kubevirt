package openapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/emicklei/go-restful/v3"
	openapi_spec "github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	openapi_validate "github.com/go-openapi/validate"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/errors"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"kubevirt.io/client-go/api"
)

type Validator struct {
	specSchemes   *openapi_spec.Schema
	statusSchemes *openapi_spec.Schema
	topLevelKeys  map[string]interface{}
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

func CreateConfig() *common.Config {
	return &common.Config{
		CommonResponses: map[int]spec.Response{
			401: {
				ResponseProps: spec.ResponseProps{
					Description: "Unauthorized",
				},
			},
		},
		Info: &spec.Info{
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
		},
		SecurityDefinitions: &spec.SecurityDefinitions{
			"BearerToken": &spec.SecurityScheme{
				SecuritySchemeProps: spec.SecuritySchemeProps{
					Type:        "apiKey",
					Name:        "authorization",
					In:          "header",
					Description: "Bearer Token authentication",
				},
			},
		},
		GetDefinitions: func(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
			m := api.GetOpenAPIDefinitions(ref)
			for k, v := range m {
				if _, ok := m[k]; !ok {
					m[k] = v
				}
			}
			return m
		},

		GetDefinitionName: func(name string) (string, spec.Extensions) {
			if strings.Contains(name, "kubevirt.io") {
				// keeping for validation
				return name[strings.LastIndex(name, "/")+1:], nil
			}
			//adpting k8s style
			return strings.ReplaceAll(name, "/", "."), nil
		},
	}
}

func LoadOpenAPISpec(webServices []*restful.WebService) *spec.Swagger {
	config := CreateConfig()
	openapispec, err := builder.BuildOpenAPISpec(webServices, config)
	if err != nil {
		panic(fmt.Errorf("Failed to build swagger: %s", err))
	}

	// creationTimestamp, lastProbeTime and lastTransitionTime are deserialized as "null"
	// Fix it here until
	// https://github.com/kubernetes/kubernetes/issues/66899 is ready
	// Otherwise CRDs can't use templates which contain metadata and controllers
	// can't set conditions without timestamps

	objectmeta := ""
	for k := range openapispec.Definitions {
		if strings.Contains(k, "v1.ObjectMeta") {
			objectmeta = k
			break
		}
	}
	resourceRequirements, exists := openapispec.Definitions["v1.ResourceRequirements"]
	if exists {
		limits, exists := resourceRequirements.Properties["limits"]
		if exists {
			limits.AdditionalProperties = nil
			resourceRequirements.Properties["limits"] = limits
		}
		requests, exists := resourceRequirements.Properties["requests"]
		if exists {
			requests.AdditionalProperties = nil
			resourceRequirements.Properties["requests"] = requests
		}

	}

	objectMeta, exists := openapispec.Definitions[objectmeta]
	if exists {
		prop := objectMeta.Properties["creationTimestamp"]
		prop.Type = spec.StringOrArray{"string", "null"}
		// mask v1.Time as in validation v1.Time override sting,null type
		prop.Ref = spec.Ref{}
		objectMeta.Properties["creationTimestamp"] = prop
	}

	for k, s := range openapispec.Definitions {
		// allow nullable statuses
		if status, found := s.Properties["status"]; found {
			if !status.Type.Contains("string") {
				definitionName := strings.Split(status.Ref.GetPointer().String(), "/")[2]
				object := openapispec.Definitions[definitionName]
				object.Nullable = true
				openapispec.Definitions[definitionName] = object
			}
		}

		if strings.HasSuffix(k, "Condition") {
			prop := s.Properties["lastProbeTime"]
			prop.Type = spec.StringOrArray{"string", "null"}
			prop.Ref = spec.Ref{}
			s.Properties["lastProbeTime"] = prop

			prop = s.Properties["lastTransitionTime"]
			prop.Type = spec.StringOrArray{"string", "null"}
			prop.Ref = spec.Ref{}
			s.Properties["lastTransitionTime"] = prop
		}
		if strings.Contains(k, "v1.HTTPGetAction") {
			prop := s.Properties["port"]
			prop.Type = spec.StringOrArray{"string", "number"}
			// As intstr.IntOrString, the ref for that must be masked
			prop.Ref = spec.Ref{}
			s.Properties["port"] = prop
		}
		if strings.Contains(k, "v1.TCPSocketAction") {
			prop := s.Properties["port"]
			prop.Type = spec.StringOrArray{"string", "number"}
			// As intstr.IntOrString, the ref for that must be masked
			prop.Ref = spec.Ref{}
			s.Properties["port"] = prop
		}
		if strings.Contains(k, "v1.PersistentVolumeClaimSpec") {
			for i, r := range s.Required {
				if r == "dataSource" {
					s.Required = append(s.Required[:i], s.Required[i+1:]...)
					openapispec.Definitions[k] = s
					break
				}
			}
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

	specSchema := &openapi_spec.Schema{}
	err = json.Unmarshal(data, specSchema)
	if err != nil {
		panic(err)
	}

	// Make sure that no unknown fields are allowed in specs
	for k, v := range specSchema.Definitions {
		v.AdditionalProperties = &openapi_spec.SchemaOrBool{Allows: false}
		v.AdditionalItems = &openapi_spec.SchemaOrBool{Allows: false}
		specSchema.Definitions[k] = v
	}

	// Expand the specSchemes
	err = openapi_spec.ExpandSchema(specSchema, specSchema, nil)
	if err != nil {
		glog.Fatal(err)
	}

	// Load spec once again for status. The status should accept unknown fields
	statusSchema := &openapi_spec.Schema{}
	err = json.Unmarshal(data, statusSchema)
	if err != nil {
		panic(err)
	}

	// Expand the statusSchemes
	err = openapi_spec.ExpandSchema(statusSchema, statusSchema, nil)
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
	for k := range obj {
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
	result := openapi_validate.NewSchemaValidator(&schema, nil, "spec", strfmt.Default).Validate(obj["spec"])
	return result.Errors
}

func (v *Validator) ValidateStatus(gvk schema.GroupVersionKind, obj map[string]interface{}) []error {
	schema := v.statusSchemes.Definitions["v1."+gvk.Kind+"Status"]
	result := openapi_validate.NewSchemaValidator(&schema, nil, "status", strfmt.Default).Validate(obj["status"])
	return result.Errors
}
