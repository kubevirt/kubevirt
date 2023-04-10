package params

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	paramTag = "param"
)

func FlagErr(flagName, format string, a ...any) error {
	return fmt.Errorf("failed to parse \"--%s\" flag: %w", flagName, fmt.Errorf(format, a...))
}

func Supported(obj interface{}) string {
	objVal := reflect.ValueOf(obj)
	if objVal.Kind() != reflect.Struct {
		panic("passed in interface needs to be a struct")
	}

	var params []string
	objValType := objVal.Type()
	for i := 0; i < objValType.NumField(); i++ {
		structField := objValType.Field(i)

		k := structField.Tag.Get(paramTag)
		if k == "" {
			continue
		}

		t := ""
		switch {
		case structField.Type.Kind() == reflect.String:
			t = structField.Type.String()
		case structField.Type == reflect.TypeOf(&resource.Quantity{}):
			t = structField.Type.Elem().String()
		default:
			panic(fmt.Errorf("unsupported struct field \"%s\" with kind \"%s\"", structField.Name, structField.Type.Kind()))
		}

		params = append(params, fmt.Sprintf("%s:%s", k, t))
	}

	return strings.Join(params, ",")
}

func Map(flagName, paramsStr string, obj interface{}) error {
	params, err := Split(paramsStr)
	if err != nil {
		return FlagErr(flagName, "%w", err)
	}

	err = Apply(params, obj)
	if err != nil {
		return FlagErr(flagName, "%w", err)
	}

	if len(params) > 0 {
		var unknown []string
		for k, v := range params {
			unknown = append(unknown, fmt.Sprintf("%s:%s", k, v))
		}
		return FlagErr(flagName, "unknown param(s): %s", strings.Join(unknown, ","))
	}

	return nil
}

func Split(paramsStr string) (map[string]string, error) {
	if paramsStr == "" {
		return nil, errors.New("params may not be empty")
	}

	paramsMap := map[string]string{}
	s := strings.Split(paramsStr, ",")
	for _, param := range s {
		sParam := strings.SplitN(param, ":", 2)
		if len(sParam) != 2 {
			return nil, fmt.Errorf("params need to have at least one colon: %s", param)
		}
		paramsMap[sParam[0]] = sParam[1]
	}

	return paramsMap, nil
}

func Apply(paramsMap map[string]string, obj interface{}) error {
	objVal := reflect.ValueOf(obj)
	if objVal.Kind() != reflect.Ptr {
		panic("passed in interface needs to be a pointer")
	}

	objValElem := objVal.Elem()
	if objValElem.Kind() != reflect.Struct {
		panic("passed in pointer needs to point to a struct")
	}

	objValElemType := objValElem.Type()
	for i := 0; i < objValElemType.NumField(); i++ {
		structField := objValElemType.Field(i)

		k := structField.Tag.Get(paramTag)
		if k == "" {
			continue
		}

		v, ok := paramsMap[k]
		if !ok {
			continue
		}

		field := objValElem.Field(i)
		switch {
		case field.Kind() == reflect.String:
			field.SetString(v)
		case field.Type() == reflect.TypeOf(&resource.Quantity{}):
			quantity, err := resource.ParseQuantity(v)
			if err != nil {
				return fmt.Errorf("failed to parse param \"%s\": %w", k, err)
			}
			field.Set(reflect.ValueOf(&quantity))
		default:
			panic(fmt.Errorf("unsupported struct field \"%s\" with kind \"%s\"", structField.Name, field.Kind()))
		}

		delete(paramsMap, k)
	}

	return nil
}

func SplitPrefixedName(prefixedName string) (prefix string, name string, err error) {
	s := strings.Split(prefixedName, "/")

	switch l := len(s); l {
	case 1:
		name = s[0]
	case 2:
		prefix = s[0]
		name = s[1]
	default:
		return "", "", fmt.Errorf("invalid count %d of slashes in prefix/name", l)
	}

	if name == "" {
		return "", "", errors.New("name cannot be empty")
	}

	return
}
