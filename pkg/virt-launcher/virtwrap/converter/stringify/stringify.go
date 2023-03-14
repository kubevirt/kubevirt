package stringify

func boolToString(value *bool, defaultPositive bool, positive string, negative string) string {
	toString := func(value bool) string {
		if value {
			return positive
		}
		return negative
	}

	if value == nil {
		return toString(defaultPositive)
	}
	return toString(*value)
}
func BoolToYesNo(value *bool, defaultYes bool) string {
	return boolToString(value, defaultYes, "yes", "no")
}

func BoolToOnOff(value *bool, defaultOn bool) string {
	return boolToString(value, defaultOn, "on", "off")
}
