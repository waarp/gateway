package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/bwmarrin/snowflake"
	"github.com/google/uuid"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

//nolint:gochecknoinits //init is needed here
func init() {
	database.AddInit(database.Initializer{
		Desc: "Initialize follow ID generator",
		Func: func(db database.Access) error {
			const snowflakeMaxNode = 1024

			var nodeID, mod, machineID big.Int

			nodeID.SetBytes([]byte(db.GetConfig().GatewayName + db.GetConfig().NodeID))
			mod.SetInt64(snowflakeMaxNode)
			machineID.Mod(&nodeID, &mod)

			generator, err := snowflake.NewNode(machineID.Int64())
			if err != nil {
				return fmt.Errorf("failed to create the ID generator: %w", err)
			}

			followIDGenerator = generator

			return nil
		},
	})
	database.AddInit(database.Initializer{
		Desc: "Initialize ID generators",
		Func: func(db database.Access) error {
			for proto, generator := range IDGenerators {
				if err := generator.Init(db); err != nil {
					return fmt.Errorf("failed to initialize ID generator for protocol %q: %w", proto, err)
				}
			}

			return nil
		},
	})
}

type Identifier struct {
	ID int64 `gorm:"autoIncrement;column:id"`
}

func ID(id int64) Identifier { return Identifier{ID: id} }

func (id Identifier) GetID() int64 { return id.ID }
func (id Identifier) NullableID() sql.NullInt64 {
	return sql.NullInt64{Int64: id.ID, Valid: true}
}

//nolint:gochecknoglobals //global var is needed here
var IDGenerators = map[string]IDGenerator{}

type IDGenerator interface {
	Init(db database.ReadAccess) error
	GetNextID() (string, error)
}

func generateRemoteID(protocol string) (string, error) {
	generator := IDGenerators[protocol]
	if generator == nil {
		generator = defaultIDGenerator{}
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return generator.GetNextID()
}

type defaultIDGenerator struct{}

func (defaultIDGenerator) Init(database.ReadAccess) error { return nil }
func (defaultIDGenerator) GetNextID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}

	return id.String(), nil
}

//nolint:gochecknoglobals //global var is needed here
var followIDGenerator *snowflake.Node

func generateFollowID() json.Number {
	return json.Number(followIDGenerator.Generate().Bytes())
}
