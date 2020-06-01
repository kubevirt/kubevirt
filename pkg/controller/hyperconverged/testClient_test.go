package hyperconverged

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type hcoTestClient struct {
	client      client.Client
	sw          *hcoTestStatusWriter
	readErrors  testErrors
	writeErrors testErrors
}

func (c *hcoTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	if ok, err := c.readErrors.getNextError(); ok {
		return err
	}
	return c.client.Get(ctx, key, obj)
}

func (c *hcoTestClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	if ok, err := c.writeErrors.getNextError(); ok {
		return err
	}
	return c.client.List(ctx, list, opts...)
}

func (c *hcoTestClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	if ok, err := c.writeErrors.getNextError(); ok {
		return err
	}
	return c.client.Create(ctx, obj, opts...)
}

func (c *hcoTestClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	if ok, err := c.writeErrors.getNextError(); ok {
		return err
	}
	return c.client.Delete(ctx, obj, opts...)
}

func (c *hcoTestClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	if ok, err := c.writeErrors.getNextError(); ok {
		return err
	}
	return c.client.Update(ctx, obj, opts...)
}

func (c *hcoTestClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	if ok, err := c.writeErrors.getNextError(); ok {
		return err
	}
	return c.client.Patch(ctx, obj, patch, opts...)
}

func (c *hcoTestClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	if ok, err := c.writeErrors.getNextError(); ok {
		return err
	}
	return c.client.DeleteAllOf(ctx, obj, opts...)
}

func (c *hcoTestClient) Status() client.StatusWriter {
	return c.sw
}

func (c *hcoTestClient) initiateReadErrors(errs ...error) {
	c.readErrors = errs
}

func (c *hcoTestClient) initiateWriteErrors(errs ...error) {
	c.writeErrors = errs
}

type hcoTestStatusWriter struct {
	client client.Client
	errors testErrors
}

func (sw *hcoTestStatusWriter) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	if ok, err := sw.errors.getNextError(); ok {
		return err
	}
	return sw.client.Update(ctx, obj, opts...)
}

func (sw *hcoTestStatusWriter) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	if ok, err := sw.errors.getNextError(); ok {
		return err
	}
	return sw.client.Patch(ctx, obj, patch, opts...)
}

func (sw *hcoTestStatusWriter) initiateErrors(errs ...error) {
	sw.errors = errs
}

type testErrors []error

func (errs *testErrors) getNextError() (bool, error) {
	if len(*errs) == 0 {
		return false, nil
	}

	err := (*errs)[0]
	*errs = (*errs)[1:]

	return true, err
}

func initClient(clientObjects []runtime.Object) *hcoTestClient {
	// Create a fake client to mock API calls
	cl := fake.NewFakeClient(clientObjects...)
	return &hcoTestClient{client: cl, sw: &hcoTestStatusWriter{client: cl}}
}
