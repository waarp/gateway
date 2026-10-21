package authorities

import (
	"bytes"
	"cmp"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"slices"
	"strconv"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
)

// Owners of a registered certificate, for the catalogue key of the sentence
// that names it ("authorities.signed.server").
const (
	ownerServer        = "server"
	ownerServerAccount = "server_account"
	ownerPartner       = "partner"
	ownerPartnerAcct   = "partner_account"
)

// signed is a certificate registered in the gateway that a TLS authority
// signs: who holds it (the kind of owner, its name and, for an account, the
// server or partner it belongs to), its name, and the page of the means.
type signed struct {
	Owner  string // ownerServer, ownerServerAccount, ownerPartner, ownerPartnerAcct
	Name   string // the server, partner or account login
	Parent string // the server or partner of an account, "" otherwise
	Means  string // the name of the certificate
	URL    string
}

// signedBy finds the certificates registered in the gateway (the certificates
// of the servers, partners and accounts, with or without their key) that the
// TLS authority signs, directly or through an intermediate the stored chain
// carries. Nothing for an SSH authority: its certificates are not stored.
func (h *Handlers) signedBy(e *entry) ([]signed, error) {
	if e.kind == nil || !e.kind.TLS || !e.Readable {
		return nil, nil
	}

	ca, parseErr := auth.ParseCertPEM(e.authority.PublicIdentity)
	if parseErr != nil {
		return nil, nil //nolint:nilerr // an unreadable identity signs nothing; the sheet says it is unreadable
	}

	var creds model.Credentials
	if err := h.db.Select(&creds).Where("type=? OR type=?", auth.TLSCertificate, auth.TLSTrustedCertificate).
		OrderBy("name", true).Run(); err != nil {
		return nil, fmt.Errorf("failed to list the registered certificates: %w", err)
	}

	var found []signed

	for _, c := range creds {
		if !signs(ca, c.Value) {
			continue
		}

		s, err := h.ownerOf(c)
		if err != nil {
			return nil, err
		}

		found = append(found, s)
	}

	slices.SortStableFunc(found, func(a, b signed) int {
		return cmp.Or(cmp.Compare(a.Owner, b.Owner), cmp.Compare(a.Parent, b.Parent),
			cmp.Compare(a.Name, b.Name), cmp.Compare(a.Means, b.Means))
	})

	return found, nil
}

// signs tells whether one of the certificates of a PEM chain is signed by
// the authority (the authority itself does not count).
func signs(ca *x509.Certificate, chainPEM string) bool {
	rest := []byte(chainPEM)

	for {
		var block *pem.Block
		if block, rest = pem.Decode(rest); block == nil {
			return false
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil || bytes.Equal(cert.Raw, ca.Raw) {
			continue
		}

		if cert.CheckSignatureFrom(ca) == nil {
			return true
		}
	}
}

// ownerOf names the holder of a certificate and gives the page of the means:
// under the server, the partner, or one of their accounts.
func (h *Handlers) ownerOf(c *model.Credential) (signed, error) {
	holder := signed{Means: c.Name}
	means := "/auth/" + strconv.FormatInt(c.ID, 10)

	switch {
	case c.LocalAgentID.Valid:
		var server model.LocalAgent
		if err := h.db.Get(&server, "id=?", c.LocalAgentID.Int64).Run(); err != nil {
			return holder, fmt.Errorf("failed to retrieve the server of %q: %w", c.Name, err)
		}

		holder.Owner, holder.Name = ownerServer, server.Name
		holder.URL = common.WebuiURL("/servers/" + strconv.FormatInt(server.ID, 10) + means)
	case c.RemoteAgentID.Valid:
		var partner model.RemoteAgent
		if err := h.db.Get(&partner, "id=?", c.RemoteAgentID.Int64).Run(); err != nil {
			return holder, fmt.Errorf("failed to retrieve the partner of %q: %w", c.Name, err)
		}

		holder.Owner, holder.Name = ownerPartner, partner.Name
		holder.URL = common.WebuiURL("/partners/" + strconv.FormatInt(partner.ID, 10) + means)
	case c.LocalAccountID.Valid:
		var (
			account model.LocalAccount
			server  model.LocalAgent
		)

		if err := h.db.Get(&account, "id=?", c.LocalAccountID.Int64).Run(); err != nil {
			return holder, fmt.Errorf("failed to retrieve the account of %q: %w", c.Name, err)
		}

		if err := h.db.Get(&server, "id=?", account.LocalAgentID).Run(); err != nil {
			return holder, fmt.Errorf("failed to retrieve the server of %q: %w", c.Name, err)
		}

		holder.Owner, holder.Name, holder.Parent = ownerServerAccount, account.Login, server.Name
		holder.URL = common.WebuiURL("/servers/" + strconv.FormatInt(server.ID, 10) +
			"/accounts/" + strconv.FormatInt(account.ID, 10) + means)
	case c.RemoteAccountID.Valid:
		var (
			account model.RemoteAccount
			partner model.RemoteAgent
		)

		if err := h.db.Get(&account, "id=?", c.RemoteAccountID.Int64).Run(); err != nil {
			return holder, fmt.Errorf("failed to retrieve the account of %q: %w", c.Name, err)
		}

		if err := h.db.Get(&partner, "id=?", account.RemoteAgentID).Run(); err != nil {
			return holder, fmt.Errorf("failed to retrieve the partner of %q: %w", c.Name, err)
		}

		holder.Owner, holder.Name, holder.Parent = ownerPartnerAcct, account.Login, partner.Name
		holder.URL = common.WebuiURL("/partners/" + strconv.FormatInt(partner.ID, 10) +
			"/accounts/" + strconv.FormatInt(account.ID, 10) + means)
	}

	return holder, nil
}
