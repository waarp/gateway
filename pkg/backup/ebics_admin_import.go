package backup

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func importEbicsAdminData(
	logger *log.Logger,
	db database.Access,
	data *file.Data,
	reset bool,
) error {
	if reset {
		if err := purgeEbicsAdminData(db); err != nil {
			return err
		}
	}

	if err := importEbicsHosts(logger, db, data.EbicsHosts); err != nil {
		return err
	}

	if err := importEbicsSubscribers(logger, db, data.EbicsSubscribers); err != nil {
		return err
	}

	if err := importEbicsBankKeys(logger, db, data.EbicsBankKeys); err != nil {
		return err
	}

	if err := importEbicsStandardBTFCatalogs(logger, db, data.EbicsStandardBTFCatalogs); err != nil {
		return err
	}

	if err := importEbicsPayloadProfiles(logger, db, data.EbicsPayloadProfiles, false); err != nil {
		return err
	}

	if err := importEbicsRTNProviders(logger, db, data.EbicsRTNProviders); err != nil {
		return err
	}

	if err := (&model.EbicsStandardBTFCatalog{}).Init(db); err != nil {
		return fmt.Errorf("failed to restore the default EBICS standard BTF catalogs: %w", err)
	}

	return nil
}

func purgeEbicsAdminData(db database.Access) error {
	if err := db.DeleteAll(&model.EbicsRTNProvider{}).Owner().Run(); err != nil {
		return fmt.Errorf("failed to purge %s: %w", model.NameEbicsRTNProvider, err)
	}
	if err := db.DeleteAll(&model.EbicsPayloadProfile{}).Owner().Run(); err != nil {
		return fmt.Errorf("failed to purge %s: %w", model.NameEbicsPayloadProfile, err)
	}
	if err := db.DeleteAll(&model.EbicsBankKey{}).Owner().Run(); err != nil {
		return fmt.Errorf("failed to purge %s: %w", model.NameEbicsBankKey, err)
	}
	if err := db.DeleteAll(&model.EbicsStandardBTFCatalog{}).Owner().Run(); err != nil {
		return fmt.Errorf("failed to purge %s: %w", model.NameEbicsStandardBTFCatalog, err)
	}
	if err := db.DeleteAll(&model.EbicsSubscriber{}).Owner().Run(); err != nil {
		return fmt.Errorf("failed to purge %s: %w", model.NameEbicsSubscriber, err)
	}
	if err := db.DeleteAll(&model.EbicsHost{}).Owner().Run(); err != nil {
		return fmt.Errorf("failed to purge %s: %w", model.NameEbicsHost, err)
	}

	return nil
}

func importEbicsHosts(logger *log.Logger, db database.Access, hosts []file.EbicsHost) error {
	for i := range hosts {
		src := &hosts[i]

		var (
			dbHost model.EbicsHost
			isNew  bool
		)

		if err := db.Get(&dbHost, "host_id=?", src.HostID).Owner().Run(); database.IsNotFound(err) {
			isNew = true
		} else if err != nil {
			return fmt.Errorf("failed to retrieve EBICS host %q: %w", src.HostID, err)
		}

		dbHost.Name = src.Name
		dbHost.HostID = src.HostID
		dbHost.Description = src.Description
		dbHost.Enabled = src.Enabled
		dbHost.IsServer = src.IsServer
		dbHost.ProtocolVersion = src.ProtocolVersion
		dbHost.Transport = src.Transport
		dbHost.DefaultBankURL = src.DefaultBankURL

		var dbErr error
		if isNew {
			logger.Infof("Create EBICS host %q", dbHost.Name)
			dbErr = db.Insert(&dbHost).Run()
		} else {
			logger.Infof("Update EBICS host %q", dbHost.Name)
			dbErr = db.Update(&dbHost).Run()
		}
		if dbErr != nil {
			return fmt.Errorf("failed to import EBICS host %q: %w", dbHost.Name, dbErr)
		}
	}

	return nil
}

