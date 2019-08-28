package model

import "fmt"

func init() {
	Tables = append(Tables, &CertChain{})
}

// CertChain represents one record of the 'certificates' table.
type CertChain struct {
	// The account's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`
	// The Id of the account this certificate belongs to
	AccountID uint64 `xorm:"unique(cert) notnull 'account_id'" json:"accountID"`
	// The certificate chain's name
	Name string `xorm:"unique(cert) notnull 'name'" json:"name"`
	// The account's private key
	PrivateKey []byte `xorm:"'private_key'" json:"privateKey"`
	// The account's public key
	PublicKey []byte `xorm:"'public_key'" json:"publicKey"`
	// The public key certificate
	Cert []byte `xorm:"'cert'" json:"cert"`
}

// Validate checks if the certificate entry can be inserted in the database.
func (c *CertChain) Validate(exists func(interface{}) (bool, error)) error {
	if c.Name == "" {
		return ErrInvalid{msg: "The certificate's name cannot be empty"}
	}
	if c.AccountID == 0 {
		return ErrInvalid{msg: "The certificate's accountID cannot be empty"}
	}

	parent := &Account{ID: c.AccountID}
	if ok, err := exists(parent); err != nil {
		return err
	} else if !ok {
		return ErrInvalid{msg: fmt.Sprintf("The account n°%v does not exist",
			parent.ID)}
	}

	test := &CertChain{
		AccountID: c.AccountID,
		Name:      c.Name,
	}
	if ok, err := exists(test); err != nil {
		return err
	} else if ok {
		return ErrInvalid{msg: fmt.Sprintf("A certificate named '%s' already exists",
			c.Name)}
	}

	return nil
}

// TableName returns the name of the certificates SQL table
func (*CertChain) TableName() string {
	return "certificates"
}
