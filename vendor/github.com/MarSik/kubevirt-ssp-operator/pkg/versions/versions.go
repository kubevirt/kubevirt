// package versions contains constants for the default versions of the
// various SSP sub-components to deploy
package versions

import "fmt"

const (
	KubevirtCommonTemplates    string = "0.6.2"
	KubevirtNodeLabeller       string = "0.1.1"
	KubevirtTemplateValidator  string = "0.6.2"
	KubevirtMetricsAggregation string = "0.0.1"
)

// TagForVersion converts the given version in a suitable tag
func TagForVersion(ver string) string {
	return fmt.Sprintf("v%s", ver)
}

// FullVersionString converts the given version in a semantic version identifier
func FullVersionString(ver string) string {
	return fmt.Sprintf("v%s", ver)
}
