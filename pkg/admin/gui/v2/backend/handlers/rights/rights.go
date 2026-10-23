// Package rights reads the rights of a user, domain by domain, for the
// screens that show them: the "My account" page (the rights of the signed-in
// user) and the "Users" screen (the rights of every user). One source for
// the domains of the product's permission mask, their labels and their
// rendering as cards.
package rights

import (
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The letters of the product's notation of a domain ("rwd", "r--"), as the
// REST API and the command line write them.
const (
	readLetter   = 'r'
	writeLetter  = 'w'
	deleteLetter = 'd'
	noLetter     = '-'

	labelKeyPrefix   = "rights.domain."
	grantedKeyPrefix = "rights.granted."

	// Stylesheet is the stylesheet of the cards of the rights (List, the
	// checkboxes of the users form), for the screens that show them.
	Stylesheet = "/static_v2/rights/rights.css"
)

// Domain is a domain of the permission mask: what the user may read, write
// or delete in it.
type Domain struct {
	Key string // names the domain in the catalogue keys ("transfers")

	Read, Write, Delete model.PermsMask
	// NoDelete is set on the transfers: the product keeps a bit for their
	// deletion but never checks it (transfers are not deleted by users), so
	// the right is neither offered nor shown as granted.
	NoDelete bool
}

// Domains are the six domains, in the order of the mask.
//
//nolint:gochecknoglobals // reference data of the product
var Domains = []Domain{
	{Key: "transfers", Read: model.PermTransfersRead, Write: model.PermTransfersWrite, NoDelete: true},
	{Key: "servers", Read: model.PermServersRead, Write: model.PermServersWrite, Delete: model.PermServersDelete},
	{Key: "partners", Read: model.PermPartnersRead, Write: model.PermPartnersWrite, Delete: model.PermPartnersDelete},
	{Key: "rules", Read: model.PermRulesRead, Write: model.PermRulesWrite, Delete: model.PermRulesDelete},
	{Key: "users", Read: model.PermUsersRead, Write: model.PermUsersWrite, Delete: model.PermUsersDelete},
	{Key: "admin", Read: model.PermAdminRead, Write: model.PermAdminWrite, Delete: model.PermAdminDelete},
}

// LabelKey is the catalogue key of the label of the domain.
func (d Domain) LabelKey() string { return labelKeyPrefix + d.Key }

// Letters is the notation of the product for the domain ("rwd", "r--").
func (d Domain) Letters(perms model.PermsMask) string {
	letters := []byte{noLetter, noLetter, noLetter}

	if perms.HasPermission(d.Read) {
		letters[0] = readLetter
	}

	if perms.HasPermission(d.Write) {
		letters[1] = writeLetter
	}

	if d.Delete != 0 && perms.HasPermission(d.Delete) {
		letters[2] = deleteLetter
	}

	return string(letters)
}

// Count is the number of rights held in the domain, to order users by it.
func (d Domain) Count(perms model.PermsMask) int {
	letters := d.Letters(perms)

	return len(letters) - strings.Count(letters, string(noLetter))
}

// Row is a domain with what a user may do in it, translated: each right,
// and all of them in words ("Lecture et écriture"), None when they hold none.
type Row struct {
	Key   string
	Label string

	Read, Write, Delete bool
	NoDelete            bool

	Granted string
	None    bool
}

// Rows reads the rights of a user, domain by domain, in the order of the
// product's permission mask.
func Rows(translate gui.Translator, perms model.PermsMask) []Row {
	rows := make([]Row, 0, len(Domains))

	for _, d := range Domains {
		held := strings.ReplaceAll(d.Letters(perms), string(noLetter), "")
		if held == "" {
			held = "none"
		}

		rows = append(rows, Row{
			Key:      d.Key,
			Label:    translate(d.LabelKey()),
			Read:     perms.HasPermission(d.Read),
			Write:    perms.HasPermission(d.Write),
			Delete:   d.Delete != 0 && perms.HasPermission(d.Delete),
			NoDelete: d.NoDelete,
			Granted:  translate(grantedKeyPrefix + held),
			None:     held == "none",
		})
	}

	return rows
}
