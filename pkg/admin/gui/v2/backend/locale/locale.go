package locale

import (
	"fmt"
	"io/fs"
	"net/http"

	"gopkg.in/yaml.v3"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
)

type Dictionary map[string]string

type LocalizationData map[string]Dictionary

func MakeLocalText(language string, locData LocalizationData,
) map[string]string {
	dictionary := make(map[string]string)

	for key, value := range locData {
		dictionary[key] = value[language]
	}

	return dictionary
}

func ParseLocalizationFile(filename string) LocalizationData {
	rawData, opErr := fs.ReadFile(webfs.WebFS, filename)
	if opErr != nil {
		panic(fmt.Sprintf("failed to read sidebar localization file: %v", opErr))
	}

	var localizationData LocalizationData

	if err := yaml.Unmarshal(rawData, &localizationData); err != nil {
		panic(fmt.Sprintf("failed to unmarshal localization file %q: %v", filename, err))
	}

	return localizationData
}

func GetLanguage(r *http.Request) string {
	//nolint:forcetypeassert //assertion always succeeds
	return r.Context().Value(constants.ContextLanguageKey).(string)
}
