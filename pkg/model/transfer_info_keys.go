package model

// The reserved keys of the transfer information (#816), of the form
// __key__ with a lower-case first letter: the gateway reads some from the
// user (set in the command that creates the transfer, in a TRANSFER or
// PREREGISTER task, by SETINFO), writes others during the transfer, a few
// both ways. Every package names its keys from here, so that the screens
// that offer them (the variables list, the name of an information) and the
// documentation follow one source: a test checks that no other package
// declares a reserved key of its own.

// InfoKeyFamily groups the reserved keys by what they concern.
type InfoKeyFamily string

const (
	// InfoFamilyTransfer: the follow ID, the user content, the synchronised
	// transfers.
	InfoFamilyTransfer InfoKeyFamily = "transfer"
	// InfoFamilyPesit: the free texts, the file attributes and the articles
	// of PeSIT.
	InfoFamilyPesit InfoKeyFamily = "pesit"
	// InfoFamilyAck: the acknowledgements (SENDMESSAGE, PeSIT).
	InfoFamilyAck InfoKeyFamily = "ack"
)

// InfoKeySetter says who sets a reserved key.
type InfoKeySetter uint8

const (
	// InfoSetByUser: the user sets it, the gateway reads it.
	InfoSetByUser InfoKeySetter = iota + 1
	// InfoSetByGateway: the gateway writes it during the transfer.
	InfoSetByGateway
	// InfoSetByBoth: the user sets it on one side of the transfer, the
	// gateway writes it on the other.
	InfoSetByBoth
)

// The reserved keys. FollowID, SyncTransferID and SyncTransferRank keep the
// names the importers of the module know.
const (
	// FollowID: the R66 follow ID, which also links a transfer and its
	// acknowledgement.
	FollowID = "__followID__"
	// InfoUserContent: the R66 user content.
	InfoUserContent = "__userContent__"
	// SyncTransferID and SyncTransferRank: the parent and the rank of a
	// synchronised child transfer.
	SyncTransferID   = "__syncTransferID__"
	SyncTransferRank = "__syncTransferRank__"

	// The free texts of PeSIT: of the client and of the server, at the
	// connection and at the selection of the file; the raw PI 99 when the
	// legacy profile puts the free text in PI 94.

	InfoClientConnFreetext     = "__clientConnFreetext__"
	InfoClientTransFreetext    = "__clientTransFreetext__"
	InfoClientTransFreetextRaw = "__clientTransFreetextRaw__"
	InfoServerConnFreetext     = "__serverConnFreetext__"
	InfoServerTransFreetext    = "__serverTransFreetext__"
	// The file attributes of PeSIT: PI 16, PI 11, PI 33, PI 61, PI 62.

	InfoFileEncoding = "__fileEncoding__"
	InfoFileType     = "__fileType__"
	InfoOrganization = "__organization__"
	InfoCustomerID   = "__customerID__"
	InfoBankID       = "__bankID__"
	// The articles of PeSIT: their format (PI 31), their lengths, the
	// separator of the records of a text file sent one per article.

	InfoArticlesFormat    = "__articlesFormat__"
	InfoArticlesLengths   = "__articlesLengths__"
	InfoArticlesSeparator = "__articlesSeparator__"

	// The acknowledgements: expected, how long to wait for one and since
	// when, received and when; sent, when, to which partner, as which
	// account.

	InfoAckExpected    = "__ackExpected__"
	InfoAckWaitTimeout = "__ackWaitTimeout__"
	InfoAckWaitSince   = "__ackWaitSince__"
	InfoAckReceived    = "__ackReceived__"
	InfoAckReceivedOn  = "__ackReceivedOn__"
	InfoAckSent        = "__ackSent__"
	InfoAckSentOn      = "__ackSentOn__"
	InfoAckSentTo      = "__ackSentTo__"
	InfoAckSentAs      = "__ackSentAs__"
)

// ReservedInfoKey describes a reserved key of the transfer information.
type ReservedInfoKey struct {
	Name   string
	Family InfoKeyFamily
	SetBy  InfoKeySetter
}

// UserSets tells whether the user may set the key (the gateway reads it on
// at least one side of the transfer).
func (k ReservedInfoKey) UserSets() bool {
	return k.SetBy != InfoSetByGateway
}

// ReservedInfoKeys lists the reserved keys by family, in the order the
// screens show them.
func ReservedInfoKeys() []ReservedInfoKey {
	return []ReservedInfoKey{
		{FollowID, InfoFamilyTransfer, InfoSetByBoth},
		{InfoUserContent, InfoFamilyTransfer, InfoSetByBoth},
		{SyncTransferID, InfoFamilyTransfer, InfoSetByGateway},
		{SyncTransferRank, InfoFamilyTransfer, InfoSetByGateway},

		{InfoClientConnFreetext, InfoFamilyPesit, InfoSetByBoth},
		{InfoClientTransFreetext, InfoFamilyPesit, InfoSetByBoth},
		{InfoClientTransFreetextRaw, InfoFamilyPesit, InfoSetByGateway},
		{InfoServerConnFreetext, InfoFamilyPesit, InfoSetByGateway},
		{InfoServerTransFreetext, InfoFamilyPesit, InfoSetByBoth},
		{InfoFileEncoding, InfoFamilyPesit, InfoSetByBoth},
		{InfoFileType, InfoFamilyPesit, InfoSetByBoth},
		{InfoOrganization, InfoFamilyPesit, InfoSetByBoth},
		{InfoCustomerID, InfoFamilyPesit, InfoSetByBoth},
		{InfoBankID, InfoFamilyPesit, InfoSetByBoth},
		{InfoArticlesFormat, InfoFamilyPesit, InfoSetByBoth},
		{InfoArticlesLengths, InfoFamilyPesit, InfoSetByBoth},
		{InfoArticlesSeparator, InfoFamilyPesit, InfoSetByUser},

		{InfoAckExpected, InfoFamilyAck, InfoSetByBoth},
		{InfoAckWaitTimeout, InfoFamilyAck, InfoSetByBoth},
		{InfoAckWaitSince, InfoFamilyAck, InfoSetByGateway},
		{InfoAckReceived, InfoFamilyAck, InfoSetByGateway},
		{InfoAckReceivedOn, InfoFamilyAck, InfoSetByGateway},
		{InfoAckSent, InfoFamilyAck, InfoSetByGateway},
		{InfoAckSentOn, InfoFamilyAck, InfoSetByGateway},
		{InfoAckSentTo, InfoFamilyAck, InfoSetByGateway},
		{InfoAckSentAs, InfoFamilyAck, InfoSetByGateway},
	}
}
