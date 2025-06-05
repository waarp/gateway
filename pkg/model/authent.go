package model

import (
	"fmt"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
)

const (
	deltaCacheClear      = 4 * time.Second
	deltaCacheExpiration = 3 * time.Second
)

//nolint:gochecknoglobals //global var is used for simplicity
var (
	clearCacheTicker = time.NewTicker(deltaCacheClear)

	localAuthentCache  = sync.Map{}
	remoteAuthentCache = sync.Map{}
)

//nolint:gochecknoinits //init is better for performance
func init() {
	go func() {
		for {
			<-clearCacheTicker.C
			clearAuthentCache(&localAuthentCache)
			clearAuthentCache(&remoteAuthentCache)
		}
	}()
}

type authentCacheKey struct {
	ID       int64
	AuthType string
	Value    any
}

type authentCacheVal struct {
	result     *authentication.Result
	expiration time.Time
}

func getCachableValue(authType string, value any) (bool, any) {
	if authType == authPassword {
		switch typedVal := value.(type) {
		case string:
			return true, typedVal
		case []byte:
			return true, string(typedVal)
		default:
			return false, nil
		}
	}

	return false, nil
}

func clearAuthentCache(cache *sync.Map) {
	now := time.Now()

	cache.Range(func(key, value any) bool {
		//nolint:forcetypeassert,errcheck //only authentCacheVal are stored into the map
		cacheVal, _ := value.(authentCacheVal)
		if cacheVal.expiration.Before(now) {
			cache.Delete(key)
		}

		return true
	})
}

func authenticate(db database.ReadAccess, owner CredOwnerTable, authType, proto string, value any,
) (res *authentication.Result, err error) {
	handler := authentication.GetInternalAuthHandler(authType, proto)
	if handler == nil {
		//nolint:err113 //dynamic error is better here for debugging
		return nil, fmt.Errorf("unknown authentication type %q", authType)
	}

	cache := &localAuthentCache
	if !owner.IsServer() {
		cache = &remoteAuthentCache
	}

	// get cache key
	if ok, v := getCachableValue(authType, value); ok {
		_, id := owner.GetCredCond()

		key := authentCacheKey{
			ID:       id,
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

		if res, ok2 := cache.Load(key); ok2 && res != nil {
			//nolint:forcetypeassert,errcheck //only authentCacheVal are stored into the map
			res, _ := res.(authentCacheVal)
			if time.Now().Before(res.expiration) {
				return res.result, nil
			}
		}
	}

	//nolint:wrapcheck //error is returned as is for better message readability
	return handler.Authenticate(db, owner, value)
}
