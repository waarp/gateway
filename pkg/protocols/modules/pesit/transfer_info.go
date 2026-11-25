package pesit

import (
	"cmp"
	"errors"
	"io"
	"maps"
	"reflect"
	"time"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	fileEncodingKey = "__fileEncoding__"
	fileTypeKey     = "__fileType__"
	organizationKey = "__organization__"
	customerIDKey   = "__customerID__"
	bankIDKey       = "__bankID__"

	ackExpectedKey   = tasks.SendMessageAckExpectedKey
	ackTimeout       = pipeline.AckWaitTimeout
	ackWaitSinceKey  = pipeline.AckWaitSince
	ackSentKey       = tasks.SendMessageAckSentKey
	ackSentOnKey     = tasks.SendMessageAckSentOnKey
	ackReceivedKey   = pipeline.AckReceived
	ackReceivedOnKey = pipeline.AckReceivedOnKey

	clientConnFreetextKey  = "__clientConnFreetext__"
	clientTransFreetextKey = "__clientTransFreetext__"
	// Legacy profile: PI 99 as received, when PI 94 took its place as free text.
	clientTransFreetextRawKey = "__clientTransFreetextRaw__"
	serverConnFreetextKey     = "__serverConnFreetext__"
	serverTransFreetextKey    = "__serverTransFreetext__"

	articlesLengthsKey   = "__articlesLengths__"
	articlesFormatKey    = "__articlesFormat__"
	articlesSeparatorKey = "__articlesSeparator__"

	// What the partner announces about the file beyond the keys above.
	fileCallerIDKey      = "__fileCallerID__"
	fileServerIDKey      = "__fileServerID__"
	priorityKey          = "__priority__"
	keyLengthKey         = "__keyLength__"
	keyOffsetKey         = "__keyOffset__"
	reservedSpaceKey     = "__reservedSpace__"
	reservedSpaceUnitKey = "__reservedSpaceUnit__"
	fileCreation         = model.InfoFileBTime
	fileLastAccess       = model.InfoFileATime
	fileTimeZone         = "__fileTimeZone__"
)

// announced is what a partner says about the file in a request or in the
// answer to a retrieval, beyond the attributes kept elsewhere: the file
// identifiers (PGI 9: PI 3, 4), the priority (PI 17), the key of an indexed
// file (PI 38, 39), the space to reserve (PI 41, 42) and the dates (PI 51, 52).
// A zero field is one the partner did not send.
type announced struct {
	callerID, serverID   string
	priority             pesit.Priority
	keyLength, keyOffset uint16
	reservedSpace        uint32
	reservedUnit         pesit.FileSizeUnit
	tz                   *time.Location
	creation, lastAccess time.Time
}

// infos are the transfer information that keep what the partner announced:
// one key per attribute it sent, none for the others. The normal priority is
// the default of the protocol, so it is not kept: it cannot be told apart from
// an absent PI 17.
func (a *announced) infos() map[string]any {
	infos := map[string]any{}
	put := func(key string, val any, set bool) {
		if set {
			infos[key] = val
		}
	}

	put(fileCallerIDKey, a.callerID, a.callerID != "")
	put(fileServerIDKey, a.serverID, a.serverID != "")
	put(priorityKey, a.priority.String(), a.priority != pesit.PriorityNormal)
	put(keyLengthKey, a.keyLength, a.keyLength != 0)
	put(keyOffsetKey, a.keyOffset, a.keyOffset != 0)
	put(reservedSpaceKey, a.reservedSpace, a.reservedSpace != 0)
	put(reservedSpaceUnitKey, a.reservedUnit.String(), a.reservedSpace != 0)
	put(fileCreation, tsInTz(a.creation, a.tz), !a.creation.IsZero())
	put(fileLastAccess, tsInTz(a.lastAccess, a.tz), !a.lastAccess.IsZero())

	return infos
}

func getTZ(pip *pipeline.Pipeline) *time.Location {
	str, err := utils.GetAs[string](pip.TransCtx.Transfer.TransferInfo, fileTimeZone)
	if err != nil {
		return time.Local
	}

	loc, err := time.LoadLocation(str)
	if err != nil {
		pip.Logger.Warningf("invalid file timezone %q, using local timezone instead", str)

		return time.Local
	}

	return loc
}

func tsInTz(ts time.Time, tz *time.Location) string {
	if tz == nil {
		tz = time.Local
	}

	return time.Date(ts.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(),
		ts.Second(), 0, tz).Format(time.RFC3339)
}

// unlessSame returns val, or "" when it repeats ref.
func unlessSame(val, ref string) string {
	if val == ref {
		return ""
	}

	return val
}

// keepAnnounced writes what the partner announced in the transfer information.
// The dates are written as transmitted: the library labels them UTC, and
// Format keeps their wall clock.
func keepAnnounced(pip *pipeline.Pipeline, a *announced) {
	maps.Copy(pip.TransCtx.Transfer.TransferInfo, a.infos())
}

func setPesitInfo[T cmp.Ordered, F ~func(T) bool](pip *pipeline.Pipeline, key string, set F) *pipeline.Error {
	val, err := utils.GetAs[T](pip.TransCtx.Transfer.TransferInfo, key)
	if errors.Is(err, utils.ErrKeyNotFound) {
		return nil
	} else if err != nil {
		return pipeline.NewError(types.TeInternal, err.Error())
	}

	set(val)

	return nil
}

