package openapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	openapi_validate "github.com/go-openapi/validate"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime/schema"
	builderv3 "k8s.io/kube-openapi/pkg/builder3"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/common/restfuladapter"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/errors"
	validationspec "k8s.io/kube-openapi/pkg/validation/spec"
	"kubevirt.io/client-go/api"
)

type Validator struct {
	specSchemes   *spec.Schema
	statusSchemes *spec.Schema
	topLevelKeys  map[string]interface{}
}

func CreateConfig() *common.OpenAPIV3Config {
	return &common.OpenAPIV3Config{
		CommonResponses: map[int]*spec3.Response{
			401: {
				ResponseProps: spec3.ResponseProps{
					Description: "Unauthorized",
				},
			},
		},
		Info: &validationspec.Info{
			InfoProps: validationspec.InfoProps{
				Title:       "KubeVirt API",
				Description: "This is KubeVirt API an add-on for Kubernetes.",
				Contact: &validationspec.ContactInfo{
					Name:  "kubevirt-dev",
					Email: "kubevirt-dev@googlegroups.com",
					URL:   "https://github.com/kubevirt/kubevirt",
				},
				License: &validationspec.License{
					Name: "Apache 2.0",
					URL:  "https://www.apache.org/licenses/LICENSE-2.0",
				},
			},
		},
		SecuritySchemes: spec3.SecuritySchemes{
			"BearerToken": &spec3.SecurityScheme{
				SecuritySchemeProps: spec3.SecuritySchemeProps{
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
		GetDefinitionName: func(name string) (string, validationspec.Extensions) {
			if strings.Contains(name, "kubevirt.io") {
				// keeping for validation
				return name[strings.LastIndex(name, "/")+1:], nil
			}
			//adpting k8s style
			return strings.ReplaceAll(name, "/", "."), nil
		},
	}
}

func LoadOpenAPISpec(webServices []*restful.WebService) *spec3.OpenAPI {
	config := CreateConfig()
	openapispec, err := builderv3.BuildOpenAPISpecFromRoutes(restfuladapter.AdaptWebServices(webServices), config)
	if err != nil {
		panic(fmt.Errorf("Failed to build swagger: %s", err))
	}

	// creationTimestamp, lastProbeTime and lastTransitionTime are deserialized as "null"
	// Fix it here until
	// https://github.com/kubernetes/kubernetes/issues/66899 is ready
	// Otherwise CRDs can't use templates which contain metadata and controllers
	// can't set conditions without timestamps

	objectmeta := ""
	for k := range openapispec.Components.Schemas {
		if strings.Contains(k, "v1.ObjectMeta") {
			objectmeta = k
			break
		}
	}
	resourceRequirements, exists := openapispec.Components.Schemas["v1.ResourceRequirements"]
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

	objectMeta, exists := openapispec.Components.Schemas[objectmeta]
	if exists {
		prop := objectMeta.Properties["creationTimestamp"]
		prop.Type = validationspec.StringOrArray{"string", "null"}
		// mask v1.Time as in validation v1.Time override sting,null type
		prop.Ref = validationspec.Ref{}
		objectMeta.Properties["creationTimestamp"] = prop
	}

	for k, s := range openapispec.Components.Schemas {
		// allow nullable statuses
		if status, found := s.Properties["status"]; found {
			if !status.Type.Contains("string") {
				var definitionName string
				if !status.Ref.GetPointer().IsEmpty() {
					definitionName = strings.Split(status.Ref.GetPointer().String(), "/")[3]
				} else if len(status.AllOf) > 0 && !status.AllOf[0].Ref.GetPointer().IsEmpty() {
					definitionName = strings.Split(status.AllOf[0].Ref.GetPointer().String(), "/")[3]
				} else {
					continue
				}

				object := openapispec.Components.Schemas[definitionName]
				object.Nullable = true
				openapispec.Components.Schemas[definitionName] = object
			}
		}

		if strings.HasSuffix(k, "Condition") {
			prop := s.Properties["lastProbeTime"]
			prop.Type = validationspec.StringOrArray{"string", "null"}
			prop.Ref = validationspec.Ref{}
			s.Properties["lastProbeTime"] = prop

			prop = s.Properties["lastTransitionTime"]
			prop.Type = validationspec.StringOrArray{"string", "null"}
			prop.Ref = validationspec.Ref{}
			s.Properties["lastTransitionTime"] = prop
		}
		if strings.Contains(k, "v1.HTTPGetAction") {
			prop := s.Properties["port"]
			prop.Type = validationspec.StringOrArray{"string", "number"}
			// As intstr.IntOrString, the ref for that must be masked
			prop.Ref = validationspec.Ref{}
			s.Properties["port"] = prop
		}
		if strings.Contains(k, "v1.TCPSocketAction") {
			prop := s.Properties["port"]
			prop.Type = validationspec.StringOrArray{"string", "number"}
			// As intstr.IntOrString, the ref for that must be masked
			prop.Ref = validationspec.Ref{}
			s.Properties["port"] = prop
		}
		if strings.Contains(k, "v1.PersistentVolumeClaimSpec") {
			for i, r := range s.Required {
				if r == "dataSource" {
					s.Required = append(s.Required[:i], s.Required[i+1:]...)
					openapispec.Components.Schemas[k] = s
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
