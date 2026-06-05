package cleanup

import (
	"fmt"
)

const (
	KubeVirtTestLabelPrefix = "test.kubevirt.io"
)

// TestLabelForNamespace is used to mark non-namespaces resources with a label bound to a test namespace.
// This will be used to clean up non-namespaced resources after a test case was executed.
func TestLabelForNamespace(namespace string) string {
	return fmt.Sprintf("%s/%s", KubeVirtTestLabelPrefix, namespace)
}
