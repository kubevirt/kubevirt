package apiserver

import (
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-apiserver/registry/migration"
	"kubevirt.io/kubevirt/pkg/virt-apiserver/registry/vm"
	"kubevirt.io/kubevirt/pkg/virt-apiserver/registry/vmreplicaset"
)

var (
	groupFactoryRegistry = make(announced.APIGroupFactoryRegistry)
	registry             = registered.NewOrDie("")
	Scheme               = runtime.NewScheme()
	Codecs               = serializer.NewCodecFactory(Scheme)
	ParameterCodec       = runtime.NewParameterCodec(Scheme)
)

func init() {
	logging.DefaultLogger().Info().Msg("Registering KubeVirt known types")
	Install(groupFactoryRegistry, registry, Scheme)

	// Note: Adding these boilderplate items is marked as
	// "find a way to remove this"
	// in the Kubernetes sample-apiserver example code.
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)

	metav1.AddToGroupVersion(Scheme, v1.GroupVersion)
	v1.AddKnownTypes(Scheme)
}

type Config struct {
	GenericConfig *genericapiserver.Config
}

type VirtApiServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	*Config
}

func (c completedConfig) New() (*VirtApiServer, error) {
	genericServer, err := c.Config.GenericConfig.SkipComplete().New("kubevirt-apiserver", genericapiserver.EmptyDelegate)
	if err != nil {
		return nil, err
	}

	s := &VirtApiServer{
		GenericAPIServer: genericServer,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(v1.GroupName, registry, Scheme, ParameterCodec, Codecs)
	apiGroupInfo.GroupMeta.GroupVersion = v1.SchemeGroupVersion
	storage := map[string]rest.Storage{}
	vms, err := vm.NewREST(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}
	storage["virtualmachines"] = vms

	vmrs, err := vmreplicaset.NewREST(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}
	storage["virtualmachinereplicasets"] = vmrs

	migrations, err := migration.NewREST(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}
	storage["migrations"] = migrations
	apiGroupInfo.VersionedResourcesStorageMap[v1.GroupVersion.Version] = storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}

// SkipComplete provides a way to construct a server instance without config completion.
func (c *Config) SkipComplete() completedConfig {
	return completedConfig{c}
}

func (c *Config) Complete() completedConfig {
	c.GenericConfig.Complete()
	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}
	return completedConfig{c}
}
