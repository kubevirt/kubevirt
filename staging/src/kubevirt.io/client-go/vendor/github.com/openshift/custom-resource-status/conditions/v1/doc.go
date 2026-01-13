// +k8s:deepcopy-gen=package,register
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true

// Package v1 provides version v1 of the types and functions necessary to
// manage and inspect a slice of conditions. It is opinionated in the
// condition types provided but leaves it to the user to define additional
// types as necessary.
package v1
