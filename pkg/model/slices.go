package model

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type (
	Credentials         = database.Slice[*Credential]
	HistoryEntries      = database.Slice[*HistoryEntry]
	LocalAccounts       = database.Slice[*LocalAccount]
	LocalAgents         = database.Slice[*LocalAgent]
	RemoteAccounts      = database.Slice[*RemoteAccount]
	RemoteAgents        = database.Slice[*RemoteAgent]
	Clients             = database.Slice[*Client]
	Rules               = database.Slice[*Rule]
	RuleAccesses        = database.Slice[*RuleAccess]
	Tasks               = database.Slice[*Task]
	Transfers           = database.Slice[*Transfer]
	Users               = database.Slice[*User]
	NormalizedTransfers = database.Slice[*NormalizedTransferView]
	CloudInstances      = database.Slice[*CloudInstance]
	Authorities         = database.Slice[*Authority]
	Hosts               = database.Slice[*Host]
	CryptoKeys          = database.Slice[*CryptoKey]
	EmailTemplates      = database.Slice[*EmailTemplate]
	SMTPCredentials     = database.Slice[*SMTPCredential]
	FileWatchers        = database.Slice[*FileWatcher]
)
