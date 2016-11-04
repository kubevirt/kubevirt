package endpoints

import (
	"encoding/xml"
	"github.com/asaskevich/govalidator"
	"github.com/go-kit/kit/endpoint"
	"github.com/rmohr/go-model"
	"golang.org/x/net/context"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/types"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/api/v1"
	"kubevirt/core/pkg/util"
	"kubevirt/core/pkg/virt-controller/services"
	"net/http"
)

func MakePrepareMigrationHandler(svc services.VMService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		postObject := request.(*PostObject)
		vm := api.VM{}
		if errs := model.Copy(&vm, postObject.Payload); errs != nil {
			return nil, errs[0]
		}
		return vm, svc.PrepareMigration(&vm)
	}
}

func MakeRawDomainEndpoint(svc services.VMService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(v1.Domain)
		vm := api.VM{}
		vm.Name = req.Name
		vm.UID = types.UID(req.UUID)
		vm.Spec.NodeSelector = req.NodeSelector

		if err := svc.StartVMRaw(&vm, req.RawDomain); err != nil {
			return nil, err //TODO with the kubelet in place it is hard to tell what went wrong
		}
		// TODO add error aggregation
		v1VM := v1.VM{}
		if errs := model.Copy(&v1VM, &vm); errs != nil {
			return nil, errs[0]
		}
		return v1VM, nil
	}
}

func DecodeRawDomainRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var domain v1.Domain
	var body []byte
	// Normally we would directly unmarshal into the struct but we need the raw body again later
	body, err := extractBodyWithLimit(r.Body, DefaultMaxContentLengthBytes)
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(body, &domain); err != nil {
		return nil, err
	}
	if domain.NodeSelector, err = util.ExtractEmbeddedMap(r.Header.Get("Node-Selector")); err != nil {
		return nil, err
	}
	if _, err := govalidator.ValidateStruct(domain); err != nil {
		return nil, err
	}
	domain.RawDomain = body
	return domain, nil
}

func MakeVMDeleteEndpoint(svc services.VMService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		name := request.(string)
		// TODO here we would fetch the VM from etcd and return 404 if the VM does not exists
		vm := api.VM{ObjectMeta: kubeapi.ObjectMeta{Name: name}}
		err := svc.DeleteVM(&vm)
		return vm, err
	}
}
