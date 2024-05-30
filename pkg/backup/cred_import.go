package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func credentialsImport(logger *log.Logger, db database.Access, list []file.Credential,
	owner model.CredOwnerTable, protocol string,
) error {
	for _, src := range list {
		// Create model with basic info to check existence
		var credential model.Credential

		// Check if crypto exists
		var exist bool

		dbErr := db.Get(&credential, "name=?", src.Name).And(
			owner.GetCredCond()).Run()
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

		// Create/Update
		if exist {
			logger.Info("Update the credential %s", credential.Name)
			dbErr = db.Update(&credential).Run()
		} else {
			logger.Info("Create the credential %s", credential.Name)
			dbErr = db.Insert(&credential).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to create/update credential %q: %w", src.Name, dbErr)
		}
	}

	return nil
}
