package appstats

import (
	"reflect"

	"github.com/golang/protobuf/proto"
)

const (
	cost_Write = 10
	cost_Read  = 7
	cost_Small = 1
)

// todo: implement read and small ops costs

func getCost(p proto.Message) int64 {
	v := reflect.ValueOf(p)
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return 0
	}

	var cost int64
	cost += extractCost(v)

	return cost
}

func extractCost(v reflect.Value) int64 {
	v = v.FieldByName("Cost")
	if v.Kind() != reflect.Ptr {
		return 0
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return 0
	}

	var cost int64

	extract := func(name string) int64 {
		w := v.FieldByName(name)
		if w.Kind() != reflect.Ptr {
			return 0
		}
		w = w.Elem()
		switch w.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			return w.Int()
		}

		return 0
	}

	cost += extract("IndexWrites")
	cost += extract("EntityWrites")

	return cost * cost_Write
}
