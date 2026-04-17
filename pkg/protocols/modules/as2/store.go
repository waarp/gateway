package as2

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"

	"code.waarp.fr/lib/as2"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrStoreNoPut       = errors.New("adding partners to store is not supported")
	ErrStoreNoDelete    = errors.New("deleting partners from store is not supported")
	ErrStoreInvalidCert = errors.New("invalid certificate")
	ErrPartnerNotFound  = errors.New("partner not found")

	ErrDatabase = errors.New("database error")
)

type partnerStore struct {
	db     *database.DB
	logger *log.Logger
	agent  *model.LocalAgent
}

func (s *server) newPartnerStore() *partnerStore {
	return &partnerStore{db: s.db, logger: s.logger, agent: s.agent}
}

func (p *partnerStore) Put(context.Context, *as2.Partner) error { return ErrStoreNoPut }
func (p *partnerStore) Delete(context.Context, string) error    { return ErrStoreNoDelete }

func (p *partnerStore) GetByAS2From(ctx context.Context, as2From string) (*as2.Partner, error) {
	return utils.RunWithCtx2(ctx, func() (*as2.Partner, error) {
		return p.getByAS2From(as2From)
	})
}

func (p *partnerStore) getByAS2From(as2From string) (*as2.Partner, error) {
	acc, err := p.getAccount(as2From)
	if err != nil {
		return nil, err
	}

	return &as2.Partner{
		Name:  acc.Login,
		AS2ID: acc.Login,
		Validator: &certValidator{
			db:     p.db,
			logger: p.logger,
			agent:  p.agent,
			acc:    acc,
		},
	}, nil
}

func (p *partnerStore) getAccount(login string) (*model.LocalAccount, error) {
	var acc model.LocalAccount
	if err := p.db.Get(&acc, "login=?", login).And("local_agent_id=?", p.agent.ID).
		Run(); database.IsNotFound(err) {
		return nil, ErrPartnerNotFound
	} else if err != nil {
		p.logger.Errorf("Failed to retrieve the local account %q: %v", login, err)

		return nil, ErrDatabase
	}

	return &acc, nil
}

func (p *partnerStore) List(ctx context.Context) ([]*as2.Partner, error) {
	return utils.RunWithCtx2(ctx, p.list)
}

func (p *partnerStore) list() ([]*as2.Partner, error) {
	var accounts model.LocalAccounts
	if err := p.db.Select(&accounts).Where("local_agent_id=?", p.agent.ID).Run(); err != nil {
		p.logger.Errorf("Failed to retrieve the local accounts: %v", err)

		return nil, ErrDatabase
	}

	partners := make([]*as2.Partner, 0, len(accounts))

	for _, acc := range accounts {
		partners = append(partners, &as2.Partner{
			Name:      acc.Login,
			AS2ID:     acc.Login,
			Validator: &certValidator{db: p.db, logger: p.logger, agent: p.agent, acc: acc},
		})
	}

	return partners, nil
}

type certValidator struct {
	db     *database.DB
	logger *log.Logger
	agent  *model.LocalAgent
	acc    *model.LocalAccount
}

func (v *certValidator) Validate(cert, issuer *x509.Certificate) error {
	res, err := v.acc.Authenticate(v.db, v.agent, auth.TLSTrustedCertificate,
		[]*x509.Certificate{cert, issuer})
	if err != nil {
		v.logger.Errorf("Failed to authenticate account %q with TLS: %v", v.acc.Login, err)

		return ErrDatabase
	}

	if !res.Success {
		return fmt.Errorf("%w: %s", ErrStoreInvalidCert, res.Reason)
	}

	return nil
}
