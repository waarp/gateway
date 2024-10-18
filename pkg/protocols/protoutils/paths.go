package protoutils

import (
	"errors"
	"fmt"
	gofs "io/fs"
	"path"
	"strings"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrDatabase         = errors.New("database error")
	ErrRuleNotFound     = errors.New("rule not found")
	ErrPermissionDenied = errors.New("permission denied")
)

// GetClosestRule returns the rule with the closest path to the given "rulePath".
// The "isSendPriority" parameter is used to prioritize "send" rules over
// "receive" rules, or vice-versa.
func GetClosestRule(db database.ReadAccess, logger *log.Logger, server *model.LocalAgent,
	acc *model.LocalAccount, rulePath string, isSendPriority bool,
) (*model.Rule, error) {
	rulePath = strings.TrimPrefix(rulePath, "/")
	if rulePath == "" || rulePath == "." || rulePath == "/" {
		return nil, ErrRuleNotFound
	}

	var rule model.Rule
	if err := db.Get(&rule, "path=? AND is_send=?", rulePath, isSendPriority).Run(); err != nil {
		if !database.IsNotFound(err) {
			logger.Error("Failed to retrieve rule: %s", err)

			return nil, ErrDatabase
		}

		if err := db.Get(&rule, "path=? AND is_send=?", rulePath, !isSendPriority).Run(); err != nil {
			if database.IsNotFound(err) {
				return GetClosestRule(db, logger, server, acc, path.Dir(rulePath), isSendPriority)
			}

			logger.Error("Failed to retrieve rule: %s", err)

			return nil, ErrDatabase
		}
	}

	if ok, err := rule.IsAuthorized(db, acc); err != nil {
		logger.Error("Failed to check rule permissions: %s", err)

		return nil, ErrDatabase
	} else if !ok {
		return &rule, ErrPermissionDenied
	}

	return &rule, nil
}

// GetRealPath returns the real filesystem path for the given "path" parameter
// by first removing the rule's path, and then building the real path using the
// given server & rule directories.
func GetRealPath(isTemp bool, db database.ReadAccess, logger *log.Logger,
	server *model.LocalAgent, acc *model.LocalAccount, filepath string,
) (string, error) {
	filepath = strings.TrimPrefix(filepath, "/")

	rule, err := GetClosestRule(db, logger, server, acc, filepath, true)
	if errors.Is(err, ErrRuleNotFound) {
		return "", nil //nolint:nilnil //returning nil here makes more sense than using a sentinel error
	} else if err != nil {
		return "", err
	}

	confPaths := &conf.GlobalConfig.Paths
	rest := strings.TrimPrefix(filepath, rule.Path)
	rest = strings.TrimPrefix(rest, "/")

	var (
		realDir string
		dirErr  error
	)

	switch {
	case rule.IsSend:
		realDir, dirErr = utils.GetPath(rest, utils.Leaf(rule.LocalDir),
			utils.Leaf(server.SendDir), utils.Branch(server.RootDir),
			utils.Leaf(confPaths.DefaultOutDir), utils.Branch(confPaths.GatewayHome))
	case isTemp:
		realDir, dirErr = utils.GetPath(rest+".part", utils.Leaf(rule.TmpLocalRcvDir),
			utils.Leaf(server.TmpReceiveDir), utils.Branch(server.RootDir),
			utils.Leaf(confPaths.DefaultTmpDir), utils.Branch(confPaths.GatewayHome))
	default:
		realDir, dirErr = utils.GetPath(rest, utils.Leaf(rule.LocalDir),
			utils.Leaf(server.ReceiveDir), utils.Branch(server.RootDir),
			utils.Leaf(confPaths.DefaultInDir), utils.Branch(confPaths.GatewayHome))
	}

	if dirErr != nil {
		return "", fmt.Errorf("failed to build the path: %w", dirErr)
	}

	return realDir, nil
}

func GetRulesPaths(db database.ReadAccess, serv *model.LocalAgent,
	acc *model.LocalAccount, dir string,
) ([]gofs.FileInfo, error) {
	dir = strings.TrimPrefix(dir, "/")

	var rules model.Rules

	query := db.Select(&rules).Distinct("path").Where(
		`(path LIKE ?) AND
		(
			(id IN 
				(SELECT DISTINCT rule_id FROM `+model.TableRuleAccesses+` WHERE
					(local_account_id=? OR local_agent_id=?)
				)
			)
			OR 
			( (SELECT COUNT(*) FROM `+model.TableRuleAccesses+` WHERE rule_id = id) = 0 )
		)`,
		dir+"%", acc.ID, serv.ID).OrderBy("path", true)

	if err := query.Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve rule list: %w", err)
	}

	if len(rules) == 0 {
		return nil, ErrRuleNotFound
	}

	paths := make([]string, 0, len(rules))
	dir += "/"

	for i := range rules {
		p := rules[i].Path
		p = strings.TrimPrefix(p, dir)
		p = strings.SplitN(p, "/", 2)[0] //nolint:mnd //not needed here

		if len(paths) == 0 || paths[len(paths)-1] != p {
			paths = append(paths, p)
		}
	}

	entries := make([]gofs.FileInfo, len(paths))

	for i := range paths {
		entries[i] = FakeDirInfo(paths[i])
	}

	return entries, nil
}
