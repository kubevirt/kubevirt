package endpoints

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/satori/go.uuid"
	"kubevirt/core/pkg/virt-controller/entities"
	"kubevirt/core/pkg/virt-controller/services"
)

const DefaultMaxContentLengthBytes = 3 << 20

func MakeRawDomainEndpoint(svc services.VMService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VMRequestDTO)
		UUID, err := uuid.FromString(req.UUID)
		vm := entities.VM{Name: req.Name, UUID: UUID}
		if err = svc.StartVMRaw(&vm, req.RawDomain); err != nil {
			return nil, err
		}
		return VMResponseDTO{UUID: req.UUID}, nil
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
	body, err := checkAndExtractBody(r.Body, DefaultMaxContentLengthBytes)
	if err != nil {
		return nil, err
	}

	if err := xml.Unmarshal(body, &vm); err != nil {
		return nil, err
	}
	if vm.Name == "" {
		return nil, errors.New("Name is missing")
	}

	if vm.UUID == "" {
		return nil, errors.New("UUID is missing")
	}

	if _, err := uuid.FromString(vm.UUID); err != nil {
		return nil, errors.New("UUID is invalid")
	}
	vm.RawDomain = body
	return vm, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(response)
}

type VMRequestDTO struct {
	XMLName   xml.Name `xml:"domain"`
	Name      string   `xml:"name"`
	UUID      string   `xml:"uuid"`
	RawDomain []byte
}

type VMResponseDTO struct {
	UUID string `json:"uuid"`
}

// TODO make this usable as a wrapping handler func or replace with http.MaxBytesReader
func checkAndExtractBody(http_body io.ReadCloser, maxContentLength int64) ([]byte, error) {
	body, err := ioutil.ReadAll(io.LimitReader(http_body, maxContentLength+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxContentLength {
		return nil, errors.New("http: POST too large")
	}
	return body, nil
}
