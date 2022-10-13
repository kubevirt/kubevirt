package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/emicklei/go-restful"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/instancetype"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

const (
	messageAccessNotAllowed = "Subject not allowed to access resource"
)

func (app *SubresourceAPIApp) ExpandSpecRequestHandler(request *restful.Request, response *restful.Response) {
	if request.Request.Body == nil {
		writeError(k8serr.NewBadRequest("empty request body"), response)
		return
	}

	bodyBytes, err := io.ReadAll(request.Request.Body)
	if err != nil {
		writeError(k8serr.NewBadRequest(err.Error()), response)
		return
	}

	rawObj := map[string]interface{}{}
	err = json.Unmarshal(bodyBytes, &rawObj)
	if err != nil {
		writeError(k8serr.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
		return
	}

	validationErrors := definitions.Validator.Validate(v1.VirtualMachineGroupVersionKind, rawObj)
	if len(validationErrors) > 0 {
		writeValidationErrors(validationErrors, response)
		return
	}

	vm := &v1.VirtualMachine{}
	err = json.Unmarshal(bodyBytes, vm)
	if err != nil {
		writeError(k8serr.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
		return
	}

	instancetype.SetDefaultInstancetypeKind(vm)
	instancetype.SetDefaultPreferenceKind(vm)

	app.expandSpecResponse(vm, request, response, func(err error) *k8serr.StatusError {
		return k8serr.NewBadRequest(err.Error())
	})
}

func (app *SubresourceAPIApp) ExpandSpecVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}

	app.expandSpecResponse(vm, request, response, k8serr.NewInternalError)
}

func (app *SubresourceAPIApp) expandSpecResponse(vm *v1.VirtualMachine, request *restful.Request, response *restful.Response, errorFunc func(error) *k8serr.StatusError) {
	authclient := app.virtCli.AuthorizationV1().SubjectAccessReviews()
	authorizer := NewAuthorizorFromClient(authclient)
	sar, err := authorizer.NewSubjectAccessReview(request)
	if err != nil {
		writeError(k8serr.NewInternalError(err), response)
		return
	}

	instancetypeResourceAttributes, err := instancetype.CreateInstancetypeResourceAttributes(vm, "get")
	if err != nil {
		writeError(errorFunc(err), response)
		return
	}
	if instancetypeResourceAttributes != nil {
		sar.Spec.ResourceAttributes = instancetypeResourceAttributes
		result, err := authclient.Create(context.Background(), sar, metav1.CreateOptions{})
		if err != nil {
			writeError(k8serr.NewInternalError(err), response)
			return
		}
		if !result.Status.Allowed {
			writeError(
				k8serr.NewForbidden(
					schema.GroupResource{
						Resource: instancetypeResourceAttributes.Resource,
					},
					instancetypeResourceAttributes.Name,
					errors.New(messageAccessNotAllowed),
				),
				response,
			)
			return
		}
	}

	instancetypeSpec, err := app.instancetypeMethods.FindInstancetypeSpec(vm)
	if err != nil {
		writeError(errorFunc(err), response)
		return
	}

	preferenceResourceAttributes, err := instancetype.CreatePreferenceResourceAttributes(vm, "get")
	if err != nil {
		writeError(errorFunc(err), response)
		return
	}
	if preferenceResourceAttributes != nil {
		sar.Spec.ResourceAttributes = preferenceResourceAttributes
		result, err := authclient.Create(context.Background(), sar, metav1.CreateOptions{})
		if err != nil {
			writeError(k8serr.NewInternalError(err), response)
			return
		}
		if !result.Status.Allowed {
			writeError(
				k8serr.NewForbidden(
					schema.GroupResource{
						Resource: preferenceResourceAttributes.Resource,
					},
					preferenceResourceAttributes.Name,
					errors.New(messageAccessNotAllowed),
				),
				response,
			)
			return
		}
	}

	preferenceSpec, err := app.instancetypeMethods.FindPreferenceSpec(vm)
	if err != nil {
		writeError(errorFunc(err), response)
		return
	}

	if instancetypeSpec == nil && preferenceSpec == nil {
		err := response.WriteEntity(vm)
		if err != nil {
			log.Log.Reason(err).Error("Failed to write http response.")
		}
		return
	}

	conflicts := app.instancetypeMethods.ApplyToVmi(field.NewPath("spec", "template", "spec"), instancetypeSpec, preferenceSpec, &vm.Spec.Template.Spec)
	if len(conflicts) > 0 {
		writeError(errorFunc(fmt.Errorf("cannot expand instancetype to VM")), response)
		return
	}

	// Remove InstancetypeMatcher and PreferenceMatcher, so the returned VM object can be used and not cause a conflict
	vm.Spec.Instancetype = nil
	vm.Spec.Preference = nil

	err = response.WriteEntity(vm)
	if err != nil {
		log.Log.Reason(err).Error("Failed to write http response.")
	}
}

func writeValidationErrors(validationErrors []error, response *restful.Response) {
	causes := make([]metav1.StatusCause, 0, len(validationErrors))
	for _, err := range validationErrors {
		causes = append(causes, metav1.StatusCause{
			Message: err.Error(),
		})
	}

	statusError := k8serr.NewBadRequest("Object is not a valid VirtualMachine")
	statusError.ErrStatus.Details = &metav1.StatusDetails{Causes: causes}

	writeError(statusError, response)
}
