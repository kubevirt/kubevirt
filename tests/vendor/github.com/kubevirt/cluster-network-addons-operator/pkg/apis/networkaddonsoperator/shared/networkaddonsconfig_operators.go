package shared

import "reflect"

func (s NetworkAddonsConfigStatus) DeepEqual(statusToCompare NetworkAddonsConfigStatus) bool {
	return reflect.DeepEqual(s, statusToCompare)
}
