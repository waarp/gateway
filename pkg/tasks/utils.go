package tasks

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func mapToStr(m map[string]string) string {
	args := make([]string, 0, len(m))
	for k, v := range m {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	return "{" + strings.Join(args, ", ") + "}"
}

func buildReplacers(db database.ReadAccess, transCtx *model.TransferContext) (replacersMap, error) {
	replacers := getReplacers()
	replacers.addInfo(transCtx)
	if err := replacers.addEbicsInfo(db, transCtx); err != nil {
		return nil, err
	}

	return replacers, nil
}

func replaceVars(orig string, db database.ReadAccess, transCtx *model.TransferContext) (string, error) {
	replacers, err := buildReplacers(db, transCtx)
	if err != nil {
		return "", err
	}

	for key, f := range replacers {
		reg := regexp.MustCompile(key)
		matches := reg.FindAllString(orig, -1)

		for _, match := range matches {
			rep, err := f(transCtx, match)
			if err != nil {
				return "", err
			}

			bytesRep, err := json.Marshal(rep)
			if err != nil {
				return "", fmt.Errorf("cannot prepare value for replacement: %w", err)
			}

			replacement := string(bytesRep[1 : len(bytesRep)-1])
			orig = strings.ReplaceAll(orig, match, replacement)
		}
	}

	return orig, nil
}
