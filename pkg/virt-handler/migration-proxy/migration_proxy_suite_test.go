package migrationproxy_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMigrationProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MigrationProxy Suite")
}