func importEbicsSubscribers(
	logger *log.Logger,
	db database.Access,
	subscribers []file.EbicsSubscriber,
) error {
	for i := range subscribers {
		src := &subscribers[i]

		hostID, err := getImportEbicsHostID(db, src.HostID)
		if err != nil {
			return err
		}

		var (
			dbSubscriber model.EbicsSubscriber
			isNew        bool
		)

		if err = db.Get(
			&dbSubscriber,
			"ebics_host_id=? AND partner_id=? AND user_id=?",
			hostID,
			src.PartnerID,
			src.UserID,
		).Owner().Run(); database.IsNotFound(err) {
			isNew = true
		} else if err != nil {
			return fmt.Errorf(
				"failed to retrieve EBICS subscriber %q/%q on host %q: %w",
				src.PartnerID,
				src.UserID,
				src.HostID,
				err,
			)
		}

		dbSubscriber.EbicsHostID = hostID
		dbSubscriber.Name = src.Name
		dbSubscriber.PartnerID = src.PartnerID
		dbSubscriber.UserID = src.UserID
		dbSubscriber.SystemID = src.SystemID
		dbSubscriber.AccountRole = src.AccountRole
		dbSubscriber.TransportURL = src.TransportURL
		dbSubscriber.Enabled = src.Enabled
		dbSubscriber.DefaultOrderDataEncoding = src.DefaultOrderDataEncoding
		dbSubscriber.LocalAccountID = sql.NullInt64{}
		dbSubscriber.RemoteAccountID = sql.NullInt64{}

		if src.LocalServer != "" || src.LocalAccount != "" {
			localAccountID, getErr := getImportLocalAccountID(db, src.LocalServer, src.LocalAccount)
			if getErr != nil {
				return fmt.Errorf(
					"failed to resolve local account reference of EBICS subscriber %q: %w",
					src.Name,
					getErr,
				)
			}
			dbSubscriber.LocalAccountID = utils.NewNullInt64(localAccountID)
		}

		if src.RemotePartner != "" || src.RemoteAccount != "" {
			remoteAccountID, getErr := getImportRemoteAccountID(db, src.RemotePartner, src.RemoteAccount)
			if getErr != nil {
				return fmt.Errorf(
					"failed to resolve remote account reference of EBICS subscriber %q: %w",
					src.Name,
					getErr,
				)
			}
			dbSubscriber.RemoteAccountID = utils.NewNullInt64(remoteAccountID)
		}

		var dbErr error
		if isNew {
			logger.Infof("Create EBICS subscriber %q", dbSubscriber.Name)
			dbErr = db.Insert(&dbSubscriber).Run()
		} else {
			logger.Infof("Update EBICS subscriber %q", dbSubscriber.Name)
			dbErr = db.Update(&dbSubscriber).Run()
		}
		if dbErr != nil {
			return fmt.Errorf("failed to import EBICS subscriber %q: %w", dbSubscriber.Name, dbErr)
		}
	}

	return nil
}

func importEbicsBankKeys(logger *log.Logger, db database.Access, keys []file.EbicsBankKey) error {
	for i := range keys {
		src := &keys[i]

		hostID, err := getImportEbicsHostID(db, src.HostID)
		if err != nil {
			return err
		}

		var (
			dbKey model.EbicsBankKey
			isNew bool
		)

		if err = db.Get(
			&dbKey,
			"ebics_host_id=? AND key_type=? AND version=?",
			hostID,
			src.KeyType,
			src.Version,
		).Owner().Run(); database.IsNotFound(err) {
			isNew = true
		} else if err != nil {
			return fmt.Errorf(
				"failed to retrieve EBICS bank key %q/%q on host %q: %w",
				src.KeyType,
				src.Version,
				src.HostID,
				err,
			)
		}

		dbKey.EbicsHostID = hostID
		dbKey.KeyType = src.KeyType
		dbKey.Version = src.Version
		dbKey.PublicKey = src.PublicKey
		dbKey.PublicKeyHash = src.PublicKeyHash
		dbKey.State = src.State
		dbKey.ValidFrom = src.ValidFrom
		dbKey.ValidTo = src.ValidTo

		var dbErr error
		if isNew {
			logger.Infof("Create EBICS bank key %q/%q", dbKey.KeyType, dbKey.Version)
			dbErr = db.Insert(&dbKey).Run()
		} else {
			logger.Infof("Update EBICS bank key %q/%q", dbKey.KeyType, dbKey.Version)
			dbErr = db.Update(&dbKey).Run()
		}
		if dbErr != nil {
			return fmt.Errorf(
				"failed to import EBICS bank key %q/%q: %w",
				dbKey.KeyType,
				dbKey.Version,
				dbErr,
			)
		}
	}

	return nil
}

