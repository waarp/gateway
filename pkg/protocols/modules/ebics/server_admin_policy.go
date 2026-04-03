package ebics

import (
	"context"
	"errors"
	"fmt"

	libebics "code.waarp.fr/lib/ebics/ebics"
	libadminhelper "code.waarp.fr/lib/ebics/ebics/provider/adminhelper"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

var errMissingServerAdminPolicy = errors.New("the EBICS server admin policy is not configured")

type serverAdminPolicy struct {
	store *providerStore
}

func newServerAdminPolicy(store *providerStore) libadminhelper.KeyMgmtPolicy {
	return &serverAdminPolicy{store: store}
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandleINI(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandleHIA(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandleHPB(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandlePUB(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandleHSA(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandleH3K(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandleHCA(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandleHCS(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) ShouldHandleSPR(_ context.Context, req libebics.OrderContext) error {
	return p.validateOperationalSubscriber(req)
}

//nolint:gocritic // lib-ebics policy interface imposes value semantics here.
func (p *serverAdminPolicy) validateOperationalSubscriber(req libebics.OrderContext) error {
	if p == nil || p.store == nil {
		return errMissingServerAdminPolicy
	}

	host, err := p.store.getHostByHostID(string(req.HostID))
	if err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS host %q does not exist", req.HostID)
		}

		return fmt.Errorf("load EBICS host %q for server admin order: %w", req.HostID, err)
	}
	if !host.Enabled {
		return database.NewValidationErrorf("the EBICS host %q is disabled", host.HostID)
	}
	if !host.IsServer {
		return database.NewValidationErrorf("the EBICS host %q is not configured for the server role", host.HostID)
	}

	subscriber, err := p.store.getSubscriber(string(req.HostID), string(req.PartnerID), string(req.UserID))
	if err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %q/%q does not exist on host %q",
				req.PartnerID,
				req.UserID,
				req.HostID,
			)
		}

		return fmt.Errorf(
			"load EBICS subscriber %q/%q on host %q for server admin order: %w",
			req.PartnerID,
			req.UserID,
			req.HostID,
			err,
		)
	}
	if !subscriber.Enabled {
		return database.NewValidationErrorf(
			"the EBICS subscriber %q/%q is disabled on host %q",
			req.PartnerID,
			req.UserID,
			req.HostID,
		)
	}
	if !subscriber.LocalAccountID.Valid {
		return database.NewValidationErrorf(
			"the EBICS subscriber %q/%q has no local account for server orders",
			req.PartnerID,
			req.UserID,
		)
	}
	if subscriber.AccountRole == "CLIENT" {
		return database.NewValidationErrorf(
			"the EBICS subscriber %q/%q cannot use the CLIENT role for server orders",
			req.PartnerID,
			req.UserID,
		)
	}

	return nil
}
