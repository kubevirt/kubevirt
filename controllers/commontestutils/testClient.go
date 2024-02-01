package commontestutils

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type FakeWriteErrorGenerator func(obj client.Object) error
type FakeReadErrorGenerator func(key client.ObjectKey) error

// implements the client.Client interface (proxy pattern)
type HcoTestClient struct {
	client      client.Client
	sw          *HcoTestStatusWriter
	getError    FakeReadErrorGenerator
	createError FakeWriteErrorGenerator
	updateError FakeWriteErrorGenerator
	deleteError FakeWriteErrorGenerator
}

func (c *HcoTestClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return c.client.GroupVersionKindFor(obj)
}

func (c *HcoTestClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return c.client.IsObjectNamespaced(obj)
}

func (c *HcoTestClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if c.getError != nil {
		if err := c.getError(key); err != nil {
			return err
		}
	}
	return c.client.Get(ctx, key, obj, opts...)
}

func (c *HcoTestClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return c.client.List(ctx, list, opts...)
}

func (c *HcoTestClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if c.createError != nil {
		if err := c.createError(obj); err != nil {
			return err
		}
	}

	if err := checkDeadline(ctx); err != nil {
		return err
	}

	return c.client.Create(ctx, obj, opts...)
}

func (c *HcoTestClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if c.deleteError != nil {
		if err := c.deleteError(obj); err != nil {
			return err
		}
	}

	if err := checkDeadline(ctx); err != nil {
		return err
	}

	return c.client.Delete(ctx, obj, opts...)
}

func (c *HcoTestClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.updateError != nil {
		if err := c.updateError(obj); err != nil {
			return err
		}
	}

	if err := checkDeadline(ctx); err != nil {
		return err
	}

	return c.client.Update(ctx, obj, opts...)
}

func checkDeadline(ctx context.Context) error {
	if timeout, ok := ctx.Deadline(); ok && timeout.Before(time.Now()) {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func (c *HcoTestClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return c.client.Patch(ctx, obj, patch, opts...)
}

func (c *HcoTestClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return c.client.DeleteAllOf(ctx, obj, opts...)
}

func (c *HcoTestClient) Status() client.StatusWriter {
	return c.sw
}

func (c *HcoTestClient) SubResource(subResource string) client.SubResourceClient {
	return c.client.SubResource(subResource)
}

func (c *HcoTestClient) InitiateDeleteErrors(f FakeWriteErrorGenerator) {
	c.deleteError = f
}

func (c *HcoTestClient) InitiateUpdateErrors(f FakeWriteErrorGenerator) {
	c.updateError = f
}

func (c *HcoTestClient) InitiateGetErrors(f FakeReadErrorGenerator) {
	c.getError = f
}

func (c *HcoTestClient) InitiateCreateErrors(f FakeWriteErrorGenerator) {
	c.createError = f
}

func (c *HcoTestClient) Scheme() *runtime.Scheme {
	return c.client.Scheme()
}

func (c *HcoTestClient) RESTMapper() meta.RESTMapper {
	return c.client.RESTMapper()
}

type HcoTestStatusWriter struct {
	client client.SubResourceWriter
	errors TestErrors
}

func (sw *HcoTestStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	if ok, err := sw.errors.GetNextError(); ok {
		return err
	}
	return sw.client.Create(ctx, obj, subResource, opts...)
}

func (sw *HcoTestStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if ok, err := sw.errors.GetNextError(); ok {
		return err
	}
	return sw.client.Update(ctx, obj, opts...)
}

func (sw *HcoTestStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if ok, err := sw.errors.GetNextError(); ok {
		return err
	}
	return sw.client.Patch(ctx, obj, patch, opts...)
}

func (sw *HcoTestStatusWriter) InitiateErrors(errs ...error) {
	sw.errors = errs
}

type TestErrors []error

func (errs *TestErrors) GetNextError() (bool, error) {
	if len(*errs) == 0 {
		return false, nil
	}

	err := (*errs)[0]
	*errs = (*errs)[1:]

	return true, err
}

func InitClient(clientObjects []client.Object) *HcoTestClient {
	// Create a fake client to mock API calls
	cl := fake.NewClientBuilder().
		WithObjects(clientObjects...).
		WithScheme(GetScheme()).
		WithStatusSubresource(clientObjects...).
		Build()

	return &HcoTestClient{client: cl, sw: &HcoTestStatusWriter{client: cl.Status()}}
}
