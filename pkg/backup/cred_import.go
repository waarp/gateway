package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
)

func credentialsImport(logger *log.Logger, db database.Access, list []file.Credential,
	owner model.CredOwnerTable,
) error {
	protocol, protErr := owner.GetProtocol(db)
	if protErr != nil {
		return fmt.Errorf("failed to retrieve %s protocol: %w", owner.Appellation(), protErr)
	}

	for _, src := range list {
		// Create model with basic info to check existence
		var credential model.Credential

		// Check if crypto exists
		var exist bool

		dbErr := db.Get(&credential, "name=?", src.Name).And(owner.GetCredCond()).Run()
		if dbErr == nil {
			exist = true
		} else if !database.IsNotFound(dbErr) {
			return fmt.Errorf("failed to check credential existence: %w", dbErr)
		}

		// Populate
		credential.Name = src.Name
		credential.Type = src.Type
		credential.Value = src.Value
		credential.Value2 = src.Value2
		owner.SetCredOwner(&credential)

		// Check for unique credentials
		var handler authentication.Handler
		if credential.IsInternal() {
			handler = authentication.GetInternalAuthHandler(credential.Type, protocol)
		} else {
			handler = authentication.GetExternalAuthMethod(credential.Type, protocol)
		}

		if handler.CanOnlyHaveOne() {
			if err := db.DeleteAll(&model.Credential{}).Where(owner.GetCredCond()).
				Where("type=?", credential.Type).Run(); err != nil {
				return fmt.Errorf("failed to delete old %s credential: %w", credential.Type, err)
			}

			exist = false
		}

		// Create/Update
		if exist {
			logger.Infof("Update the credential %q", credential.Name)
			dbErr = db.Update(&credential).Run()
		} else {
			logger.Infof("Create the credential %q", credential.Name)
			dbErr = db.Insert(&credential).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to create/update credential %q: %w", src.Name, dbErr)
		}
	}

	return nil
}
