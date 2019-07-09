package model

func init() {
	Tables = append(Tables, &Account{})
}

// Account represents one record of the 'accounts' table.
type Account struct {
	// The account's unique ID
	ID int64 `xorm:"pk autoincr 'id'"`
	// The account's username
	Username string `xorm:"unique notnull 'username'"`
	// The account's password
	Password []byte `xorm:"notnull 'password'"`
	// The Id of the partner this account belongs to
	PartnerID int64 `xorm:"notnull 'partner_id'"`
}

// TableName returns the name of the accounts SQL table
func (*Account) TableName() string {
	return "accounts"
}
