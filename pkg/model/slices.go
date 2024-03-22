package model

import "code.waarp.fr/apps/gateway/gateway/pkg/database"

type Slice[T database.Table] []T

func (*Slice[T]) TableName() string { return T.TableName(*new(T)) }
func (*Slice[T]) Elem() string      { return T.Appellation(*new(T)) }

type (
	Cryptos             = Slice[*Crypto]
	Clients             = Slice[*Client]
	TransferInfoList    = Slice[*TransferInfo]
	HistoryEntries      = Slice[*HistoryEntry]
	LocalAccounts       = Slice[*LocalAccount]
	LocalAgents         = Slice[*LocalAgent]
	RemoteAccounts      = Slice[*RemoteAccount]
	RemoteAgents        = Slice[*RemoteAgent]
	Rules               = Slice[*Rule]
	RuleAccesses        = Slice[*RuleAccess]
	Tasks               = Slice[*Task]
	Transfers           = Slice[*Transfer]
	Users               = Slice[*User]
	NormalizedTransfers = Slice[*NormalizedTransferView]
	CloudInstances      = Slice[*CloudInstance]
)
