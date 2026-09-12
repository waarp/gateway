package pesit

import (
	"cmp"
	"errors"
	"io"
	"reflect"
	"slices"

	"code.waarp.fr/lib/pesit"

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
	ackSentKey       = tasks.SendMessageAckSentKey
	ackSentOnKey     = tasks.SendMessageAckSentOnKey
	ackReceivedKey   = "__ackReceived__"
	ackReceivedOnKey = "__ackReceivedOn__"

	clientConnFreetextKey  = "__clientConnFreetext__"
	clientTransFreetextKey = "__clientTransFreetext__"
	serverConnFreetextKey  = "__serverConnFreetext__"
	serverTransFreetextKey = "__serverTransFreetext__"

	articlesLengthsKey   = "__articlesLengths__"
	articlesFormatKey    = "__articlesFormat__"
	articlesSeparatorKey = "__articlesSeparator__"
)

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

func isMultiArticles(pip *pipeline.Pipeline) ([]uint16, bool) {
	vals, err := utils.GetAs[[]uint16](pip.TransCtx.Transfer.TransferInfo, articlesLengthsKey)
	if err != nil {
		return nil, false
	}

	return vals, len(vals) > 0
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
	vals, err := utils.GetAs[[]uint16](pip.TransCtx.Transfer.TransferInfo, articlesLengthsKey)
	if err != nil {
		return defaultArticleSize
	}

	return slices.Max(vals)
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
