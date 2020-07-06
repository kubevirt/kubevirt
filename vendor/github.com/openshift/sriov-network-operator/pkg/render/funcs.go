package render

// Functions available for all templates

// getOr returns the value of m[key] if it exists, fallback otherwise.
// As a special case, it also returns fallback if the value of m[key] is
// the empty string
func getOr(m map[string]interface{}, key, fallback string) interface{} {
	val, ok := m[key]
	if !ok {
		return fallback
	}

	s, ok := val.(string)
	if ok && s == "" {
		return fallback
	}

	return val
}

// isSet returns the value of m[key] if key exists, otherwise false
// Different from getOr because it will return zero values.
func isSet(m map[string]interface{}, key string) interface{} {
	val, ok := m[key]
	if !ok {
		return false
	}
	return val
}
