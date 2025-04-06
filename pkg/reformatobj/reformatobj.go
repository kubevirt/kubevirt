package reformatobj

import (
	"encoding/json"
	"fmt"
)

// ReformatObj reformats client objects to solve the quantity bug.
// The bug is happening when setting a quantity field without quantity
// type.
// for example
//
//	limit:
//	  cpu: "1.5"
//
// In this case, the client actually set a formatted value to the resource
// in K8s cluster, while HCO keep use the original un-typed quantity value.
// That causes an endless loop of updates, because HCO compares the values
// and finds out they are different.
func ReformatObj[T any](obj *T) (*T, error) {
	bts, err := json.Marshal(obj)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal %T: %w", obj, err)
	}

	var obj2 T
	err = json.Unmarshal(bts, &obj2)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %T: %w", obj, err)
	}

	return &obj2, nil
}