func importEbicsStandardBTFCatalogs(
	logger *log.Logger,
	db database.Access,
	catalogs []file.EbicsStandardBTFCatalog,
) error {
	for i := range catalogs {
		src := &catalogs[i]

		var (
			dbCatalog model.EbicsStandardBTFCatalog
			isNew     bool
		)

		if err := db.Get(
			&dbCatalog,
			"name=? AND scope=? AND catalog_version=?",
			src.Name,
			src.Scope,
			src.CatalogVersion,
		).Owner().Run(); database.IsNotFound(err) {
			isNew = true
		} else if err != nil {
			return fmt.Errorf(
				"failed to retrieve EBICS standard BTF catalog %q/%q version %q: %w",
				src.Name,
				src.Scope,
				src.CatalogVersion,
				err,
			)
		}

		dbCatalog.Name = src.Name
		dbCatalog.Scope = src.Scope
		dbCatalog.CatalogVersion = src.CatalogVersion
		dbCatalog.SourceType = src.SourceType
		dbCatalog.SourceRef = src.SourceRef
		dbCatalog.Status = src.Status
		dbCatalog.SeedChecksum = src.SeedChecksum

		var dbErr error
		if isNew {
			logger.Infof("Create EBICS standard BTF catalog %q/%q", dbCatalog.Name, dbCatalog.Scope)
			dbErr = db.Insert(&dbCatalog).Run()
		} else {
			logger.Infof("Update EBICS standard BTF catalog %q/%q", dbCatalog.Name, dbCatalog.Scope)
			dbErr = db.Update(&dbCatalog).Run()
		}
		if dbErr != nil {
			return fmt.Errorf(
				"failed to import EBICS standard BTF catalog %q/%q: %w",
				dbCatalog.Name,
				dbCatalog.Scope,
				dbErr,
			)
		}

		if err := db.DeleteAll(&model.EbicsStandardBTFEntry{}).
			Owner().
			Where("catalog_id=?", dbCatalog.ID).
			Run(); err != nil {
			return fmt.Errorf(
				"failed to purge entries of EBICS standard BTF catalog %q/%q: %w",
				dbCatalog.Name,
				dbCatalog.Scope,
				err,
			)
		}

		for j := range src.Entries {
			entrySrc := &src.Entries[j]
			dbEntry := model.EbicsStandardBTFEntry{
				CatalogID:         dbCatalog.ID,
				EntryKey:          entrySrc.EntryKey,
				OrderType:         entrySrc.OrderType,
				Direction:         entrySrc.Direction,
				ServiceName:       entrySrc.ServiceName,
				ServiceOption:     entrySrc.ServiceOption,
				Scope:             entrySrc.Scope,
				MsgName:           entrySrc.MsgName,
				ContainerType:     entrySrc.ContainerType,
				CountryGroup:      entrySrc.CountryGroup,
				IsDefaultTemplate: entrySrc.IsDefaultTemplate,
				Status:            entrySrc.Status,
				MetadataMap:       entrySrc.Metadata,
			}

			if err := db.Insert(&dbEntry).Run(); err != nil {
				return fmt.Errorf(
					"failed to import EBICS standard BTF entry %q in catalog %q/%q: %w",
					dbEntry.EntryKey,
					dbCatalog.Name,
					dbCatalog.Scope,
					err,
				)
			}
		}
	}

	return nil
}

