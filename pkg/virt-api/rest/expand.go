package rest

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

func (app *SubresourceAPIApp) ExpandSpecRequestHandler(request *restful.Request, response *restful.Response) {
	if request.Request.Body == nil {
		writeError(errors.NewBadRequest("empty request body"), response)
		return
	}

	bodyBytes, err := io.ReadAll(request.Request.Body)
	if err != nil {
		writeError(errors.NewBadRequest(err.Error()), response)
		return
	}

	rawObj := map[string]interface{}{}
	err = json.Unmarshal(bodyBytes, &rawObj)
	if err != nil {
		writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
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
		writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
		return
	}

	requestNamespace := request.PathParameter("namespace")
	if requestNamespace == "" {
		writeError(errors.NewBadRequest("The request namespace must not be empty"), response)
		return
	}
	if vm.Namespace != "" && vm.Namespace != requestNamespace {
		writeError(errors.NewBadRequest(
			fmt.Sprintf("VM namespace must be empty or %s", requestNamespace)),
			response,
		)
		return
	}
	vm.Namespace = request.PathParameter("namespace")

	expandSpecResponse(vm, app.instancetypeMethods, func(err error) *errors.StatusError {
		return errors.NewBadRequest(err.Error())
	}, response)
}

func (app *SubresourceAPIApp) ExpandSpecVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}

	expandSpecResponse(vm, app.instancetypeMethods, errors.NewInternalError, response)
}

func expandSpecResponse(vm *v1.VirtualMachine, instancetypeMethods instancetype.Methods, errorFunc func(error) *errors.StatusError, response *restful.Response) {
	instancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
	if err != nil {
		writeError(errorFunc(err), response)
		return
	}
	preferenceSpec, err := instancetypeMethods.FindPreferenceSpec(vm)
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

	conflicts := instancetypeMethods.ApplyToVmi(field.NewPath("spec", "template", "spec"), instancetypeSpec, preferenceSpec, &vm.Spec.Template.Spec, &vm.Spec.Template.ObjectMeta)
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

	statusError := errors.NewBadRequest("Object is not a valid VirtualMachine")
	statusError.ErrStatus.Details = &metav1.StatusDetails{Causes: causes}

	writeError(statusError, response)
}
