package utils

// CloneMap duplicate a map
func CloneMap(content map[string]interface{}) map[string]interface{} {
	clone := make(map[string]interface{})
	for key, value := range content {
		if mapValue, ok := value.(map[string]interface{}); ok {
			clone[key] = CloneMap(mapValue)
		} else {
			clone[key] = value
		}
	}

	return clone
}