func importEbicsRTNProviders(
	logger *log.Logger,
	db database.Access,
	providers []file.EbicsRTNProvider,
) error {
	for i := range providers {
		src := &providers[i]

		subscriberID, err := getImportEbicsSubscriberID(db, src.HostID, src.PartnerID, src.UserID)
		if err != nil {
			return err
		}

		var (
			dbProvider model.EbicsRTNProvider
			isNew      bool
		)

		if err = db.Get(&dbProvider, "name=?", src.Name).Owner().Run(); database.IsNotFound(err) {
			isNew = true
		} else if err != nil {
			return fmt.Errorf("failed to retrieve EBICS RTN provider %q: %w", src.Name, err)
		}

		dbProvider.Name = src.Name
		dbProvider.Transport = src.Transport
		dbProvider.Enabled = src.Enabled
		dbProvider.EbicsSubscriberID = subscriberID
		dbProvider.AutoPullPolicy = src.AutoPullPolicy
		dbProvider.ConfigurationMap = src.Configuration

		var dbErr error
		if isNew {
			logger.Infof("Create EBICS RTN provider %q", dbProvider.Name)
			dbErr = db.Insert(&dbProvider).Run()
		} else {
			logger.Infof("Update EBICS RTN provider %q", dbProvider.Name)
			dbErr = db.Update(&dbProvider).Run()
		}
		if dbErr != nil {
			return fmt.Errorf("failed to import EBICS RTN provider %q: %w", dbProvider.Name, dbErr)
		}
	}

	return nil
}

func getImportEbicsHostID(db database.Access, hostID string) (int64, error) {
	host := &model.EbicsHost{}
	if err := db.Get(host, "host_id=?", hostID).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, database.NewValidationErrorf("the EBICS host %q does not exist", hostID)
		}

		return 0, fmt.Errorf("failed to retrieve EBICS host %q: %w", hostID, err)
	}

	return host.ID, nil
}

func getImportEbicsSubscriberID(
	db database.Access,
	hostID, partnerID, userID string,
) (int64, error) {
	dbHostID, err := getImportEbicsHostID(db, hostID)
	if err != nil {
		return 0, err
	}

	subscriber := &model.EbicsSubscriber{}
	if err = db.Get(
		subscriber,
		"ebics_host_id=? AND partner_id=? AND user_id=?",
		dbHostID,
		partnerID,
		userID,
	).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, database.NewValidationErrorf(
				"the EBICS subscriber %q/%q on host %q does not exist",
				partnerID,
				userID,
				hostID,
			)
		}

		return 0, fmt.Errorf(
			"failed to retrieve EBICS subscriber %q/%q on host %q: %w",
			partnerID,
			userID,
			hostID,
			err,
		)
	}

	return subscriber.ID, nil
}

func getImportLocalAccountID(db database.Access, serverName, login string) (int64, error) {
	server := &model.LocalAgent{}
	if err := db.Get(server, "name=?", serverName).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, database.NewValidationErrorf("the local server %q does not exist", serverName)
		}

		return 0, fmt.Errorf("failed to retrieve local server %q: %w", serverName, err)
	}

	account := &model.LocalAccount{}
	if err := db.Get(account, "local_agent_id=? AND login=?", server.ID, login).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, database.NewValidationErrorf(
				"the local account %q does not exist on server %q",
				login,
				serverName,
			)
		}

		return 0, fmt.Errorf(
			"failed to retrieve local account %q on server %q: %w",
			login,
			serverName,
			err,
		)
	}

	return account.ID, nil
}

func getImportRemoteAccountID(db database.Access, partnerName, login string) (int64, error) {
	partner := &model.RemoteAgent{}
	if err := db.Get(partner, "name=?", partnerName).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, database.NewValidationErrorf("the remote partner %q does not exist", partnerName)
		}

		return 0, fmt.Errorf("failed to retrieve remote partner %q: %w", partnerName, err)
	}

	account := &model.RemoteAccount{}
	if err := db.Get(account, "remote_agent_id=? AND login=?", partner.ID, login).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, database.NewValidationErrorf(
				"the remote account %q does not exist on partner %q",
				login,
				partnerName,
			)
		}

		return 0, fmt.Errorf(
			"failed to retrieve remote account %q on partner %q: %w",
			login,
			partnerName,
			err,
		)
	}

	return account.ID, nil
}
