package internal

func SetDefaultValue(m map[string]string, key, defaultValue string) {
	if value := m[key]; value == "" {
		m[key] = defaultValue
	}
}
