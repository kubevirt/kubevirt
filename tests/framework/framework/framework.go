/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file was originally copied from https://github.com/kubernetes/kubernetes/blob/master/test/e2e/framework/framework.go

package framework

import (
	"fmt"
	"math/rand"

	testclient "kubevirt.io/kubevirt/tests/framework/client"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/client-go/kubecli"

	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo/v2"
)

// Framework supports common operations used by e2e tests; it will keep a client & clean the cluster for you.
type Framework struct {
	BaseName string

	UniqueName string

	clientConfig   *rest.Config
	KubevirtClient kubecli.KubevirtClient
	// afterEaches is a map of name to function to be called after each test.  These are not
	// cleared.  The call order is randomized so that no dependencies can grow between
	// the various afterEaches
	afterEaches map[string]AfterEachActionFunc
}

// AfterEachActionFunc is a function that can be called after each test
type AfterEachActionFunc func(f *Framework, failed bool)

// NewDefaultFramework makes a new framework and sets up a BeforeEach/AfterEach for
// you (you can write additional before/after each functions).
func NewDefaultFramework(baseName string) *Framework {
	return NewFramework(baseName, nil, nil)
}

// NewFramework creates a test framework.
func NewFramework(baseName string, client kubecli.KubevirtClient, config *rest.Config) *Framework {
	f := &Framework{
		BaseName:       baseName,
		KubevirtClient: client,
		clientConfig:   config,
	}
	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

// BeforeEach gets a client and clean the cluster.
func (f *Framework) BeforeEach() {
	if f.KubevirtClient == nil {
		By("Creating a kubevirt client")
		client, err := testclient.GetKubevirtClient()
		util.PanicOnError(err)
		f.KubevirtClient = client
		config, err := kubecli.GetKubevirtClientConfig()
		util.PanicOnError(err)
		f.clientConfig = config
	}
	f.UniqueName = fmt.Sprintf("%s-%08x", f.BaseName, rand.Int31())
	tests.BeforeTestCleanup()
}

// AddAfterEach is a way to add a function to be called after every test.  The execution order is intentionally random
// to avoid growing dependencies.  If you register the same name twice, it is a coding error and will panic.
func (f *Framework) AddAfterEach(name string, fn AfterEachActionFunc) {
	if _, ok := f.afterEaches[name]; ok {
		panic(fmt.Sprintf("%q is already registered", name))
	}

	if f.afterEaches == nil {
		f.afterEaches = map[string]AfterEachActionFunc{}
	}
	f.afterEaches[name] = fn
}

// AfterEach deletes the namespace, after reading its events.
func (f *Framework) AfterEach() {
	// This should not happen. Given KubevirtClient is a public field a test must have updated it!
	// Error out early before any API calls during cleanup.
	if f.KubevirtClient == nil {
		Fail("The framework KubevirtClient must not be nil at this point")
	}
	f.KubevirtClient.(testclient.CleanableResource).Clean()

	// run all aftereach functions in random order to ensure no dependencies grow
	for _, afterEachFn := range f.afterEaches {
		afterEachFn(f, CurrentSpecReport().Failed())
	}
}
