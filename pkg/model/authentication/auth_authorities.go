package authentication

//nolint:gochecknoglobals //global var is required here
var authorityTypes map[string]Authority

func AddAuthorityType(typ string, authority Authority) {
	if authorityTypes == nil {
		authorityTypes = map[string]Authority{}
	}

	authorityTypes[typ] = authority
}

func GetAuthorityHandler(typ string) Authority {
	return authorityTypes[typ]
}

type Authority interface {
	Validate(identity string) error
}
