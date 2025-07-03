package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getCredential(db database.ReadAccess, owner model.CredOwnerTable, name string,
) (*model.Credential, error) {
	var cred model.Credential

	return &cred, db.Get(&cred, "name=?", name).And(owner.GetCredCond()).Run()
}

func getCredentialByID(db database.ReadAccess, owner model.CredOwnerTable, id int64,
) (*model.Credential, error) {
	var cred model.Credential

	return &cred, db.Get(&cred, "id=?", id).And(owner.GetCredCond()).Run()
}

func GetCredentialsLike(db *database.DB, owner model.CredOwnerTable, prefix string) ([]*model.Credential, error) {
	const limit = 5
	var cred model.Credentials

	return cred, db.Select(&cred).Where(owner.GetCredCond()).Where("name LIKE ?", prefix+"%").
		OrderBy("name", true).Limit(limit, 0).Run()
}

func listCredentials(db database.ReadAccess, owner model.CredOwnerTable,
	orderByCol string, orderByAsc bool, limit, offset int, types ...string,
) ([]*model.Credential, error) {
	var credentials model.Credentials
	query := db.Select(&credentials).Where(owner.GetCredCond()).
		OrderBy(orderByCol, orderByAsc).Limit(limit, offset)

	for _, t := range types {
		query = query.Where("type=?", t)
	}

	return credentials, query.Run()
}

func InsertCredential(db database.Access, credential *model.Credential) error {
	return db.Insert(credential).Run()
}

func UpdateCredential(db database.Access, credential *model.Credential) error {
	return db.Update(credential).Run()
}

func DeleteCredential(db database.Access, credential *model.Credential) error {
	return db.Delete(credential).Run()
}
