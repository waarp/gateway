package model

import (
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type Slice[T database.Table] []T

func (*Slice[T]) TableName() string { return T.TableName(*new(T)) }
func (*Slice[T]) Elem() string      { return T.Appellation(*new(T)) }

func (s *Slice[T]) AfterRead(db database.ReadAccess) error {
	for _, elem := range *s {
		if hook, ok := any(elem).(database.ReadCallback); ok {
			if err := hook.AfterRead(db); err != nil {
				return err //nolint:wrapcheck //wrapping adds nothing here
			}
		}
	}

	return nil
}

func (s Slice[T]) String() string {
	builder := &strings.Builder{}

	for i, elem := range s {
		builder.WriteString(fmt.Sprintf("Item #%d: %+v\n", i, elem))
	}

	return builder.String()
}

type (
	Credentials         = Slice[*Credential]
	TransferInfoList    = Slice[*TransferInfo]
	HistoryEntries      = Slice[*HistoryEntry]
	LocalAccounts       = Slice[*LocalAccount]
	LocalAgents         = Slice[*LocalAgent]
	RemoteAccounts      = Slice[*RemoteAccount]
	RemoteAgents        = Slice[*RemoteAgent]
	Clients             = Slice[*Client]
	Rules               = Slice[*Rule]
	RuleAccesses        = Slice[*RuleAccess]
	Tasks               = Slice[*Task]
	Transfers           = Slice[*Transfer]
	Users               = Slice[*User]
	NormalizedTransfers = Slice[*NormalizedTransferView]
	CloudInstances      = Slice[*CloudInstance]
	Authorities         = Slice[*Authority]
	Hosts               = Slice[*Host]
)
