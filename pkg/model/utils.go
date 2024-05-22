package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
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
		var val interface{}
		if err := json.Unmarshal([]byte(info.Value), &val); err != nil {
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
	if err := ses.DeleteAll(&TransferInfo{}).Where(owner.getTransInfoCondition()).
		Run(); err != nil {
		return fmt.Errorf("failed to delete transfer info: %w", err)
	}

	for name, val := range info {
		str, err := json.Marshal(val)
		if err != nil {
			return database.NewValidationError(`invalid transfer info value "%v": %w`, val, err)
		}

		i := &TransferInfo{Name: name, Value: string(str)}
		owner.setTransInfoOwner(i)

		if err := ses.Insert(i).Run(); err != nil {
			return fmt.Errorf("failed to insert transfer info: %w", err)
		}
	}

	return nil
}

func makeIDGenerator() (*snowflake.Node, error) {
	var nodeID, mod, machineID big.Int

	nodeID.SetBytes([]byte(conf.GlobalConfig.NodeID))
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

type authentCacheKey struct {
	Id       int64
	AuthType string
	Value    any
}

type authentCacheVal struct {
	result     *authentication.Result
	expiration time.Time
}

func getCachableValue(authType string, value any) (bool, any) {
    if authType == "password_hash" {
        v, ok := value.(string)
        if !ok {
            return false, nil
        }
        return true, v
    }
    if authType == "password" {
        return true, value
    }
    return false, nil
}

const deltaCacheExpiration time.Duration = 3 * time.Second

var cacheLocalAuthent sync.Map = sync.Map{}
var cacheRemoteAuthent sync.Map = sync.Map{}

func authenticate(db database.ReadAccess, owner CredOwnerTable, authType string, value any,
) (res *authentication.Result, err error) {
	handler := authentication.GetInternalAuthHandler(authType)
	if handler == nil {
		//nolint:goerr113 //dynamic error is better here for debugging
		return nil, fmt.Errorf("unknown authentication type %q", authType)
	}

	// TODO get correct map and key
	cache := cacheLocalAuthent

	// get cache key
  if ok, v := getCachableValue(authType, value); ok {
		_, id := owner.GetCredCond()
		
    key := authentCacheKey{
			Id:       id,
			AuthType: authType,
			Value:    v,
		}
   

		defer func() {
			if err == nil {
				cache.Store(key, authentCacheVal{
					result:     res,
					expiration: time.Now().Add(deltaCacheExpiration),
				})
			}
		}()

		if res, ok := cache.Load(key); ok && res != nil {
			res := res.(authentCacheVal)
			if time.Now().Before(res.expiration) {
				return res.result, nil
			}
		}
	}

	//nolint:wrapcheck //error is returned as is for better message readability
	return handler.Authenticate(db, owner, value)
}
