package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

//nolint:dupl //duplicate is for a different type, keep separate
func retrieveDBSNMPMonitor(r *http.Request, db *database.DB) (*snmp.MonitorConfig, error) {
	name, ok := mux.Vars(r)["snmp_monitor"]
	if !ok {
		return nil, notFound("missing SNMP monitor name")
	}

	var monitor snmp.MonitorConfig
	if err := db.Get(&monitor, "name=? AND owner=?", name,
		conf.GlobalConfig.GatewayName).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("SNMP monitor %q not found", name)
		}

		return nil, fmt.Errorf("failed to retrieve SNMP monitor %q: %w", name, err)
	}

	return &monitor, nil
}

func addSnmpMonitor(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restMonitor api.PostSnmpMonitorReqObject
		if err := readJSON(r, &restMonitor); handleError(w, logger, err) {
			return
		}

		dbMonitor := snmp.MonitorConfig{
			Name:            restMonitor.Name,
			Version:         restMonitor.Version,
			UDPAddress:      restMonitor.UDPAddress,
			Community:       restMonitor.Community,
			UseInforms:      restMonitor.UseInforms,
			SNMPv3Security:  restMonitor.SNMPv3Security,
			ContextName:     restMonitor.ContextName,
			ContextEngineID: restMonitor.ContextEngineID,
			AuthEngineID:    restMonitor.AuthEngineID,
			AuthUsername:    restMonitor.AuthUsername,
			AuthProtocol:    restMonitor.AuthProtocol,
			AuthPassphrase:  types.CypherText(restMonitor.AuthPassphrase),
			PrivProtocol:    restMonitor.PrivProtocol,
			PrivPassphrase:  types.CypherText(restMonitor.PrivPassphrase),
		}
		if err := db.Insert(&dbMonitor).Run(); handleError(w, logger, err) {
			return
		}

		if handleError(w, logger, reloadSNMPMonitorConf()) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbMonitor.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func dbSNMPMonitorToREST(dbMonitor *snmp.MonitorConfig) *api.GetSnmpMonitorRespObject {
	return &api.GetSnmpMonitorRespObject{
		Name:            dbMonitor.Name,
		Version:         dbMonitor.Version,
		UDPAddress:      dbMonitor.UDPAddress,
		Community:       dbMonitor.Community,
		UseInforms:      dbMonitor.UseInforms,
		SNMPv3Security:  dbMonitor.SNMPv3Security,
		ContextName:     dbMonitor.ContextName,
		ContextEngineID: dbMonitor.ContextEngineID,
		AuthEngineID:    dbMonitor.AuthEngineID,
		AuthUsername:    dbMonitor.AuthUsername,
		AuthProtocol:    dbMonitor.AuthProtocol,
		AuthPassphrase:  string(dbMonitor.AuthPassphrase),
		PrivProtocol:    dbMonitor.PrivProtocol,
		PrivPassphrase:  string(dbMonitor.PrivPassphrase),
	}
}

func listSnmpMonitors(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":  order{"name", true},
		"name+":    order{"name", true},
		"name-":    order{"name", false},
		"address+": order{"udp_address", true},
		"address-": order{"udp_address", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbMonitors model.Slice[*snmp.MonitorConfig]

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbMonitors)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Where("owner=?", conf.GlobalConfig.GatewayName).
			Run(); handleError(w, logger, err) {
			return
		}

		restMonitors := make([]*api.GetSnmpMonitorRespObject, len(dbMonitors))
		for i, dbMonitor := range dbMonitors {
			restMonitors[i] = dbSNMPMonitorToREST(dbMonitor)
		}

		response := map[string][]*api.GetSnmpMonitorRespObject{"snmpMonitors": restMonitors}
		handleError(w, logger, writeJSON(w, response))
	}
}

func getSnmpMonitor(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitor, getErr := retrieveDBSNMPMonitor(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restUser := dbSNMPMonitorToREST(monitor)
		handleError(w, logger, writeJSON(w, restUser))
	}
}

func updateSnmpMonitor(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldDbMonitor, getErr := retrieveDBSNMPMonitor(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restMonitor := api.PatchSnmpMonitorReqObject{
			Name:       asNullableStr(oldDbMonitor.Name),
			Version:    asNullableStr(oldDbMonitor.Version),
			UDPAddress: asNullableStr(oldDbMonitor.UDPAddress),
			Community:  asNullableStr(oldDbMonitor.Community),
		}
		if err := readJSON(r, &restMonitor); handleError(w, logger, err) {
			return
		}

		dbMonitor := snmp.MonitorConfig{ID: oldDbMonitor.ID}
		setIfValid(&dbMonitor.Name, restMonitor.Name)
		setIfValid(&dbMonitor.Version, restMonitor.Version)
		setIfValid(&dbMonitor.UDPAddress, restMonitor.UDPAddress)
		setIfValid(&dbMonitor.Community, restMonitor.Community)
		setIfValid(&dbMonitor.UseInforms, restMonitor.UseInforms)
		setIfValid(&dbMonitor.SNMPv3Security, restMonitor.SNMPv3Security)
		setIfValid(&dbMonitor.ContextName, restMonitor.ContextName)
		setIfValid(&dbMonitor.ContextEngineID, restMonitor.ContextEngineID)
		setIfValid(&dbMonitor.AuthEngineID, restMonitor.AuthEngineID)
		setIfValid(&dbMonitor.AuthUsername, restMonitor.AuthUsername)
		setIfValid(&dbMonitor.AuthProtocol, restMonitor.AuthProtocol)
		setIfValid(&dbMonitor.PrivProtocol, restMonitor.PrivProtocol)

		if restMonitor.AuthPassphrase.Valid {
			dbMonitor.AuthPassphrase = types.CypherText(restMonitor.AuthPassphrase.Value)
		}

		if restMonitor.PrivPassphrase.Valid {
			dbMonitor.PrivPassphrase = types.CypherText(restMonitor.PrivPassphrase.Value)
		}

		if err := db.Update(&dbMonitor).Run(); handleError(w, logger, err) {
			return
		}

		if handleError(w, logger, reloadSNMPMonitorConf()) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbMonitor.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteSnmpMonitor(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitor, getErr := retrieveDBSNMPMonitor(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.Delete(monitor).Run(); handleError(w, logger, err) {
			return
		}

		if handleError(w, logger, reloadSNMPMonitorConf()) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func reloadSNMPMonitorConf() error {
	if snmp.GlobalService != nil {
		//nolint:wrapcheck //no need to wrap here
		return snmp.GlobalService.ReloadMonitorsConf()
	}

	return nil
}
