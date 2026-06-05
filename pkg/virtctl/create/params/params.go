package params

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	paramTag            = "param"
	paramSeparatorCount = 2
)

type NotFoundError struct {
	Name string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s must be specified", e.Name)
}

func (e NotFoundError) Is(target error) bool {
	switch x := target.(type) {
	case NotFoundError:
		return x.Name == e.Name
	case *NotFoundError:
		return x.Name == e.Name
	}
	return false
}

func FlagErr(flagName, format string, a ...any) error {
	return fmt.Errorf("failed to parse \"--%s\" flag: %w", flagName, fmt.Errorf(format, a...))
}

/*
The functions below can be used for complicated flags with multiple parameters.
For example, let's think of the following flag: "--my-flag param1:value1,param2:value2".
To automatically define such a flag, the following struct could be defined:

type MyFlag struct {
	Param1 string `param:"param1"`
	Param2 string `param:"param2"`
}

The functions below use reflection to automatically handle such flags.
*/

// Supported returns the list of supported flags for a parameter struct. This is mainly used to show the user the
// list of supported parameters
func Supported(obj interface{}) string {
	objVal := reflect.ValueOf(obj)
	if objVal.Kind() != reflect.Struct {
		panic(errors.New("passed in interface needs to be a struct"))
	}

	var params []string
	objValType := objVal.Type()
	for i := range objValType.NumField() {
		structField := objValType.Field(i)

		tagVal := structField.Tag.Get(paramTag)
		if tagVal == "" {
			continue
		}

		fieldType := ""
		switch {
		case structField.Type.Kind() == reflect.String:
			fieldType = structField.Type.String()
		case structField.Type == reflect.TypeOf((*uint)(nil)):
			fieldType = structField.Type.Elem().String()
		case structField.Type == reflect.TypeOf(&resource.Quantity{}):
			fieldType = structField.Type.Elem().String()
		case structField.Type.Kind() == reflect.Slice && structField.Type.Elem().Kind() == reflect.String:
			fieldType = structField.Type.String()
		default:
			panic(fmt.Errorf("unsupported struct field \"%s\" with kind \"%s\"", structField.Name, structField.Type.Kind()))
		}

		params = append(params, fmt.Sprintf("%s:%s", tagVal, fieldType))
	}

	return strings.Join(params, ",")
}

// Map assigns the parameter value into the right struct field, which is represented by obj.
// For example, if we use Map("param1", "value1", &myFlag) with MyFlag struct above, Param1 field would be
// assigned with "value1".
// Note that this function may modify the passed in object, even if an error is returned.
// The reason for this is that we don't know the type of the passed in object and if there is a copy
// function for it. It is up to the caller to create a copy of the passed in object if required.
func Map(flagName, paramsStr string, obj interface{}) error {
	params, err := split(paramsStr)
	if err != nil {
		return FlagErr(flagName, "%w", err)
	}

	if err := apply(params, obj); err != nil {
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

// split parses a flag with multiple parameters into a map
func split(paramsStr string) (map[string]string, error) {
	if paramsStr == "" {
		return nil, errors.New("params may not be empty")
	}

	paramsMap := map[string]string{}
	for _, param := range strings.Split(paramsStr, ",") {
		sParam := strings.SplitN(param, ":", paramSeparatorCount)
		if len(sParam) != paramSeparatorCount {
			return nil, fmt.Errorf("params need to have at least one colon: %s", param)
		}
		paramsMap[sParam[0]] = sParam[1]
	}

	return paramsMap, nil
}

// apply assigns the different parameters into obj's corresponding fields
func apply(paramsMap map[string]string, obj interface{}) error {
	objVal := reflect.ValueOf(obj)
	if objVal.Kind() != reflect.Ptr {
		return errors.New("passed in interface needs to be a pointer")
	}

	objValElem := objVal.Elem()
	if objValElem.Kind() != reflect.Struct {
		return errors.New("passed in pointer needs to point to a struct")
	}

	objValElemType := objValElem.Type()
	for i := range objValElemType.NumField() {
		structField := objValElemType.Field(i)

		tagVal := structField.Tag.Get(paramTag)
		if tagVal == "" {
			continue
		}

		paramVal, ok := paramsMap[tagVal]
		if !ok {
			continue
		}

		field := objValElem.Field(i)
		switch {
		case field.Kind() == reflect.String:
			field.SetString(paramVal)
		case field.Type() == reflect.TypeOf((*uint)(nil)):
			u64, err := strconv.ParseUint(paramVal, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse param \"%s\": %w", tagVal, err)
			}
			field.Set(reflect.ValueOf(pointer.P(uint(u64))))
		case field.Type() == reflect.TypeOf(&resource.Quantity{}):
			quantity, err := resource.ParseQuantity(paramVal)
			if err != nil {
				return fmt.Errorf("failed to parse param \"%s\": %w", tagVal, err)
			}
			field.Set(reflect.ValueOf(&quantity))
		case field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String:
			field.Set(reflect.ValueOf(strings.Split(paramVal, ";")))
		default:
			return fmt.Errorf("unsupported struct field \"%s\" with kind \"%s\"", structField.Name, field.Kind())
		}

		delete(paramsMap, tagVal)
	}

	return nil
}

// SplitPrefixedName splits prefixedName with "/" as a separator
func SplitPrefixedName(prefixedName string) (
	prefix string,
	name string,
	err error,
) {
	s := strings.Split(prefixedName, "/")
	switch l := len(s); l {
	case 1:
		name = s[0]
	case paramSeparatorCount:
		prefix = s[0]
		name = s[1]
	default:
		return "", "", fmt.Errorf("invalid count %d of slashes in prefix/name", l-1)
	}

	if name == "" {
		return "", "", errors.New("name cannot be empty")
	}

	return prefix, name, nil
}

func GetParamByName(paramName, paramsStr string) (string, error) {
	paramsMap, err := split(paramsStr)
	if err != nil {
		return "", err
	}

	paramVal, exists := paramsMap[paramName]
	if !exists {
		return "", &NotFoundError{Name: paramName}
	}

	return paramVal, nil
}
