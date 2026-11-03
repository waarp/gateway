package dashboard

import (
	"fmt"
	"sort"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/certinfo"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
)

// The owners of a certificate, as the strip names them.
const (
	ownerServer    = "server"
	ownerPartner   = "partner"
	ownerAccount   = "account"
	ownerAuthority = "authority"
)

// certSummary sums up the certificates of the node (#836): how many there
// are, how many are expired, how many expire within certinfo.ExpiryWarning,
// and those ones, the soonest first. Unread says they could not be read
// (a value the base cannot decipher, for instance): the cell says so, the
// other figures stand.
type certSummary struct {
	Total, Expired, Expiring int
	Items                    []certItem
	Unread                   bool
}

// certItem is a certificate about to expire, or expired: who holds it and
// when it ends.
type certItem struct {
	Owner    string // ownerServer, ownerPartner, ownerAccount, ownerAuthority
	Name     string
	NotAfter time.Time
	Expired  bool
}

// readCerts reads the certificates of the node: the TLS certificates and
// trusted certificates of the servers, partners and accounts, and the TLS
// authorities; each is judged at now by the rule of the sheets.
func readCerts(db database.ReadAccess, now time.Time) (certSummary, error) {
	var (
		creds       model.Credentials
		authorities model.Authorities
		sum         certSummary
	)

	if err := db.Select(&creds).Where("type=? OR type=?", auth.TLSCertificate, auth.TLSTrustedCertificate).
		Run(); err != nil {
		return sum, fmt.Errorf("failed to read the certificates: %w", err)
	}

	if err := db.Select(&authorities).Where("type=?", auth.AuthorityTLS).Run(); err != nil {
		return sum, fmt.Errorf("failed to read the authorities: %w", err)
	}

	for _, cred := range creds {
		owner, name, err := ownerOf(db, cred)
		if err != nil {
			return sum, err
		}

		sum.add(now, owner, name, cred.Value)
	}

	for _, authority := range authorities {
		sum.add(now, ownerAuthority, authority.Name, authority.PublicIdentity)
	}

	sort.Slice(sum.Items, func(i, j int) bool { return sum.Items[i].NotAfter.Before(sum.Items[j].NotAfter) })

	return sum, nil
}

// add judges a certificate: expired, expiring within the warning, or valid
// (counted only); a text that is not a certificate is skipped.
func (s *certSummary) add(now time.Time, owner, name, pemText string) {
	info, ok := certinfo.Certificate(pemText)
	if !ok || info.NotAfter.IsZero() {
		return
	}

	s.Total++

	item := certItem{Owner: owner, Name: name, NotAfter: info.NotAfter}

	switch {
	case info.NotAfter.Before(now):
		s.Expired++
		item.Expired = true
	case info.NotAfter.Before(now.Add(certinfo.ExpiryWarning)):
		s.Expiring++
	default:
		return
	}

	s.Items = append(s.Items, item)
}

// ownerOf names the holder of a credential: a server, a partner or an
// account (by its login); the credential itself when it has none.
func ownerOf(db database.ReadAccess, cred *model.Credential) (owner, name string, err error) {
	switch {
	case cred.LocalAgentID.Valid:
		var server model.LocalAgent
		if getErr := db.Get(&server, "id=?", cred.LocalAgentID.Int64).Run(); getErr != nil {
			return "", "", fmt.Errorf("failed to read the server of a certificate: %w", getErr)
		}

		return ownerServer, server.Name, nil
	case cred.RemoteAgentID.Valid:
		var partner model.RemoteAgent
		if getErr := db.Get(&partner, "id=?", cred.RemoteAgentID.Int64).Run(); getErr != nil {
			return "", "", fmt.Errorf("failed to read the partner of a certificate: %w", getErr)
		}

		return ownerPartner, partner.Name, nil
	case cred.LocalAccountID.Valid:
		var account model.LocalAccount
		if getErr := db.Get(&account, "id=?", cred.LocalAccountID.Int64).Run(); getErr != nil {
			return "", "", fmt.Errorf("failed to read the account of a certificate: %w", getErr)
		}

		return ownerAccount, account.Login, nil
	case cred.RemoteAccountID.Valid:
		var account model.RemoteAccount
		if getErr := db.Get(&account, "id=?", cred.RemoteAccountID.Int64).Run(); getErr != nil {
			return "", "", fmt.Errorf("failed to read the account of a certificate: %w", getErr)
		}

		return ownerAccount, account.Login, nil
	default:
		return "", cred.Name, nil
	}
}
