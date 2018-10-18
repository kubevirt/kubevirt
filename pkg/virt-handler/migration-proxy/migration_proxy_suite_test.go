package migrationproxy_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/log"
)

func TestMigrationProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	log.Log.SetIOWriter(GinkgoWriter)
	RunSpecs(t, "MigrationProxy Suite")
}
