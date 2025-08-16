package migration

import (
	"fmt"
	"testing"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

const count = 9999

func BenchmarkListMatchingTargetPods(b *testing.B) {
	informer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
	controller := Controller{
		podIndexer: informer.GetIndexer(),
	}
	migration := v1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("something"),
			Namespace: "vm-debug",
		},
	}

	vmi := v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("something"),
			Namespace: "vm-debug",
		},
	}

	for i := range count {
		pod := k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID(fmt.Sprintf("something%d", i)),
				Namespace: "vm-debug",
				Name:      fmt.Sprintf("name%d", i),
				Labels: map[string]string{
					"kubevirt.io":                         "virt-launcher",
					"kubevirt.io/created-by":              "b0c482c0-0e0d-40b9-8dff-de967de41e5a",
					"kubevirt.io/domain":                  "rhel9",
					"kubevirt.io/migrationJobUID":         "815e781c-44f6-4ae7-9403-dc156cebd1fd",
					"kubevirt.io/migrationTargetNodeName": "e28-h03-000-r650",
					"kubevirt.io/nodeName":                "e28-h03-000-r650",
					"vm.kubevirt.io/name":                 "rhel9-2236",
				},
			},
		}
		err := controller.podIndexer.Add(&pod)
		if err != nil {
			panic(err)
		}
	}

	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("something"),
			Namespace: "vm-debug",
			Name:      "name",
			Labels: map[string]string{
				"kubevirt.io":                         "virt-launcher",
				"kubevirt.io/created-by":              string(vmi.UID),
				"kubevirt.io/domain":                  "rhel9",
				"kubevirt.io/migrationJobUID":         string(migration.UID),
				"kubevirt.io/migrationTargetNodeName": "e28-h03-000-r650",
				"kubevirt.io/nodeName":                "e28-h03-000-r650",
				"vm.kubevirt.io/name":                 "rhel9-2236",
			},
		},
	}

	err := controller.podIndexer.Add(&pod)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for range b.N {
		pods, err := controller.listMatchingTargetPods(&migration, &vmi)
		if err != nil {
			panic(err)
		}
		if len(pods) != 1 {
			panic("hi")

		}
		if pods[0] != &pod {
			panic("hi")
		}
	}

}

func BenchmarkListMatchingTargetPods2(b *testing.B) {
	informer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
	controller := Controller{
		podIndexer: informer.GetIndexer(),
	}
	controller.podIndexer.AddIndexers(cache.Indexers{
		"migrationJobUID": func(obj interface{}) ([]string, error) {
			mObj, err := meta.Accessor(obj)
			if err != nil {
				return []string{""}, fmt.Errorf("object has no meta: %v", err)
			}
			labels := mObj.GetLabels()
			value, ok := labels["kubevirt.io/migrationJobUID"]
			if !ok {
				return []string{""}, fmt.Errorf("object has no kubevirt.io/migrationJobUID label: %v", err)
			}
			return []string{value}, nil
		},
	})
	migration := v1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("something"),
			Namespace: "vm-debug",
		},
	}

	vmi := v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("something"),
			Namespace: "vm-debug",
		},
	}

	for i := range count {
		pod := k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID(fmt.Sprintf("something%d", i)),
				Namespace: "vm-debug",
				Name:      fmt.Sprintf("name%d", i),
				Labels: map[string]string{
					"kubevirt.io":                         "virt-launcher",
					"kubevirt.io/created-by":              "b0c482c0-0e0d-40b9-8dff-de967de41e5a",
					"kubevirt.io/domain":                  "rhel9",
					"kubevirt.io/migrationJobUID":         "815e781c-44f6-4ae7-9403-dc156cebd1fd",
					"kubevirt.io/migrationTargetNodeName": "e28-h03-000-r650",
					"kubevirt.io/nodeName":                "e28-h03-000-r650",
					"vm.kubevirt.io/name":                 "rhel9-2236",
				},
			},
		}
		err := controller.podIndexer.Add(&pod)
		if err != nil {
			panic(err)
		}
	}

	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("something"),
			Namespace: "vm-debug",
			Name:      "name",
			Labels: map[string]string{
				"kubevirt.io":                         "virt-launcher",
				"kubevirt.io/created-by":              string(vmi.UID),
				"kubevirt.io/domain":                  "rhel9",
				"kubevirt.io/migrationJobUID":         string(migration.UID),
				"kubevirt.io/migrationTargetNodeName": "e28-h03-000-r650",
				"kubevirt.io/nodeName":                "e28-h03-000-r650",
				"vm.kubevirt.io/name":                 "rhel9-2236",
			},
		},
	}

	err := controller.podIndexer.Add(&pod)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for range b.N {
		pods, err := controller.listMatchingTargetPods2(&migration, &vmi)
		if err != nil {
			panic(err)
		}
		if len(pods) != 1 {
			panic("hi")

		}
		if pods[0] != &pod {
			panic("hi")
		}
	}

}

const migrationCount = 10000

func BenchmarkListMigrationsMatchingVMI(b *testing.B) {
	informer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceMigration{})
	controller := Controller{
		migrationIndexer: informer.GetIndexer(),
	}

	for i := range migrationCount - 10 {
		migration := v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID(fmt.Sprintf("something%d", i)),
				Name:      fmt.Sprintf("name%d", i),
				Namespace: "vm-debug",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: fmt.Sprintf("%d", i),
			},
		}
		if err := controller.migrationIndexer.Add(&migration); err != nil {
			b.Fatal(err)
		}
	}
	for i := range 10 {
		migration := v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID(fmt.Sprintf("something%d", i)),
				Name:      fmt.Sprintf("name%d", i),
				Namespace: "vm-debug",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: "theone",
			},
		}
		if err := controller.migrationIndexer.Add(&migration); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for range b.N {
		migrations, err := controller.listMigrationsMatchingVMI("vm-debug", "theone")
		if err != nil {
			b.Fatal(err)
		}
		if len(migrations) != 10 {
			b.Fatal("wexected 10 migrations")
		}
	}

}

func BenchmarkListMigrationsMatchingVMI2(b *testing.B) {
	informer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceMigration{})
	controller := Controller{
		migrationIndexer: informer.GetIndexer(),
	}
	controller.migrationIndexer.AddIndexers(cache.Indexers{
		vmiIndex: func(obj interface{}) ([]string, error) {
			mig, ok := obj.(*v1.VirtualMachineInstanceMigration)
			if !ok {
				return []string{}, fmt.Errorf("not migration")
			}
			return []string{fmt.Sprintf("%s/%s", mig.Namespace, mig.Spec.VMIName)}, nil
		},
	})

	for i := range migrationCount - 10 {
		migration := v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID(fmt.Sprintf("something%d", i)),
				Name:      fmt.Sprintf("name%d", i),
				Namespace: "vm-debug",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: fmt.Sprintf("%d", i),
			},
		}
		if err := controller.migrationIndexer.Add(&migration); err != nil {
			b.Fatal(err)
		}
	}
	for i := range 10 {
		migration := v1.VirtualMachineInstanceMigration{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID(fmt.Sprintf("something%d", i)),
				Name:      fmt.Sprintf("name%d", i),
				Namespace: "vm-debug",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: "theone",
			},
		}
		if err := controller.migrationIndexer.Add(&migration); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for range b.N {
		migrations, err := controller.filterMigrationsByVMI("vm-debug", "theone",
			func(migration *v1.VirtualMachineInstanceMigration) bool { return true },
		)
		if err != nil {
			b.Fatal(err)
		}
		if len(migrations) != 10 {
			b.Fatal("expected 10 migration")
		}
	}
}
