package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"golang.org/x/net/context"
	"net/http"

	"flag"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"os"
	"strconv"
)

const DefaultMaxContentLengthBytes = 3 << 20

type VMService interface {
	StartVMRaw(uuid.UUID, string, []byte) error
}

type vmService struct {
	logger levels.Levels
}

func (v *vmService) StartVMRaw(UUID uuid.UUID, name string, rawXML []byte) error {
	v.logger.Info().Log("action", "StartVMRaw", "object", "VM", "UUID", UUID, "name", name)
	return nil
}

func makeVMService(logger log.Logger) *vmService {
	svc := vmService{logger: levels.New(logger).With("component", "VMService")}
	return &svc
}

type Handlers struct {
	RawDomainHandler http.Handler
}

func main() {

	host := flag.String("address", "0.0.0.0", "Address to bind to")
	port := flag.Int("port", 8080, "Port to listen on")

	logger := log.NewLogfmtLogger(os.Stderr)
	flag.Parse()
	ctx := context.Background()
	svc := makeVMService(logger)

	endpoints := Handlers{
		RawDomainHandler: makeRawDomainHandler(ctx, makeRawDomainEndpoint(svc)),
	}

	http.Handle("/", defineRoutes(&endpoints))
	httpLogger := levels.New(logger).With("component", "http")
	httpLogger.Info().Log("action", "listening", "interface", *host, "port", *port)
	http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil)
}

func defineRoutes(endpoints *Handlers) http.Handler {
	router := mux.NewRouter()
	restV1Route := router.PathPrefix("/api/v1/").Subrouter()
	restV1Route.Methods("POST").Path("/domain/raw").Headers("Content-Type", "application/xml").Handler(endpoints.RawDomainHandler)
	return router
}

func makeRawDomainEndpoint(svc VMService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VMRequestDTO)
		UUID, err := uuid.FromString(req.UUID)
		if err = svc.StartVMRaw(UUID, req.Name, req.RawDomain); err != nil {
			return nil, err
		}
		return VMResponseDTO{UUID: req.UUID}, nil
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
		return nil, errors.New(".name is missing")
	}

	if vm.UUID == "" {
		return nil, errors.New(".uuid name is missing")
	}

	if _, err := uuid.FromString(vm.UUID); err != nil {
		return nil, errors.New(".uuid is invalid")
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
