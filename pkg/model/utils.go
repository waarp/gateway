package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/bwmarrin/snowflake"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var errWriteOnView = errors.New("cannot insert/update on a view")

// getCredentials fetch from the database then return the associated Credentials if they exist.
func getCredentials(db database.ReadAccess, owner authentication.Owner,
	authTypes ...string,
) (Credentials, error) {
	var auths Credentials
	query := db.Select(&auths).Where(owner.GetCredCond()).OrderBy("id", true)

	if len(authTypes) > 0 {
		vals := make([]interface{}, len(authTypes))

		for i := range authTypes {
			vals[i] = authTypes[i]
		}

		query.In("type", vals...)
	}

	if err := query.Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the cryptos: %w", err)
	}

	// TODO: get only validate certificates
	return auths, nil
}

func getTransferInfo(db database.ReadAccess, owner transferInfoOwner,
) (map[string]any, error) {
	var infoList TransferInfoList
	if err := db.Select(&infoList).Where(owner.getTransInfoCondition()).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the transfer info list: %w", err)
	}

	infoMap := map[string]interface{}{}

	for _, info := range infoList {
		decoder := json.NewDecoder(strings.NewReader(info.Value))
		decoder.UseNumber()

		var val interface{}
		if err := decoder.Decode(&val); err != nil {
			return nil, database.NewValidationError(`invalid transfer info value "%v": %s`, info.Value, err)
		}

		infoMap[info.Name] = val
	}

	return infoMap, nil
}

func setTransferInfo(access database.Access, owner transferInfoOwner,
	info map[string]any,
) error {
	switch db := access.(type) {
	case *database.DB:
		//nolint:wrapcheck //wrapping this error would add nothing
		return db.Transaction(func(ses *database.Session) error {
			return doSetTransferInfo(ses, owner, info)
		})
	case *database.Session:
		return doSetTransferInfo(db, owner, info)
	default:
		panic(fmt.Sprintf("unknown database access type %T", access))
	}
}

func doSetTransferInfo(ses *database.Session, owner transferInfoOwner,
	info map[string]any,
) error {
	delQuery := ses.DeleteAll(&TransferInfo{}).Where(owner.getTransInfoCondition())
	if _, ok := info[FollowID]; !ok {
		delQuery.Where("name <> ?", FollowID)
	}

	if err := delQuery.Run(); err != nil {
		return fmt.Errorf("failed to delete transfer info: %w", err)
	}

	for name, val := range info {
		str, err := json.Marshal(val)
		if err != nil {
			return database.NewValidationError(`invalid transfer info value "%v": %w`, val, err)
		}

		i := &TransferInfo{Name: name, Value: string(str)}
		owner.setTransInfoOwner(i)

		if err2 := ses.Insert(i).Run(); err2 != nil {
			return fmt.Errorf("failed to insert transfer info: %w", err2)
		}
	}

	return nil
}

func makeIDGenerator() (*snowflake.Node, error) {
	var nodeID, mod, machineID big.Int

	nodeID.SetBytes([]byte(conf.GlobalConfig.GatewayName + conf.GlobalConfig.NodeID))
	mod.SetInt64(math.MaxInt64)

	machineID.Mod(&nodeID, &mod)

	generator, err := snowflake.NewNode(machineID.Int64())
	if err != nil {
		return nil, fmt.Errorf("failed to create the ID generator: %w", err)
	}

	return generator, nil
}

func countTrue(b ...bool) int {
	count := 0

	for _, v := range b {
		if v {
			count++
		}
	}

	return count
}

func checkCloudInstance(db database.ReadAccess, fsPath *types.FSPath) error {
	if fsPath.Path == "" || fsPath.Backend == "" {
		return nil
	}

	if _, ok := filesystems.TestFileSystems.Load(fsPath.Backend); ok {
		return nil
	}

	var cloud CloudInstance
	if err := db.Get(&cloud, "name=?", fsPath.Backend).And("owner=?",
		conf.GlobalConfig.GatewayName).Run(); database.IsNotFound(err) {
		return database.NewValidationError("no remote cloud instance %q found", fsPath.Backend)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve cloud instance: %w", err)
	}

	return nil
}
