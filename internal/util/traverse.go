package util

func TraverseJSON(
	data any,
	keys any,
) any {
	var keySlice []string
	switch k := keys.(type) {
	case string:
		keySlice = []string{k}
	case []string:
		keySlice = k
	default:
		// logger.L.Warnf("unsupported keys type: %T", keys)
		return nil
	}

	return traverseObject(data, keySlice)
}

func traverseObject(data any, keys []string) any {
	if len(keys) == 0 {
		return data
	}

	key := keys[0]
	remainingKeys := keys[1:]

	switch d := data.(type) {
	case map[string]any:
		if value, exists := d[key]; exists {
			return traverseObject(value, remainingKeys)
		}

		for _, value := range d {
			result := traverseObject(value, keys)
			if result != nil {
				return result
			}
		}
	case []any:
		for _, item := range d {
			result := traverseObject(item, keys)
			if result != nil {
				return result
			}
		}
	}
	return nil
}
