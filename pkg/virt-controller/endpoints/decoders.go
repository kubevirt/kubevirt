package endpoints

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
)

type PostObject struct {
	Name    string
	Payload interface{}
}

func NameDecodeRequestFunc(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	name, ok := vars["name"]
	if !ok || name == "" {
		// TODO should this be panic? Definitely a 500 and not 400
		return nil, errors.New("Could not find a 'name' variable.")
	}

	if !govalidator.IsAlphanumeric(name) {
		return nil, errors.New("Variable 'name' does not validate as alphanumeric.")
	}
	return name, nil
}

func UUIDDecodeRequestFunc(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	UUID, ok := vars["uuid"]
	if !ok || UUID == "" {
		// TODO should this be panic? Definitely a 500 and not 400
		return nil, errors.New("Could not find a 'uuid' variable.")
	}
	return uuid.FromString(UUID)
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

func NewJsonDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	payloadType := reflect.TypeOf(payloadTypePtr).Elem()
	return func(_ context.Context, r *http.Request) (interface{}, error) {
		obj := reflect.New(payloadType).Interface()
		if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
			return nil, err
		}
		if _, err := govalidator.ValidateStruct(obj); err != nil {
			return nil, err
		}
		return obj, nil
	}
}

func NewJsonPutDecodeRequestFunc(payloadTypePtr interface{}) gokithttp.DecodeRequestFunc {
	jsonDecodeRequestFunc := NewJsonDecodeRequestFunc(payloadTypePtr)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		name, err := NameDecodeRequestFunc(ctx, r)
		if err != nil {
			fmt.Printf("%v", err)
			return nil, err
		}
		payload, err := jsonDecodeRequestFunc(ctx, r)
		if err != nil {
			return nil, err
		}
		return &PostObject{Name: name.(string), Payload: payload}, nil
	}
}
