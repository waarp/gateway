package model

func init() {
	Tables = append(Tables, &CertChain{})
}

// CertChain represents one record of the 'certificates' table.
type CertChain struct {
	// The account's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"-"`
	// The Id of the account this certificate belongs to
	AccountID uint64 `xorm:"notnull 'account_id'" json:"-"`
	// The certificate chain's name
	Name string `xorm:"unique notnull 'name'" json:"name"`
	// The certificate's private key
	PrivateKey []byte `xorm:"'private_key'" json:"privateKey"`
	// The certificate's public key
	PublicKey []byte `xorm:"'public_key'" json:"publicKey"`
	// The private certificate
	PrivateCert []byte `xorm:"'private_cert'" json:"privateCert"`
	// The public certificate
	PublicCert []byte `xorm:"'public_cert'" json:"publicCert"`
}

// TableName returns the name of the certificates SQL table
func (*CertChain) TableName() string {
	return "certificates"
}
