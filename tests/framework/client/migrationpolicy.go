package client

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	migrationsv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/migrations/v1alpha1"

	v1alpha1 "kubevirt.io/api/migrations/v1alpha1"
)

var createdMigrationPolicies = make(map[string]context.Context, 0)

// migrationPolicies implements MigrationPolicyInterface
type migrationPolicies struct {
	migrationsv1.MigrationPolicyInterface
}

// newMigrationPolicies returns a MigrationPolicies
func newMigrationPolicies(c *TestMigrationPolicyClient) *migrationPolicies {
	return &migrationPolicies{
		c.MigrationsV1alpha1Client.MigrationPolicies(),
	}
}

func (c *migrationPolicies) Clean() {
	for name, ctx := range createdMigrationPolicies {
		err := c.Delete(ctx, name, v1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			panic(err)
		}
	}
}

func (c *migrationPolicies) Create(ctx context.Context, migrationPolicy *v1alpha1.MigrationPolicy, opts v1.CreateOptions) (result *v1alpha1.MigrationPolicy, err error) {
	created, err := c.MigrationPolicyInterface.Create(ctx, migrationPolicy, opts)
	if err == nil && opts.DryRun == nil {
		createdMigrationPolicies[created.Name] = ctx
	}

	return created, err
}

func (c *migrationPolicies) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	err := c.MigrationPolicyInterface.Delete(ctx, name, opts)
	if _, exist := createdMigrationPolicies[name]; exist && err == nil && opts.DryRun == nil {
		delete(createdMigrationPolicies, name)
	}

	return err
}
