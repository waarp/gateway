package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type exportAccountRef struct {
	agentName string
	login     string
}

func exportEbicsHosts(logger *log.Logger, db database.ReadAccess) ([]file.EbicsHost, error) {
	var dbHosts model.EbicsHosts
	if err := db.Select(&dbHosts).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS hosts: %w", err)
	}

	hosts := make([]file.EbicsHost, len(dbHosts))
	for i, host := range dbHosts {
		logger.Infof("Exporting EBICS host %q", host.Name)
		hosts[i] = file.EbicsHost{
			Name:            host.Name,
			HostID:          host.HostID,
			Description:     host.Description,
			Enabled:         host.Enabled,
			IsServer:        host.IsServer,
			ProtocolVersion: host.ProtocolVersion,
			Transport:       host.Transport,
			DefaultBankURL:  host.DefaultBankURL,
		}
	}

	return hosts, nil
}

func exportEbicsSubscribers(
	logger *log.Logger,
	db database.ReadAccess,
) ([]file.EbicsSubscriber, error) {
	var dbSubscribers model.EbicsSubscribers
	if err := db.Select(&dbSubscribers).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS subscribers: %w", err)
	}

	subscribers := make([]file.EbicsSubscriber, len(dbSubscribers))
	for i, subscriber := range dbSubscribers {
		logger.Infof("Exporting EBICS subscriber %q", subscriber.Name)

		host, err := getExportEbicsHost(db, subscriber.EbicsHostID)
		if err != nil {
			return nil, fmt.Errorf("failed to export EBICS subscriber %q host: %w", subscriber.Name, err)
		}

		out := file.EbicsSubscriber{
			Name:                     subscriber.Name,
			HostID:                   host.HostID,
			PartnerID:                subscriber.PartnerID,
			UserID:                   subscriber.UserID,
			SystemID:                 subscriber.SystemID,
			AccountRole:              subscriber.AccountRole,
			TransportURL:             subscriber.TransportURL,
			Enabled:                  subscriber.Enabled,
			DefaultOrderDataEncoding: subscriber.DefaultOrderDataEncoding,
		}

		if subscriber.LocalAccountID.Valid {
			ref, refErr := getExportLocalAccountRef(db, subscriber.LocalAccountID.Int64)
			if refErr != nil {
				return nil, fmt.Errorf(
					"failed to export local account reference of EBICS subscriber %q: %w",
					subscriber.Name,
					refErr,
				)
			}
			out.LocalServer = ref.agentName
			out.LocalAccount = ref.login
		}

		if subscriber.RemoteAccountID.Valid {
			ref, refErr := getExportRemoteAccountRef(db, subscriber.RemoteAccountID.Int64)
			if refErr != nil {
				return nil, fmt.Errorf(
					"failed to export remote account reference of EBICS subscriber %q: %w",
					subscriber.Name,
					refErr,
				)
			}
			out.RemotePartner = ref.agentName
			out.RemoteAccount = ref.login
		}

		subscribers[i] = out
	}

	return subscribers, nil
}

func exportEbicsBankKeys(
	logger *log.Logger,
	db database.ReadAccess,
) ([]file.EbicsBankKey, error) {
	var dbKeys model.EbicsBankKeys
	if err := db.Select(&dbKeys).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS bank keys: %w", err)
	}

	keys := make([]file.EbicsBankKey, len(dbKeys))
	for i, key := range dbKeys {
		logger.Infof("Exporting EBICS bank key %q/%q", key.KeyType, key.Version)

		host, err := getExportEbicsHost(db, key.EbicsHostID)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to export EBICS bank key %q/%q host: %w",
				key.KeyType,
				key.Version,
				err,
			)
		}

		keys[i] = file.EbicsBankKey{
			HostID:        host.HostID,
			KeyType:       key.KeyType,
			Version:       key.Version,
			PublicKey:     key.PublicKey,
			PublicKeyHash: key.PublicKeyHash,
			State:         key.State,
			ValidFrom:     key.ValidFrom,
			ValidTo:       key.ValidTo,
		}
	}

	return keys, nil
}

func exportEbicsRTNProviders(
	logger *log.Logger,
	db database.ReadAccess,
) ([]file.EbicsRTNProvider, error) {
	var dbProviders model.EbicsRTNProviders
	if err := db.Select(&dbProviders).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS RTN providers: %w", err)
	}

	providers := make([]file.EbicsRTNProvider, len(dbProviders))
	for i, provider := range dbProviders {
		logger.Infof("Exporting EBICS RTN provider %q", provider.Name)

		subscriber, err := getExportEbicsSubscriber(db, provider.EbicsSubscriberID)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to export EBICS RTN provider %q subscriber: %w",
				provider.Name,
				err,
			)
		}

		host, err := getExportEbicsHost(db, subscriber.EbicsHostID)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to export EBICS RTN provider %q host: %w",
				provider.Name,
				err,
			)
		}

		configuration, err := decodeStringMapJSON(provider.Configuration)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to decode configuration of EBICS RTN provider %q: %w",
				provider.Name,
				err,
			)
		}

		providers[i] = file.EbicsRTNProvider{
			Name:           provider.Name,
			Transport:      provider.Transport,
			Enabled:        provider.Enabled,
			HostID:         host.HostID,
			PartnerID:      subscriber.PartnerID,
			UserID:         subscriber.UserID,
			AutoPullPolicy: provider.AutoPullPolicy,
			Configuration:  configuration,
		}
	}

	return providers, nil
}

func getExportEbicsHost(db database.ReadAccess, id int64) (*model.EbicsHost, error) {
	host := &model.EbicsHost{}
	if err := db.Get(host, "id=?", id).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS host %d: %w", id, err)
	}

	return host, nil
}

func getExportEbicsSubscriber(db database.ReadAccess, id int64) (*model.EbicsSubscriber, error) {
	subscriber := &model.EbicsSubscriber{}
	if err := db.Get(subscriber, "id=?", id).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS subscriber %d: %w", id, err)
	}

	return subscriber, nil
}

func getExportLocalAccountRef(db database.ReadAccess, id int64) (*exportAccountRef, error) {
	account := &model.LocalAccount{}
	if getErr := db.Get(account, "id=?", id).Run(); getErr != nil {
		return nil, fmt.Errorf("failed to retrieve local account %d: %w", id, getErr)
	}

	agent := &model.LocalAgent{}
	if getErr := db.Get(agent, "id=?", account.LocalAgentID).Owner().Run(); getErr != nil {
		return nil, fmt.Errorf("failed to retrieve local agent %d: %w", account.LocalAgentID, getErr)
	}

	return &exportAccountRef{agentName: agent.Name, login: account.Login}, nil
}

func getExportRemoteAccountRef(db database.ReadAccess, id int64) (*exportAccountRef, error) {
	account := &model.RemoteAccount{}
	if getErr := db.Get(account, "id=?", id).Run(); getErr != nil {
		return nil, fmt.Errorf("failed to retrieve remote account %d: %w", id, getErr)
	}

	agent := &model.RemoteAgent{}
	if getErr := db.Get(agent, "id=?", account.RemoteAgentID).Owner().Run(); getErr != nil {
		return nil, fmt.Errorf("failed to retrieve remote partner %d: %w", account.RemoteAgentID, getErr)
	}

	return &exportAccountRef{agentName: agent.Name, login: account.Login}, nil
}
