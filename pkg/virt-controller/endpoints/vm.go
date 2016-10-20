package endpoints

import (
	"encoding/xml"
	"github.com/asaskevich/govalidator"
	"github.com/go-kit/kit/endpoint"
	"github.com/rmohr/go-model"
	"golang.org/x/net/context"
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
		req := request.(VMRequestDTO)
		// TODO here we would fetch the VM from etcd and return 409 if the VM already exists
		vm := api.VM{}
		if errs := model.Copy(&vm, &req.VM); errs != nil {
			return nil, errs[0]
		}
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
	var vm VMRequestDTO
	var body []byte
	// Normally we would directly unmarshal into the struct but we need the raw body again later
	body, err := extractBodyWithLimit(r.Body, DefaultMaxContentLengthBytes)
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(body, &vm); err != nil {
		return nil, err
	}
	if vm.NodeSelector, err = util.ExtractEmbeddedMap(r.Header.Get("Node-Selector")); err != nil {
		return nil, err
	}
	if _, err := govalidator.ValidateStruct(vm); err != nil {
		return nil, err
	}
	vm.RawDomain = body
	return vm, nil
}

type VMRequestDTO struct {
	v1.VM     `valid:"required"`
	XMLName   xml.Name `xml:"domain"`
	RawDomain []byte
}

func MakeVMDeleteEndpoint(svc services.VMService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		name := request.(string)
		// TODO here we would fetch the VM from etcd and return 404 if the VM does not exists
		vm := api.VM{Name: name}
		err := svc.DeleteVM(&vm)
		return nil, err
	}
}
