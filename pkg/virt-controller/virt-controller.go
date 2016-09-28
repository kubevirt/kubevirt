package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"golang.org/x/net/context"
	"log"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

type VmService interface {
	StartVmRaw(string) error
}

type vmService struct{}

func (vmService) StartVmRaw(name string) error {
	log.Printf("VM name is %s", name)
	return nil
}

type Handlers struct {
	rawDomainHandler http.Handler
}

func main() {
	ctx := context.Background()
	svc := vmService{}

	endpoints := Handlers{
		rawDomainHandler: makeRawDomainHandler(ctx, makeRawDomainEndpoint(svc)),
	}

	http.Handle("/", defineRoutes(&endpoints))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func defineRoutes(endpoints *Handlers) http.Handler {
	router := mux.NewRouter()
	restV1Route := router.PathPrefix("/api/v1/").Subrouter()
	restV1Route.Methods("POST").Path("/domain/raw").Headers("Content-Type", "application/xml").Handler(endpoints.rawDomainHandler)
	return router
}

func makeRawDomainEndpoint(svc VmService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VmRequestDto)
		err := svc.StartVmRaw(req.Name)
		if err != nil {
			return ResponseDto{Err: err.Error()}, nil
		}
		return ResponseDto{Err: ""}, nil
	}
}

func makeRawDomainHandler(ctx context.Context, endpoint endpoint.Endpoint) http.Handler {
	return kithttp.NewServer(
		ctx,
		endpoint,
		decodeRawDomainRequest,
		encodeResponse,
	)
}

func decodeRawDomainRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var vm VmRequestDto
	if err := xml.NewDecoder(r.Body).Decode(&vm); err != nil {
		return nil, err
	}
	if vm.Name == "" {
		return nil, errors.New("VM name is missing")
	}
	return vm, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

type VmRequestDto struct {
	XMLName   xml.Name `xml:"domain"`
	Name      string   `xml:"name"`
	RawDomain []byte
}

type ResponseDto struct {
	Err string `json:"err,omitempty"`
}
