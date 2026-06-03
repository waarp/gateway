package pipeline

import (
	"path"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func GetOldTransferByRemoteID(db database.ReadAccess, remoteID string,

	account *model.LocalAccount, rule *model.Rule,

) (*model.Transfer, *Error) {

	var oldTrans model.Transfer

	if err := db.Get(&oldTrans, "remote_transfer_id=?", remoteID).
		And("local_account_id=?", account.ID).
		And("rule_id=?", rule.ID).
		OrderBy("start", false).
		Run(); err != nil {

		return nil, NewErrorWith(types.TeInternal, "failed to retrieve old server transfer", err)

	}

	if oldTrans.Status == types.StatusRunning {

		return nil, NewError(types.TeForbidden,

			"cannot resume a currently running transfer")

	}

	return &oldTrans, nil

}

func GetOldTransferByFilename(db database.ReadAccess, filepath string, offset int64,

	account *model.LocalAccount, rule *model.Rule,

) (*model.Transfer, *Error) {

	var oldTrans model.Transfer

	query := db.Get(&oldTrans, "local_account_id=?", account.ID).
		And("rule_id=?", rule.ID).
		And("status <> ?", types.StatusRunning).
		And("progress >= ?", offset).
		OrderBy("start", false)

	if rule.IsSend {

		query.And("src_filename=?", filepath)

	} else {

		query.And("dest_filename=?", filepath)

	}

	if err := query.Run(); err != nil {

		return nil, NewErrorWith(types.TeInternal, "failed to retrieve old server transfer", err)

	}

	return &oldTrans, nil

}

func GetAvailableTransferByFilename(db database.ReadAccess, filepath, remoteID string,

	account *model.LocalAccount, rule *model.Rule,

) (*model.Transfer, *Error) {

	var availableTrans model.Transfer

	query := db.Get(&availableTrans, "local_account_id=?", account.ID).
		And("rule_id=?", rule.ID).
		In("status", types.StatusAvailable).
		OrderBy("start", false)

	if rule.IsSend {

		query.And("src_filename=?", filepath)

	} else {

		query.And("dest_filename=?", filepath)

	}

	if err := query.Run(); err != nil {

		return nil, NewErrorWith(types.TeInternal, "failed to retrieve server transfer", err)

	}

	if remoteID != "" {

		availableTrans.RemoteTransferID = remoteID

	}

	return &availableTrans, nil

}

// ContainsWildcard reports whether the given filepath contains glob

// metacharacters ('*' or '?').

func ContainsWildcard(filepath string) bool {

	return strings.ContainsAny(filepath, "*?")

}

// GetAvailableTransferByPattern retrieves the first AVAILABLE transfer

// whose filename matches the given glob pattern. It fetches all AVAILABLE

// transfers for the rule/account and filters them in Go using path.Match,

// which keeps the implementation database-agnostic.

func GetAvailableTransferByPattern(db database.ReadAccess, pattern, remoteID string,

	account *model.LocalAccount, rule *model.Rule,

) (*model.Transfer, *Error) {

	var candidates model.Transfers

	query := db.Select(&candidates).
		Where("local_account_id=?", account.ID).
		Where("rule_id=?", rule.ID).
		Where("status=?", types.StatusAvailable).
		OrderBy("start", false)

	if err := query.Run(); err != nil {

		return nil, NewErrorWith(types.TeInternal,

			"failed to retrieve available transfers for pattern matching", err)

	}

	for _, trans := range candidates {

		filename := trans.SrcFilename

		if !rule.IsSend {

			filename = trans.DestFilename

		}

		matched, matchErr := path.Match(pattern, filename)

		if matchErr != nil {

			continue

		}

		if matched {

			if remoteID != "" {

				trans.RemoteTransferID = remoteID

			}

			return trans, nil

		}

	}

	return nil, NewError(types.TeFileNotFound, "no available transfer matches pattern")

}

func GetAvailableTransferByRule(db database.ReadAccess, remoteID string,

	account *model.LocalAccount, rule *model.Rule,

) (*model.Transfer, *Error) {

	var availableTrans model.Transfer

	if err := db.Get(&availableTrans, "local_account_id=?", account.ID).
		And("rule_id=?", rule.ID).
		In("status", types.StatusAvailable, types.StatusError).
		OrderBy("start", false).
		Run(); err != nil {

		return nil, NewErrorWith(types.TeInternal, "failed to retrieve server transfer", err)

	}

	if remoteID != "" {

		availableTrans.RemoteTransferID = remoteID

	}

	return &availableTrans, nil

}

func MakeServerTransfer(remoteID, filepath string, account *model.LocalAccount, rule *model.Rule,

) *model.Transfer {

	newTrans := &model.Transfer{

		RemoteTransferID: remoteID,

		RuleID: rule.ID,

		LocalAccountID: utils.NewNullInt64(account.ID),

		Filesize: model.UnknownSize,

		Start: time.Now(),

		Status: types.StatusPlanned,
	}

	if rule.IsSend {

		newTrans.SrcFilename = filepath

	} else {

		newTrans.DestFilename = filepath

	}

	return newTrans

}

// NewServerPipeline initializes and returns a new pipeline suitable for a

// server transfer.

func NewServerPipeline(db *database.DB, logger *log.Logger, trans *model.Transfer,

	snmpService *snmp.Service,

) (*Pipeline, *Error) {

	transCtx, ctxErr := model.GetTransferContext(db, logger, trans)

	if ctxErr != nil {

		return nil, NewError(types.TeInternal, "database error")

	}

	pipeline, pipErr := newPipeline(db, logger, transCtx, snmpService)

	if pipErr != nil {

		logger.Errorf("Failed to initialize the server transfer pipeline %d: %v",

			trans.ID, pipErr)

		return nil, pipErr

	}

	if transCtx.Rule.IsSend {

		pipeline.Logger.Infof(

			"Starting download of file %q requested by %q on the server %q using rule %q",

			transCtx.Transfer.LocalPath, transCtx.LocalAccount.Login,

			transCtx.LocalAgent.Name, transCtx.Rule.Name)

	} else {

		pipeline.Logger.Infof(

			"Starting upload of file %q requested by %q on the server %q using rule %q",

			transCtx.Transfer.LocalPath, transCtx.LocalAccount.Login,

			transCtx.LocalAgent.Name, transCtx.Rule.Name)

	}

	return pipeline, nil

}
