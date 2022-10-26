// +k8s:deepcopy-gen=package,register
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true

// Package v1 provides version v1 of the functions necessary to
// manage and inspect a slice of object references. This can be
// used to add a RelatedObjects field on the status of your custom
// resource, adding objects that your operator manages to the status.
package v1
