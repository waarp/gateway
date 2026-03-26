package backup

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func importEbicsPayloadProfiles(
	logger *log.Logger,
	db database.Access,
	profiles []file.EbicsPayloadProfile,
	reset bool,
) error {
	if reset {
		if err := db.DeleteAll(&model.EbicsPayloadProfile{}).Owner().Run(); err != nil {
			return fmt.Errorf("failed to purge EBICS payload profiles: %w", err)
		}
	}

	for i := range profiles {
		var (
			dbProfile model.EbicsPayloadProfile
			isNew     bool
		)

		if err := db.Get(&dbProfile, "name=?", profiles[i].Name).Owner().Run(); database.IsNotFound(err) {
			isNew = true
		} else if err != nil {
			return fmt.Errorf("failed to retrieve EBICS payload profile %q: %w", profiles[i].Name, err)
		}

		dbProfile.Name = profiles[i].Name
		dbProfile.Label = profiles[i].Label
		dbProfile.Description = profiles[i].Description
		dbProfile.OrderType = profiles[i].OrderType
		dbProfile.Direction = profiles[i].Direction
		dbProfile.ServiceName = profiles[i].ServiceName
		dbProfile.ServiceOption = profiles[i].ServiceOption
		dbProfile.Scope = profiles[i].Scope
		dbProfile.MsgName = profiles[i].MsgName
		dbProfile.ContainerType = profiles[i].ContainerType
		dbProfile.DefaultTargetDirectory = profiles[i].DefaultTargetDirectory
		dbProfile.RequiresDeclaredAmount = profiles[i].RequiresDeclaredAmount
		dbProfile.DefaultCurrency = profiles[i].DefaultCurrency
		dbProfile.AllowedExtensionsList = profiles[i].AllowedExtensions
		dbProfile.FilenamePattern = profiles[i].FilenamePattern
		dbProfile.StrictContractCheck = profiles[i].StrictContractCheck
		dbProfile.IsEnabled = profiles[i].IsEnabled
		dbProfile.MetadataMap = profiles[i].Metadata
		dbProfile.DefaultRuleID = sql.NullInt64{}

		if profiles[i].DefaultRule != "" {
			ruleID, err := getDefaultRuleIDForEbicsPayloadProfile(db, &profiles[i])
			if err != nil {
				return err
			}

			dbProfile.DefaultRuleID = ruleID
		}

		var dbErr error
		if isNew {
			logger.Infof("Inserting new EBICS payload profile %q", dbProfile.Name)
			dbErr = db.Insert(&dbProfile).Run()
		} else {
			logger.Infof("Updating existing EBICS payload profile %q", dbProfile.Name)
			dbErr = db.Update(&dbProfile).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import EBICS payload profile %q: %w", dbProfile.Name, dbErr)
		}
	}

	return nil
}

func getDefaultRuleIDForEbicsPayloadProfile(
	db database.Access,
	profile *file.EbicsPayloadProfile,
) (sql.NullInt64, error) {
	var rules model.Rules
	if err := db.Select(&rules).Where("name=?", profile.DefaultRule).Owner().Run(); err != nil {
		return sql.NullInt64{}, fmt.Errorf(
			"failed to retrieve default rule %q of EBICS payload profile %q: %w",
			profile.DefaultRule, profile.Name, err,
		)
	}

	if len(rules) == 0 {
		return sql.NullInt64{}, database.NewValidationErrorf(
			"the default rule %q of EBICS payload profile %q does not exist",
			profile.DefaultRule, profile.Name,
		)
	}

	if len(rules) == 1 {
		return utils.NewNullInt64(rules[0].ID), nil
	}

	for i := range rules {
		if profile.Direction == "UPLOAD" && rules[i].IsSend {
			return utils.NewNullInt64(rules[i].ID), nil
		}

		if profile.Direction == "DOWNLOAD" && !rules[i].IsSend {
			return utils.NewNullInt64(rules[i].ID), nil
		}
	}

	return sql.NullInt64{}, database.NewValidationErrorf(
		"the default rule %q of EBICS payload profile %q is ambiguous",
		profile.DefaultRule, profile.Name,
	)
}
