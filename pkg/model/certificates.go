package model

func init() {
	Tables = append(Tables, &CertChain{})
}

// CertChain represents one record of the 'certificates' table.
type CertChain struct {
	// The account's unique ID
	ID int64 `xorm:"pk autoincr 'id'"`
	// The Id of the account this certificate belongs to
	AccountID int64 `xorm:"notnull 'account_id'"`
	// The certificate's private key
	PrivateKey []byte `xorm:"'private_key'"`
	// The certificate's public key
	PublicKey []byte `xorm:"'public_key'"`
	// The private certificate
	PrivateCert []byte `xorm:"'private_cert'"`
	// The public certificate
	PublicCert []byte `xorm:"'public_cert'"`
}

// TableName returns the name of the certificates SQL table
func (*CertChain) TableName() string {
	return "certificates"
}
