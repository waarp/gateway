package model

// Transfer represents one record of the 'transfers' table.
type Transfer struct {
	IsGet       bool          `json:"isGet"`
	Remote      RemoteAgent   `json:"remote"`
	Account     RemoteAccount `json:"account"`
	Source      string        `json:"source"`
	Destination string        `json:"destination"`
}
