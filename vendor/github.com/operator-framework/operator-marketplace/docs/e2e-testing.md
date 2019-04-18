# Marketplace End to End (e2e) Testing

The marketplace operator leverages the [Operator-SDK](https://github.com/operator-framework/operator-sdk/) to create and run its e2e tests. While the [Operator-SDK's e2e Test Framework](https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/writing-e2e-tests.md) provides a number of tools that makes writing e2e tests easier, it is up to the authors of the operator to decide how to implement the tests. This document will cover the approach that the marketplace team has taken and will provide users with enough information to extend the e2e test portfolio.

## Marketplace e2e Design

### Entry Point

Marketplace e2e test code is contained within the [main_test.go](../test/e2e/main_test.go) file. When the e2e tests are kicked off, the `TestMarketplace` function will be called which will initialize the test framework and run all e2e tests.

### Test Suites

Marketplace tests that validate similar functionality are grouped together in test suites. The [testsuites package](../test/testsuites) contains functions that implements these test suites. It often makes sense to reuse runtime objects between tests that have similar dependencies - this can be acomplished by following the steps outlined below:

```go
// e2eEntryPoint is the entry point for the e2e tests
func e2eEntryPoint(t *testing.T) {
    // init code...

    // Call the test suite.
    t.Run("test-suite", genericTestSuite)
}

// genericTestSuite implements a test suite.
func genericTestSuite(t *testing.T) {
    // Use ctx if you want to create runtime object that will be used by tests in the test suite.
    ctx := test.NewTestCtx(t)

    // Defer a cleanup for the runtime objects you create.
    defer ctx.Cleanup()

    // Code that creates a runtime object using the context above...
    // Get global framework variables.
    f := test.Global

    // Create the operatorsource.
    err = helpers.CreateRuntimeObject(f, ctx, helpers.CreateOperatorSource(namespace))
    if err != nil {
        t.Errorf("Could not create operator source: %v", err)
    }

    // Run the tests that rely on the runtime objects created earlier.
    t.Run("test1", testsuites.Test1)
    t.Run("test2", testsuites.Test2)
}
```

Adhere to the following practices when implementing your test suites:

* Do not add multiple testsuites to a single file. When creating a new test suite, the file name you create should describe the focus of the tests and end with `tests.go`.
* Avoid altering runtime objects that multiple tests rely on - if there is no work around revert any changes at the end of the test or move the test out of the suite.
* If a test doesn't create any new runtime object, don't call `test.NewTestCtx(t)`.
* If a test creates new runtime objects, make sure that the ctx is created in a method with the `func(t testting.T) error` signature and pass the context into a method that implements the test.
* In version 0.3.0 of the Operator-SDK, the `TestCtx.Cleanup()` method does not wait for runtime objects to be deleted before exiting. Until we can upgrade to the latest version of the Operator-SDK tests must cleanup after themselves.

### Test Groups

Test groups are very similar to test suites but they are used to prepare the test environment for a series of test suites and then run said test suites. The [testgroups package](../test/testgroups) contains functions that implements these test groups.

### Helper Functions

The [helpers package](../test/helpers) contains useful functions that can be shared accross a variety of e2e tests.

## Running e2e Tests

To run the e2e tests defined in test/e2e that were created using the operator-sdk, first ensure that you have the following additional prerequisites:

1. The operator-sdk binary installed on your environment. You can get it by either downloading a released binary on the sdk release page [here](https://github.com/operator-framework/operator-sdk/releases/) or by pulling down the source and compiling it [locally](https://github.com/operator-framework/operator-sdk).
2. A namespace on your cluster to run the tests on, e.g.
```bash
    $ oc create namespace test-namespace
```
3. A Kubeconfig file that points to the cluster you want to run the tests on.

To run the tests, just call operator-sdk test and point to the test directory:

```bash
operator-sdk test local ./test/e2e --up-local --kubeconfig=$KUBECONFIG --namespace $TEST_NAMESPACE
```

You can also run the tests with `make e2e-test`.
