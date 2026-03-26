package backup

import (
	"encoding/json"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportEbicsPayloadProfiles(
	logger *log.Logger,
	db database.ReadAccess,
) ([]file.EbicsPayloadProfile, error) {
	var dbProfiles model.EbicsPayloadProfiles
	if err := db.Select(&dbProfiles).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS payload profiles: %w", err)
	}

	profiles := make([]file.EbicsPayloadProfile, len(dbProfiles))

	for i := range dbProfiles {
		logger.Infof("Exporting EBICS payload profile %q", dbProfiles[i].Name)

		allowedExtensions, err := decodeStringSliceJSON(dbProfiles[i].AllowedExtensions)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to decode allowed extensions of EBICS payload profile %q: %w",
				dbProfiles[i].Name, err,
			)
		}

		metadata, err := decodeStringMapJSON(dbProfiles[i].Metadata)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to decode metadata of EBICS payload profile %q: %w",
				dbProfiles[i].Name, err,
			)
		}

		profile := file.EbicsPayloadProfile{
			Name:                   dbProfiles[i].Name,
			Label:                  dbProfiles[i].Label,
			Description:            dbProfiles[i].Description,
			OrderType:              dbProfiles[i].OrderType,
			Direction:              dbProfiles[i].Direction,
			ServiceName:            dbProfiles[i].ServiceName,
			ServiceOption:          dbProfiles[i].ServiceOption,
			Scope:                  dbProfiles[i].Scope,
			MsgName:                dbProfiles[i].MsgName,
			ContainerType:          dbProfiles[i].ContainerType,
			DefaultTargetDirectory: dbProfiles[i].DefaultTargetDirectory,
			RequiresDeclaredAmount: dbProfiles[i].RequiresDeclaredAmount,
			DefaultCurrency:        dbProfiles[i].DefaultCurrency,
			AllowedExtensions:      allowedExtensions,
			FilenamePattern:        dbProfiles[i].FilenamePattern,
			StrictContractCheck:    dbProfiles[i].StrictContractCheck,
			IsEnabled:              dbProfiles[i].IsEnabled,
			Metadata:               metadata,
		}

		if dbProfiles[i].DefaultRuleID.Valid {
			var rule model.Rule
			if getErr := db.Get(&rule, "id=?", dbProfiles[i].DefaultRuleID.Int64).Run(); getErr != nil {
				return nil, fmt.Errorf(
					"failed to retrieve default rule of EBICS payload profile %q: %w",
					dbProfiles[i].Name, getErr,
				)
			}

			profile.DefaultRule = rule.Name
		}

		profiles[i] = profile
	}

	return profiles, nil
}

func decodeStringSliceJSON(raw string) ([]string, error) {
	if raw == "" {
		return []string{}, nil
	}

	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("failed to unmarshal string slice JSON: %w", err)
	}

	return values, nil
}

func decodeStringMapJSON(raw string) (map[string]any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}

	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("failed to unmarshal string map JSON: %w", err)
	}

	if values == nil {
		values = map[string]any{}
	}

	return values, nil
}
