package controlplane_test

import (
	"context"
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"

	"kubevirt.io/kubevirt/tests/integration/framework"
)

const (
	// celCostLimitHit is part of every API server error for a CEL cost limit: the budget for the
	// whole object, or the limit for a single rule.
	celCostLimitHit = "no further validation rules will be run"
	// celSkipped means the object had errors that make the API server skip CEL validation.
	celSkipped = "some validation rules were not checked"
)

// Regression test: rules that each pass the CRD install checks can still exhaust the API
// server's per-object CEL cost budget together. For every CRD, this creates the largest object
// the schema allows against CRDs carrying every rule twice, so it fails once real rules use
// half the budget or one rule exceeds its own limit. The cost is approximate: unbounded lists
// and strings hold one item, and transition and status-subresource rules don't run.
var _ = Describe("CEL validation cost", func() {
	var f *framework.Framework

	BeforeEach(func() {
		f = framework.New(framework.WithCRDTransform(duplicateCELRules))
		f.Start(context.Background())
	})

	It("should keep maximal objects of every CRD within half the CEL cost budget", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		crds, err := f.VirtClient().ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		tested := 0

		for _, crd := range crds.Items {
			for _, version := range crd.Spec.Versions {
				if !version.Served {
					continue
				}
				tested++
				target := crd.Name + "/" + version.Name

				// CRD root schemas are objects, so the generated value is a map.
				object := maximalValue(version.Schema.OpenAPIV3Schema).(map[string]any)
				object["apiVersion"] = crd.Spec.Group + "/" + version.Name
				object["kind"] = crd.Spec.Names.Kind
				object["metadata"] = map[string]any{"name": "maximal"}

				gvr := schema.GroupVersionResource{Group: crd.Spec.Group, Version: version.Name, Resource: crd.Spec.Names.Plural}
				resource := f.VirtClient().DynamicClient().Resource(gvr)
				var client dynamic.ResourceInterface = resource
				if crd.Spec.Scope == extv1.NamespaceScoped {
					client = resource.Namespace(ns)
				}

				_, err = client.Create(ctx, &unstructured.Unstructured{Object: object}, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
				// A rejection is fine, since the placeholder values ignore patterns and the rules; so is success.
				if err != nil {
					Expect(apierrors.IsInvalid(err)).To(BeTrue(), "%s: %v", target, err)
					Expect(err.Error()).NotTo(ContainSubstring(celCostLimitHit), "%s hit a CEL cost limit with every rule installed twice", target)
					Expect(err.Error()).NotTo(ContainSubstring(celSkipped), "%s: the generated object skipped CEL validation", target)
				}
			}
		}

		Expect(tested).NotTo(BeZero(), "no CRD version was tested")
	})
})

// maximalValue returns a value for props with every field set and every list and string at its
// maximum. It only satisfies what makes the API server skip CEL validation when violated: types
// (including date-time formats), enums, required fields and size limits.
func maximalValue(props *extv1.JSONSchemaProps) any {
	GinkgoHelper()
	if enum := enumOf(props); len(enum) > 0 {
		var value any
		Expect(json.Unmarshal(enum[0].Raw, &value)).To(Succeed())
		return value
	}
	if props.XIntOrString {
		return 1
	}

	switch props.Type {
	case "array":
		// The items are identical, which is fine: duplicate list keys are reported but do not
		// make the API server skip CEL.
		item := maximalValue(props.Items.Schema)
		list := make([]any, ptr.Deref(props.MaxItems, 1))
		for i := range list {
			list[i] = item
		}
		return list
	case "string":
		if props.Format == "date-time" {
			return "2026-01-01T00:00:00Z"
		}
		return strings.Repeat("a", int(ptr.Deref(props.MaxLength, 1)))
	case "integer", "number":
		return 1
	case "boolean":
		return true
	default: // "object", or an untyped x-kubernetes-preserve-unknown-fields node
		object := map[string]any{}
		for name, property := range props.Properties {
			object[name] = maximalValue(&property)
		}
		if props.AdditionalProperties != nil && props.AdditionalProperties.Schema != nil {
			object["key"] = maximalValue(props.AdditionalProperties.Schema)
		}
		return object
	}
}

// enumOf returns the allowed values of props. controller-gen nests them in allOf when both a
// field and its type declare an enum.
func enumOf(props *extv1.JSONSchemaProps) []extv1.JSON {
	if len(props.Enum) == 0 && len(props.AllOf) > 0 {
		return props.AllOf[0].Enum
	}
	return props.Enum
}

// duplicateCELRules installs every CEL rule twice. Rules must be unique, so the copy is wrapped
// in parentheses, which does not change its cost.
func duplicateCELRules(crd *extv1.CustomResourceDefinition) {
	for _, version := range crd.Spec.Versions {
		duplicateSchemaRules(version.Schema.OpenAPIV3Schema)
	}
}

// duplicateSchemaRules doubles the rules on props and recurses into every nested schema. Map
// values are not addressable, so each visited property is written back.
func duplicateSchemaRules(props *extv1.JSONSchemaProps) {
	original := props.XValidations
	for _, rule := range original {
		rule.Rule = "(" + rule.Rule + ")"
		props.XValidations = append(props.XValidations, rule)
	}
	for name, property := range props.Properties {
		duplicateSchemaRules(&property)
		props.Properties[name] = property
	}
	if props.Items != nil && props.Items.Schema != nil {
		duplicateSchemaRules(props.Items.Schema)
	}
	if props.AdditionalProperties != nil && props.AdditionalProperties.Schema != nil {
		duplicateSchemaRules(props.AdditionalProperties.Schema)
	}
}