//nolint:dupl //keep separate of setFileOrganization
func setFileEncoding(pip *pipeline.Pipeline, f interface {
	SetDataCoding(encoding pesit.DataCoding) bool
},
) *pipeline.Error {
	return setPesitInfo(pip, fileEncodingKey, func(str string) bool {
		var enc pesit.DataCoding

		switch str {
		case pesit.CodingBinary.String():
			enc = pesit.CodingBinary
		case pesit.CodingASCII.String():
			enc = pesit.CodingASCII
		case pesit.CodingEBCDIC.String():
			enc = pesit.CodingEBCDIC
		default:
			return false
		}

		return f.SetDataCoding(enc)
	})
}

func setFileType(pip *pipeline.Pipeline, f interface {
	SetFileType(fileType uint16) bool
},
) *pipeline.Error {
	return setPesitInfo(pip, fileTypeKey, f.SetFileType)
}

//nolint:dupl //keep separate of setFileOrganization
func setFileOrganization(pip *pipeline.Pipeline, f interface {
	SetFileOrganization(organization pesit.FileOrganization) bool
},
) *pipeline.Error {
	return setPesitInfo(pip, organizationKey, func(str string) bool {
		var org pesit.FileOrganization

		switch str {
		case pesit.OrgSequential.String():
			org = pesit.OrgSequential
		case pesit.OrgRelative.String():
			org = pesit.OrgRelative
		case pesit.OrgIndexed.String():
			org = pesit.OrgIndexed
		default:
			return false
		}

		return f.SetFileOrganization(org)
	})
}

func setCustomerID(pip *pipeline.Pipeline, f interface {
	SetCustomerID(customerID string) bool
},
) *pipeline.Error {
	return setPesitInfo(pip, customerIDKey, f.SetCustomerID)
}

func setBankID(pip *pipeline.Pipeline, f interface {
	SetBankID(bankID string) bool
},
) *pipeline.Error {
	return setPesitInfo(pip, bankIDKey, f.SetBankID)
}

func setFreetext(pip *pipeline.Pipeline, key string, f interface {
	SetFreeText(freetext string) bool
},
) *pipeline.Error {
	return setPesitInfo(pip, key, f.SetFreeText)
}

func setTransInfo[T cmp.Ordered](pip *pipeline.Pipeline, key string, val T) {
	if !reflect.ValueOf(val).IsZero() {
		pip.TransCtx.Transfer.TransferInfo[key] = val
	}
}

// getArticlesLengths returns the lengths of the articles declared in the
// transfer info, nil when there are none.
func getArticlesLengths(pip *pipeline.Pipeline) *articleLengths {
	lengths, err := utils.GetAs[*articleLengths](pip.TransCtx.Transfer.TransferInfo, articlesLengthsKey)
	if err != nil || lengths.Len() == 0 {
		return nil
	}

	return lengths
}

func isMultiArticles(pip *pipeline.Pipeline) (*articleLengths, bool) {
	lengths := getArticlesLengths(pip)

	return lengths, lengths != nil
}

func getArticlesFormat(pip *pipeline.Pipeline) pesit.ArticleFormat {
	val, err := utils.GetAs[string](pip.TransCtx.Transfer.TransferInfo, articlesFormatKey)
	if err != nil {
		return defaultArticleFormat
	}

	switch val {
	case pesit.FormatFixed.String():
		return pesit.FormatFixed
	case pesit.FormatVariable.String():
		return pesit.FormatVariable
	default:
		return defaultArticleFormat
	}
}

func getArticlesSize(pip *pipeline.Pipeline) uint16 {
	lengths := getArticlesLengths(pip)
	if lengths == nil {
		return defaultArticleSize
	}

	return lengths.Max()
}

func addArticleFormat(pip *pipeline.Pipeline, f interface {
	ArticleFormat() pesit.ArticleFormat
},
) {
	pip.TransCtx.Transfer.TransferInfo[articlesFormatKey] = f.ArticleFormat().String()
}

// getArticlesSeparator returns the separator ending each record of the file
// to send when the transfer info declares one (a delimited text file), nil
// otherwise.
func getArticlesSeparator(pip *pipeline.Pipeline) ([]byte, *pipeline.Error) {
	name, err := utils.GetAs[string](pip.TransCtx.Transfer.TransferInfo, articlesSeparatorKey)
	if errors.Is(err, utils.ErrKeyNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, pipeline.NewError(types.TeInternal, err.Error())
	}

	if name == "" {
		return nil, nil
	}

	separator, parseErr := parseArticlesSeparator(name)
	if parseErr != nil {
		return nil, pipeline.NewError(types.TeInternal, parseErr.Error())
	}

	return separator, nil
}

// getSendArticlesSize returns the article size (PI 32) to announce for a file
// to send: the longest configured length or, for a delimited text file
// without configured lengths, the longest record of the file. The file is
// read through the given stream when there is one (it is then left at its
// position), or opened from its path otherwise.
func getSendArticlesSize(pip *pipeline.Pipeline, file io.ReadSeeker) (uint16, *pipeline.Error) {
	separator, sepErr := getArticlesSeparator(pip)
	if sepErr != nil {
		return 0, sepErr
	}

	if _, hasLengths := isMultiArticles(pip); separator == nil || hasLengths ||
		getArticlesFormat(pip) != pesit.FormatVariable {
		return getArticlesSize(pip), nil
	}

	var (
		size uint16
		err  error
	)

	if file != nil {
		size, err = longestRecord(file, separator)
	} else {
		size, err = longestRecordOf(pip.TransCtx.Transfer.LocalPath, separator)
	}

	if err != nil {
		pip.Logger.Errorf("Failed to measure the records of the file: %v", err)

		return 0, pipeline.NewError(types.TeInternal, "failed to measure the records of the file")
	}

	return size, nil
}
