package endpoints

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/satori/go.uuid"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/api/v1"
	"kubevirt/core/pkg/middleware"
	"kubevirt/core/pkg/virt-controller/services"
)

const DefaultMaxContentLengthBytes = 3 << 20

func MakeRawDomainEndpoint(svc services.VMService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VMRequestDTO)
		UUID := uuid.FromStringOrNil(req.UUID)
		vm := api.VM{Name: req.Name, UUID: UUID}
		if err := svc.StartVMRaw(&vm, req.RawDomain); err != nil {
			return nil, err //TODO with the kubelet in place it is hard to tell what went wrong
		}
		return v1.VM{UUID: vm.UUID.String()}, nil
	}
}

func MakeRawDomainHandler(ctx context.Context, endpoint endpoint.Endpoint) http.Handler {
	return kithttp.NewServer(
		ctx,
		endpoint,
		decodeRawDomainRequest,
		encodeResponse,
	)
}

func decodeRawDomainRequest(_ context.Context, r *http.Request) (interface{}, error) {
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
	if _, err := govalidator.ValidateStruct(vm); err != nil {
		return nil, err
	}
	vm.RawDomain = body
	return vm, nil
}

func encodeResponse(context context.Context, w http.ResponseWriter, response interface{}) error {
	if _, ok := response.(middleware.AppError); ok != false {
		return encodeApplicationErrors(context, w, response)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(response)
}

func encodeApplicationErrors(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "text/plain")
	switch t := response.(type) {
	// More specific AppErrors  like 404 must be handled before the AppError case
	case middleware.AppError:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(t.Cause().Error()))
	default:
		w.WriteHeader(http.StatusInternalServerError)
		// TODO log the error but don't send it along
		w.Write([]byte("Error handling failed, that should never happen."))
	}
	return json.NewEncoder(w).Encode(response)
}

type VMRequestDTO struct {
	v1.VM     `valid:"required"`
	XMLName   xml.Name `xml:"domain"`
	RawDomain []byte
}

func extractBodyWithLimit(http_body io.ReadCloser, maxContentLength int64) ([]byte, error) {
	body, err := ioutil.ReadAll(io.LimitReader(http_body, maxContentLength+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxContentLength {
		return nil, errors.New("http: POST too large")
	}
	return body, nil
}
