package namespaced

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

// FactoryArgs contains the required parameters to generate all namespaced resources
type FactoryArgs struct {
	DockerRepo        string
	DockerTag         string
	ControllerImage   string
	ImporterImage     string
	ClonerImage       string
	APIServerImage    string
	UploadProxyImage  string
	UploadServerImage string
	Verbosity         string
	PullPolicy        string
	Namespace         string
}

type factoryFunc func(*FactoryArgs) []Resource

// Resource defines the interface for namespaced resources
type Resource interface {
	runtime.Object
	SetNamespace(string)
	GetNamespace() string
}

var factoryFunctions = map[string]factoryFunc{
	"apiserver":   createAPIServerResources,
	"controller":  createControllerResources,
	"uploadproxy": createUploadProxyResources,
}

// CreateAllResources creates all namespaced resources
func CreateAllResources(args *FactoryArgs) ([]Resource, error) {
	resources := []Resource{}
	for group := range factoryFunctions {
		rs, err := CreateResourceGroup(group, args)
		if err != nil {
			return nil, err
		}
		resources = append(resources, rs...)
	}
	return resources, nil
}

// CreateResourceGroup creates namespaced resources for a specific group/component
func CreateResourceGroup(group string, args *FactoryArgs) ([]Resource, error) {
	f, ok := factoryFunctions[group]
	if !ok {
		return nil, fmt.Errorf("Group %s does not exist", group)
	}
	resources := []Resource{}
	for _, o := range f(args) {
		if o.GetNamespace() == "" {
			o.SetNamespace(args.Namespace)
		}
		resources = append(resources, o)
	}
	return resources, nil
}
