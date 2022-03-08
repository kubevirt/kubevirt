package client

import (
	"sync"

	"k8s.io/client-go/rest"

	migrationsv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/migrations/v1alpha1"
)

type TestMigrationPolicyClient struct {
	migrationsv1.MigrationsV1alpha1Client
}

var (
	migrationPolicyOnce      sync.Once
	migrationPolicyInterface migrationsv1.MigrationPolicyInterface
)

func (c *TestMigrationPolicyClient) MigrationPolicies() migrationsv1.MigrationPolicyInterface {
	migrationPolicyOnce.Do(func() {
		migrationPolicyInterface = newMigrationPolicies(c)
		resourcesToClean = append(resourcesToClean, migrationPolicyInterface.(CleanableResource))
	})
	return migrationPolicyInterface
}

// MigrationPolicyNewForConfig creates a new MigrationPolicy for the given RESTClient.
func MigrationPolicyNewForConfig(c *rest.Config) *TestMigrationPolicyClient {
	migrationPolicyClient := migrationsv1.NewForConfigOrDie(c)
	return &TestMigrationPolicyClient{*migrationPolicyClient}
}
