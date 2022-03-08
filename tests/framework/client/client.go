package client

import (
	"sync"

	migrationsv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/migrations/v1alpha1"

	"kubevirt.io/kubevirt/tests/util"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"kubevirt.io/client-go/kubecli"
)

// TestClient is a wrapped kubevirt client that exposes all client functionality
// and can be used as a normal client. It provides a mechanism that track
// non-namespaced resources created or updated during e2e tests and functions to rollback them at the end of the execution.
type TestClient struct {
	kubecli.KubevirtClient
}

type CleanableResource interface {
	Clean()
}

var (
	coreOnce                  sync.Once
	testCoreV1Client          *TestCoreV1Client
	migrationOnce             sync.Once
	testMigrationPolicyClient *TestMigrationPolicyClient
	resourcesToClean          []CleanableResource
)

func GetKubevirtClient() (kubecli.KubevirtClient, error) {
	client, err := kubecli.GetKubevirtClient()
	testClient := TestClient{
		client,
	}
	return &testClient, err
}

func (c *TestClient) CoreV1() corev1.CoreV1Interface {
	config, err := kubecli.GetKubevirtClientConfig()
	util.PanicOnError(err)
	coreOnce.Do(func() {
		testCoreV1Client = CoreNewForConfig(config)
	})
	return testCoreV1Client
}

func (c *TestClient) MigrationPolicy() migrationsv1.MigrationPolicyInterface {
	config, err := kubecli.GetKubevirtClientConfig()
	util.PanicOnError(err)
	migrationOnce.Do(func() {
		testMigrationPolicyClient = MigrationPolicyNewForConfig(config)
	})

	return testMigrationPolicyClient.MigrationPolicies()
}

func (c *TestClient) Clean() {
	for resourceIndex, _ := range resourcesToClean {
		resourcesToClean[resourceIndex].Clean()
	}
}
