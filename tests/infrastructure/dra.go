package infrastructure

import (
	"bytes"
	"io"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/manifests"
	"kubevirt.io/kubevirt/tests/testsuite"
)

func DeployDRATestDriver() ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured
	namespace := testsuite.NamespacePrivileged
	for _, manifest := range [][]byte{
		manifests.ServiceAccount,
		manifests.RBAC,
		manifests.DeviceClass,
		manifests.DaemonSet,
	} {

		decoded, err := decodeYAMLObjects(manifest)
		if err != nil {
			return nil, err
		}
		objects = append(objects, decoded...)
	}
	for i := range objects {
		setObjectNamespace(&objects[i], namespace)
		if err := testsuite.ApplyRawManifest(objects[i]); err != nil {
			return nil, err
		}
	}
	return objects, nil
}

func setObjectNamespace(obj *unstructured.Unstructured, namespace string) {
	if obj.GetNamespace() != "" {
		obj.SetNamespace(namespace)
	}
	if obj.GetKind() == "ClusterRoleBinding" {
		subjects, found, _ := unstructured.NestedSlice(obj.Object, "subjects")
		if found {
			for i, s := range subjects {
				subject, ok := s.(map[string]any)
				if ok && subject["namespace"] != nil {
					subject["namespace"] = namespace
					subjects[i] = subject
				}
			}
			unstructured.SetNestedSlice(obj.Object, subjects, "subjects")
		}
	}
}

func DeployDRASidecarResourceClaimsPolicy() {
	manifest := []byte(`
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingAdmissionPolicy
metadata:
  name: dra-sidecar-resource-claims
spec:
  failurePolicy: Fail
  matchConditions:
  - name: has-hook-sidecar
    expression: object.spec.containers.exists(c, c.name == "hook-sidecar-0")
  - name: has-compute-with-claims
    expression: object.spec.containers.exists(c, c.name == "compute" && has(c.resources.claims))
  matchConstraints:
    matchPolicy: Equivalent
    namespaceSelector: {}
    objectSelector: {}
    resourceRules:
    - apiGroups: [""]
      apiVersions: ["v1"]
      operations: ["CREATE"]
      resources: ["pods"]
      scope: '*'
  mutations:
  - patchType: JSONPatch
    jsonPatch:
      expression: |
        [
          JSONPatch{
            op: "replace",
            path: "/spec/containers/" + string(object.spec.containers.map(c, c.name).indexOf("hook-sidecar-0")) + "/resources",
            value: {"claims": object.spec.containers.filter(c, c.name == "compute")[0].resources.claims}
          }
        ]
  reinvocationPolicy: IfNeeded
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingAdmissionPolicyBinding
metadata:
  name: dra-sidecar-resource-claims-binding
spec:
  policyName: dra-sidecar-resource-claims
  matchResources:
    resourceRules:
    - apiGroups: [""]
      apiVersions: ["v1"]
      resources: ["pods"]
      operations: ["CREATE"]
    objectSelector:
      matchLabels:
        kubevirt.io: virt-launcher
`)

	objects, err := decodeYAMLObjects(manifest)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	for _, obj := range objects {
		gomega.Expect(testsuite.ApplyRawManifest(obj)).To(gomega.Succeed())
	}

	ginkgo.DeferCleanup(func() {
		for i := len(objects) - 1; i >= 0; i-- {
			gomega.Expect(testsuite.DeleteRawManifest(objects[i])).To(gomega.Succeed())
		}
	})
}

func decodeYAMLObjects(data []byte) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 1024)
	docIndex := 0
	for {
		obj := map[string]any{}
		err := decoder.Decode(&obj)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(obj) == 0 {
			docIndex++
			continue
		}
		u := unstructured.Unstructured{Object: obj}
		objects = append(objects, u)
		docIndex++
	}
	return objects, nil
}
