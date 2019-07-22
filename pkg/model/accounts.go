package model

func init() {
	Tables = append(Tables, &Account{})
}

// Account represents one record of the 'accounts' table.
type Account struct {
	// The account's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"-"`
	// The account's username
	Username string `xorm:"unique notnull 'username'" json:"username"`
	// The account's password
	Password []byte `xorm:"notnull 'password'" json:"password,omitempty"`
	// The Id of the partner this account belongs to
	PartnerID uint64 `xorm:"notnull 'partner_id'" json:"-"`
}

// BeforeUpdate hashes the account password before updating the record.
func (a *Account) BeforeUpdate() {
	a.Password = hashPassword(a.Password)
}

// BeforeInsert hashes the account password before updating the record.
func (a *Account) BeforeInsert() {
	a.Password = hashPassword(a.Password)
}

// TableName returns the name of the accounts SQL table
func (*Account) TableName() string {
	return "accounts"
}
