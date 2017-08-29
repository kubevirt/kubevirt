package apiserver

import (
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/api"
	"kubevirt.io/kubevirt/pkg/api/v1"
)

// Install registers the API group and adds types to a scheme
func Install(groupFactoryRegistry announced.APIGroupFactoryRegistry, registry *registered.APIRegistrationManager, scheme *runtime.Scheme) {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			// NOTE: RootScopedKinds are not namespaced. don't use it for general resources
			GroupName: v1.GroupName,
			//RootScopedKinds:            sets.NewString("VirtualMachine", "VirtualMachineList", "Migration", "MigrationList", "VirtualMachineReplicaSet", "VirtualMachineReplicaSetList"),
			VersionPreferenceOrder:     []string{v1.SchemeGroupVersion.Version},
			AddInternalObjectsToScheme: api.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			v1.SchemeGroupVersion.Version: v1.AddToScheme,
		},
	).Announce(groupFactoryRegistry).RegisterAndEnable(registry, scheme); err != nil {
		panic(err)
	}
}
