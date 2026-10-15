/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package instancetype_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"kubevirt.io/kubevirt/tests/integration/framework"
)

func TestInstancetypeIntegration(t *testing.T) {
	// Envtest uses this logger for control plane startup and CRD installation diagnostics.
	ctrllog.SetLogger(GinkgoLogr)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instancetype Integration Tests")
}

// instancetypeSuite shares the default controller configuration across test areas.
// Each spec gets its own namespace and fixtures; configuration-specific tests
// create separate frameworks in their own ordered containers.
type instancetypeSuite struct {
	f         *framework.Framework
	namespace string
}

var _ = Describe("Instancetype", Ordered, ContinueOnFailure, func() {
	s := &instancetypeSuite{}
	BeforeAll(func() {
		s.f = framework.New()
		s.f.Start(context.Background())
	})
	BeforeEach(func(ctx SpecContext) {
		s.namespace = s.f.CreateNamespace(ctx)
	})

	s.describeUpgrades()
	s.describeReferences()
	s.describeApplication()
})
