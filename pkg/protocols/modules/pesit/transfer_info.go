package pesit

import (
	"reflect"
	"strings"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	fileEncodingKey = "__fileEncoding__"
	fileTypeKey     = "__fileType__"
	organizationKey = "__organization__"
	customerIDKey   = "__customerID__"
	bankIDKey       = "__bankID__"

	clientConnFreetextKey  = "__clientConnFreetext__"
	clientTransFreetextKey = "__clientTransFreetext__"
	serverConnFreetextKey  = "__serverConnFreetext__"
	serverTransFreetextKey = "__serverTransFreetext__"

	articlesLengthsKey = "__articlesLengths__"
	articleFormatKey   = "__articleFormat__"

	// Store & Forward reply info keys, extracted from PI 99 freetext
	// convention "REPLY=partner:account".
	replyPartnerKey = "__replyPartner__"
	replyAccountKey = "__replyAccount__"
)

type valueTypes interface {
	string |
		int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 |
		float32 | float64
}

func setPesitInfo[T valueTypes, F ~func(T) bool](pip *pipeline.Pipeline, key string, set F) *pipeline.Error {
	valAny, hasKey := pip.TransCtx.Transfer.TransferInfo[key]
	if !hasKey {
		return nil
	}

	val, isStr := valAny.(T)
	if !isStr {
		return pipeline.NewErrorf(types.TeInternal,
			"freetext variable %q must be a %T (was of type %T)", key, val, valAny)
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

func setTransInfo[T valueTypes](pip *pipeline.Pipeline, key string, val T) {
	if !reflect.ValueOf(val).IsZero() {
		pip.TransCtx.Transfer.TransferInfo[key] = val
	}
}

// getArticleFormat returns the article format for a transfer, checking
// TransferInfo override first, then falling back to the config value.
func getArticleFormat(pip *pipeline.Pipeline, configValue string) pesit.ArticleFormat {
	if val, ok := pip.TransCtx.Transfer.TransferInfo[articleFormatKey]; ok {
		if str, isStr := val.(string); isStr {
			return resolveArticleFormat(str)
		}
	}

	return resolveArticleFormat(configValue)
}

// parseReplyInfo extracts Store & Forward reply info from a PI 99 freetext
// field. Convention: "REPLY=partner:account" (or "REPLY=partner" with the
// account defaulting to first available on the partner).
//
// This is the single mechanism for all PeSIT emitters (Waarp or third-party)
// to specify where the ACK F.MESSAGE should be sent back. The last PI 99
// containing REPLY= wins (transfer freetext takes priority over connection).
func parseReplyInfo(pip *pipeline.Pipeline, freetext string) {
	if freetext == "" {
		return
	}

	const prefix = "REPLY="

	idx := strings.Index(freetext, prefix)
	if idx < 0 {
		return
	}

	value := freetext[idx+len(prefix):]

	// Trim at first space, comma, or semicolon (PI 99 may contain other data).
	for _, sep := range []string{" ", ",", ";"} {
		if i := strings.Index(value, sep); i >= 0 {
			value = value[:i]
		}
	}

	parts := strings.SplitN(value, ":", 2)
	if len(parts) >= 1 && parts[0] != "" {
		pip.TransCtx.Transfer.TransferInfo[replyPartnerKey] = parts[0]
	}

	if len(parts) >= 2 && parts[1] != "" { //nolint:mnd // index 1 is the account
		pip.TransCtx.Transfer.TransferInfo[replyAccountKey] = parts[1]
	}
}

func isMultiArticles(pip *pipeline.Pipeline) ([]int64, bool) {
	vals, err := utils.GetAs[[]int64](pip.TransCtx.Transfer.TransferInfo, articlesLengthsKey)
	if err != nil {
		return nil, false
	}

	return vals, len(vals) > 0
}
