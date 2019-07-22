package model

func init() {
	Tables = append(Tables, &Partner{})
}

const (
	// TypeSFTP is the string constant used to designate partners which use SFTP
	TypeSFTP = "sftp"
	// TypeHTTP is the string constant used to designate partners which use HTTP
	// TypeHTTP = "http"
	// TypeFTP is the string constant used to designate partners which use FTP
	// TypeFTP = "ftp"
)

// Partner represents one record of the 'partners' table.
type Partner struct {
	// The partner's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"-"`
	// The partner's name
	Name string `xorm:"unique notnull 'name'" json:"name"`
	// The partner's address
	Address string `xorm:"notnull 'address'" json:"address"`
	// The partner's password
	Port uint16 `xorm:"notnull 'port'" json:"port"`
	// The protocol used by the partner
	Type string `xorm:"notnull 'type'" json:"type"`
}

// TableName returns the name of the partners SQL table
func (*Partner) TableName() string {
	return "partners"
}
