package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func dbServerToRESTInput(dbServer *model.LocalAgent) *api.InServer {
	return &api.InServer{
		Name:          &dbServer.Name,
		Protocol:      &dbServer.Protocol,
		Address:       &dbServer.Address,
		RootDir:       &dbServer.RootDir,
		ReceiveDir:    &dbServer.ReceiveDir,
		SendDir:       &dbServer.SendDir,
		TmpReceiveDir: &dbServer.TmpReceiveDir,
		ProtoConfig:   dbServer.ProtoConfig,
	}
}

// restServerToDB transforms the JSON local agent into its database equivalent.
func restServerToDB(restServer *api.InServer, logger *log.Logger,
) *model.LocalAgent {
	root := str(restServer.RootDir)
	sndDir := str(restServer.SendDir)
	rcvDir := str(restServer.ReceiveDir)
	tmpDir := str(restServer.TmpReceiveDir)

	if root == "" && restServer.Root != nil {
		logger.Warning("JSON field 'root' is deprecated, use 'rootDir' instead")

		root = utils.DenormalizePath(str(restServer.Root))
	}

	if rcvDir == "" && restServer.InDir != nil {
		logger.Warning("JSON field 'inDir' is deprecated, use 'receiveDir' instead")

		rcvDir = utils.DenormalizePath(str(restServer.InDir))
	}

	if sndDir == "" && restServer.OutDir != nil {
		logger.Warning("JSON field 'outDir' is deprecated, use 'sendDir' instead")

		sndDir = utils.DenormalizePath(str(restServer.OutDir))
	}

	if tmpDir == "" && restServer.WorkDir != nil {
		logger.Warning("JSON field 'workDir' is deprecated, use 'tmpLocalRcvDir' instead")

		tmpDir = utils.DenormalizePath(str(restServer.WorkDir))
	}

	return &model.LocalAgent{
		Owner:         conf.GlobalConfig.GatewayName,
		Name:          str(restServer.Name),
		Address:       str(restServer.Address),
		RootDir:       root,
		ReceiveDir:    rcvDir,
		SendDir:       sndDir,
		TmpReceiveDir: tmpDir,
		Protocol:      str(restServer.Protocol),
		ProtoConfig:   restServer.ProtoConfig,
	}
}

func dbPartnerToRESTInput(dbPartner *model.RemoteAgent) *api.InPartner {
	return &api.InPartner{
		Name:        &dbPartner.Name,
		Protocol:    &dbPartner.Protocol,
		Address:     &dbPartner.Address,
		ProtoConfig: dbPartner.ProtoConfig,
	}
}

// restPartnerToDB transforms the JSON remote agent into its database equivalent.
func restPartnerToDB(restPartner *api.InPartner) *model.RemoteAgent {
	return &model.RemoteAgent{
		Name:        str(restPartner.Name),
		Protocol:    str(restPartner.Protocol),
		Address:     str(restPartner.Address),
		ProtoConfig: restPartner.ProtoConfig,
	}
}

// DBServerToREST transforms the given database local agent into its JSON
// equivalent.
func DBServerToREST(db database.ReadAccess, dbServer *model.LocalAgent) (*api.OutServer, error) {
	if dbServer.Protocol == config.ProtocolR66TLS {
		var r66Conf *config.R66ServerProtoConfig
		if json.Unmarshal(dbServer.ProtoConfig, r66Conf) == nil && r66Conf.IsTLS != nil {
			// To preserve backwards compatibility, when `ìsTLS` is defined, we
			// change the protocol back to config.ProtocolR66, like it was before the addition
			// of the config.ProtocolR66TLS protocol.
			dbServer.Protocol = config.ProtocolR66
		}
	}

	authorizedRules, err := getAuthorizedRules(db, dbServer)
	if err != nil {
		return nil, err
	}

	return &api.OutServer{
		Name:            dbServer.Name,
		Enabled:         dbServer.Enabled,
		Protocol:        dbServer.Protocol,
		Address:         dbServer.Address,
		RootDir:         dbServer.RootDir,
		SendDir:         dbServer.SendDir,
		ReceiveDir:      dbServer.ReceiveDir,
		TmpReceiveDir:   dbServer.TmpReceiveDir,
		ProtoConfig:     dbServer.ProtoConfig,
		AuthorizedRules: authorizedRules,

		Root:    utils.NormalizePath(dbServer.RootDir),
		InDir:   utils.NormalizePath(dbServer.ReceiveDir),
		OutDir:  utils.NormalizePath(dbServer.SendDir),
		WorkDir: utils.NormalizePath(dbServer.TmpReceiveDir),
	}, nil
}

// DBServersToREST transforms the given list of database local agents into
// its JSON equivalent.
func DBServersToREST(db database.ReadAccess, dbServers []*model.LocalAgent) ([]*api.OutServer, error) {
	restServers := make([]*api.OutServer, len(dbServers))

	for i, dbServer := range dbServers {
		var err error
		if restServers[i], err = DBServerToREST(db, dbServer); err != nil {
			return nil, err
		}
	}

	return restServers, nil
}

// DBPartnerToREST transforms the given database remote agent into its JSON
// equivalent.
func DBPartnerToREST(db database.ReadAccess, ag *model.RemoteAgent) (*api.OutPartner, error) {
	if ag.Protocol == config.ProtocolR66TLS {
		var r66Conf *config.R66PartnerProtoConfig
		if json.Unmarshal(ag.ProtoConfig, r66Conf) == nil && r66Conf.IsTLS != nil {
			// To preserve backwards compatibility, when `ìsTLS` is defined, we
			// change the protocol back to config.ProtocolR66, like it was before the addition
			// of the config.ProtocolR66TLS protocol.
			ag.Protocol = config.ProtocolR66
		}
	}

	authorizedRules, err := getAuthorizedRules(db, ag)
	if err != nil {
		return nil, err
	}

	return &api.OutPartner{
		Name:            ag.Name,
		Protocol:        ag.Protocol,
		Address:         ag.Address,
		ProtoConfig:     ag.ProtoConfig,
		AuthorizedRules: authorizedRules,
	}, nil
}

// DBPartnersToREST transforms the given list of database remote agents into
// its JSON equivalent.
func DBPartnersToREST(db database.ReadAccess, dbPartners []*model.RemoteAgent) ([]*api.OutPartner, error) {
	restPartners := make([]*api.OutPartner, len(dbPartners))

	for i, dbPartner := range dbPartners {
		var err error
		if restPartners[i], err = DBPartnerToREST(db, dbPartner); err != nil {
		}
	}

	return restPartners, nil
}

func parseProtoParam(r *http.Request, query *database.SelectQuery) error {
	if len(r.Form["protocol"]) > 0 {
		protos := make([]string, len(r.Form["protocol"]))

		for i, p := range r.Form["protocol"] {
			if _, ok := config.ProtoConfigs[p]; !ok {
				return badRequest(fmt.Sprintf("'%s' is not a valid protocol", p))
			}

			protos[i] = p
		}

		query.In("protocol", protos)
	}

	return nil
}
