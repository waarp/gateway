package pesit

import (
	"reflect"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
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
)

type valueTypes interface {
	string |
		int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 |
		float32 | float64
}

func setPesitInfo[T valueTypes, F ~func(T) bool](pip *pipeline.Pipeline, key string, set F) *pipeline.Error {
	valAny, hasKey := pip.TransCtx.TransInfo[key]
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
		pip.TransCtx.TransInfo[key] = val
	}
}
