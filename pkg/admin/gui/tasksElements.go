package gui

import (
	"net/http"
	"slices"
	"strings"

	"golang.org/x/exp/maps" //nolint:exptostd // does not work when I put only "maps"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

const (
	TaskCopy          = "COPY"
	TaskCopyRename    = "COPYRENAME"
	TaskDelete        = "DELETE"
	TaskExec          = "EXEC"
	TaskExecMove      = "EXECMOVE"
	TaskExecOutput    = "EXECOUTPUT"
	TaskMove          = "MOVE"
	TaskMoveRename    = "MOVERENAME"
	TaskRename        = "RENAME"
	TaskTransfer      = "TRANSFER"
	TaskTranscode     = "TRANSCODE"
	TaskArchive       = "ARCHIVE"
	TaskExtract       = "EXTRACT"
	TaskIcap          = "ICAP"
	TaskEncrypt       = "ENCRYPT"
	TaskDecrypt       = "DECRYPT"
	TaskSign          = "SIGN"
	TaskVerify        = "VERIFY"
	TaskEncryptSign   = "ENCRYPT&SIGN"
	TaskDecryptVerify = "DECRYPT&VERIFY"
)

//nolint:gochecknoglobals // Constant
var (
	TranscodeFormats      []string
	ArchiveExtensions     []string
	EncryptMethods        []string
	EncryptKeyTypes       map[string][]string
	SignMethods           []string
	SignKeyTypes          map[string][]string
	EncryptSignMethods    []string
	EncryptSignKeyTypes   map[string]map[string][]string
	DecryptVerifyMethods  []string
	DecryptVerifyKeyTypes map[string]map[string][]string
	DecryptMethods        []string
	DecryptKeyTypes       map[string][]string
	VerifyMethods         []string
	VerifyKeyTypes        map[string][]string
	IcapOnErrorOptions    = []string{
		"",
		tasks.IcapOnErrorDelete,
		tasks.IcapOnErrorMove,
	}
	CompressionLevelList = []string{
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	}
	ListAesKeyName     []string
	ListHmacKeyName    []string
	ListPgpPubKeyName  []string
	ListPgpPrivKeyName []string
)

//nolint:gochecknoinits, exptostd, gocritic // to initialie map needed
func init() {
	TranscodeFormats = maps.Keys(tasks.TranscodeFormats)
	slices.Sort(TranscodeFormats)

	ArchiveExtensions = maps.Keys(tasks.ArchiveExtensions)
	slices.Sort(ArchiveExtensions)

	EncryptMethods = maps.Keys(tasks.EncryptMethods)
	slices.Sort(EncryptMethods)

	EncryptKeyTypes = make(map[string][]string, len(tasks.EncryptMethods))
	for m, tab := range tasks.EncryptMethods {
		EncryptKeyTypes[m] = tab.KeyTypes
	}

	SignMethods = maps.Keys(tasks.SignMethods)
	slices.Sort(SignMethods)

	SignKeyTypes = make(map[string][]string, len(tasks.SignMethods))
	for m, tab := range tasks.SignMethods {
		SignKeyTypes[m] = tab.KeyTypes
	}

	DecryptMethods = maps.Keys(tasks.DecryptMethods)
	slices.Sort(DecryptMethods)

	DecryptKeyTypes = make(map[string][]string, len(tasks.DecryptMethods))
	for m, tab := range tasks.DecryptMethods {
		DecryptKeyTypes[m] = tab.KeyTypes
	}

	VerifyMethods = maps.Keys(tasks.VerifyMethods)
	slices.Sort(VerifyMethods)

	VerifyKeyTypes = make(map[string][]string, len(tasks.VerifyMethods))
	for m, tab := range tasks.VerifyMethods {
		VerifyKeyTypes[m] = tab.KeyTypes
	}

	EncryptSignMethods = maps.Keys(tasks.EncryptSignMethods)
	slices.Sort(EncryptSignMethods)

	EncryptSignKeyTypes = make(map[string]map[string][]string, len(tasks.EncryptSignMethods))
	for m, tab := range tasks.EncryptSignMethods {
		EncryptSignKeyTypes[m] = map[string][]string{
			"encrypt": tab.KeyTypesEncrypt,
			"sign":    tab.KeyTypesSign,
		}
	}

	DecryptVerifyMethods = maps.Keys(tasks.EncryptSignMethods)
	slices.Sort(DecryptVerifyMethods)

	DecryptVerifyKeyTypes = make(map[string]map[string][]string, len(tasks.EncryptSignMethods))
	for m, tab := range tasks.EncryptSignMethods {
		DecryptVerifyKeyTypes[m] = map[string][]string{
			"decrypt": tab.KeyTypesEncrypt,
			"verify":  tab.KeyTypesSign,
		}
	}
}

func listKeyname(db *database.DB) {
	listAesKey, err := internal.ListCryptoKeys(db, "name", true, 0, 0, "AES")
	if err == nil {
		ListAesKeyName = make([]string, 0, len(listAesKey))
		for _, key := range listAesKey {
			ListAesKeyName = append(ListAesKeyName, key.Name)
		}
	}

	listHmacKey, err := internal.ListCryptoKeys(db, "name", true, 0, 0, "HMAC")
	if err == nil {
		ListHmacKeyName = make([]string, 0, len(listHmacKey))
		for _, key := range listHmacKey {
			ListHmacKeyName = append(ListHmacKeyName, key.Name)
		}
	}

	listPgpPubKey, err := internal.ListCryptoKeys(db, "name", true, 0, 0, "PGP-PUBLIC")
	if err == nil {
		ListPgpPubKeyName = make([]string, 0, len(listPgpPubKey))
		for _, key := range listPgpPubKey {
			ListPgpPubKeyName = append(ListPgpPubKeyName, key.Name)
		}
	}

	listPgpPrivKey, err := internal.ListCryptoKeys(db, "name", true, 0, 0, "PGP-PRIVATE")
	if err == nil {
		ListPgpPrivKeyName = make([]string, 0, len(listPgpPrivKey))
		for _, key := range listPgpPrivKey {
			ListPgpPrivKeyName = append(ListPgpPrivKeyName, key.Name)
		}
	}
}

//nolint:gochecknoglobals // Constant
var TaskTypes = []string{
	TaskCopy,
	TaskCopyRename,
	TaskDelete,
	TaskExec,
	TaskExecMove,
	TaskExecOutput,
	TaskMove,
	TaskMoveRename,
	TaskRename,
	TaskTransfer,
	TaskTranscode,
	TaskArchive,
	TaskExtract,
	TaskIcap,
	TaskEncrypt,
	TaskDecrypt,
	TaskSign,
	TaskVerify,
	TaskEncryptSign,
	TaskDecryptVerify,
}

func taskCOPY(r *http.Request) map[string]string {
	taskCopyMap := make(map[string]string)

	if pathCopy := r.FormValue("pathCopy"); pathCopy != "" {
		taskCopyMap["path"] = pathCopy
	}

	return taskCopyMap
}

func taskCOPYRENAME(r *http.Request) map[string]string {
	taskCopyRenameMap := make(map[string]string)

	if pathCopyRename := r.FormValue("pathCopyRename"); pathCopyRename != "" {
		taskCopyRenameMap["path"] = pathCopyRename
	}

	return taskCopyRenameMap
}

func taskEXEC(r *http.Request) map[string]string {
	taskExec := make(map[string]string)

	if pathExec := r.FormValue("pathExec"); pathExec != "" {
		taskExec["path"] = pathExec
	}

	if argsExec := r.FormValue("argsExec"); argsExec != "" {
		taskExec["args"] = argsExec
	}

	if delayExec := r.FormValue("delayExec"); delayExec != "" {
		taskExec["delay"] = delayExec
	}

	return taskExec
}

func taskEXECMOVE(r *http.Request) map[string]string {
	taskExecMove := make(map[string]string)

	if pathExecMove := r.FormValue("pathExecMove"); pathExecMove != "" {
		taskExecMove["path"] = pathExecMove
	}

	if argsExecMove := r.FormValue("argsExecMove"); argsExecMove != "" {
		taskExecMove["args"] = argsExecMove
	}

	if delayExecMove := r.FormValue("delayExecMove"); delayExecMove != "" {
		taskExecMove["delay"] = delayExecMove
	}

	return taskExecMove
}

func taskEXECOUTPUT(r *http.Request) map[string]string {
	taskExecOutput := make(map[string]string)

	if pathExecOutput := r.FormValue("pathExecOutput"); pathExecOutput != "" {
		taskExecOutput["path"] = pathExecOutput
	}

	if argsExecOutput := r.FormValue("argsExecOutput"); argsExecOutput != "" {
		taskExecOutput["args"] = argsExecOutput
	}

	if delayExecOutput := r.FormValue("delayExecOutput"); delayExecOutput != "" {
		taskExecOutput["delay"] = delayExecOutput
	}

	return taskExecOutput
}

func taskMOVE(r *http.Request) map[string]string {
	taskMove := make(map[string]string)

	if pathMove := r.FormValue("pathMove"); pathMove != "" {
		taskMove["path"] = pathMove
	}

	return taskMove
}

func taskMOVERENAME(r *http.Request) map[string]string {
	taskMoveRename := make(map[string]string)

	if pathMoveRename := r.FormValue("pathMoveRename"); pathMoveRename != "" {
		taskMoveRename["path"] = pathMoveRename
	}

	return taskMoveRename
}

func taskRENAME(r *http.Request) map[string]string {
	taskRename := make(map[string]string)

	if pathRename := r.FormValue("pathRename"); pathRename != "" {
		taskRename["path"] = pathRename
	}

	return taskRename
}

func taskTRANSFER(r *http.Request) map[string]string {
	taskTransfer := make(map[string]string)

	if fileTransfer := r.FormValue("fileTransfer"); fileTransfer != "" {
		taskTransfer["file"] = fileTransfer
	}

	if usingTransfer := r.FormValue("usingTransfer"); usingTransfer != "" {
		taskTransfer["using"] = usingTransfer
	}

	if toTransfer := r.FormValue("toTransfer"); toTransfer != "" {
		taskTransfer["to"] = toTransfer
	}

	if asTransfer := r.FormValue("asTransfer"); asTransfer != "" {
		taskTransfer["as"] = asTransfer
	}

	if ruleTransfer := r.FormValue("ruleTransfer"); ruleTransfer != "" {
		taskTransfer["rule"] = ruleTransfer
	}

	if copyInfoTransfer := r.FormValue("copyInfoTransfer"); copyInfoTransfer != "" {
		taskTransfer["copyInfo"] = copyInfoTransfer
	}

	infoKeys := r.Form["infoTransferKey[]"]
	infoVals := r.Form["infoTransferValue[]"]

	var infoPairs []string

	for i := range infoKeys {
		if infoKeys[i] != "" && i < len(infoVals) && infoVals[i] != "" {
			infoPairs = append(infoPairs, infoKeys[i]+":"+infoVals[i])
		}
	}

	if len(infoPairs) > 0 {
		taskTransfer["info"] = strings.Join(infoPairs, "\n")
	}

	return taskTransfer
}

func taskTRANSCODE(r *http.Request) map[string]string {
	taskTranscode := make(map[string]string)

	if fromCharsetTranscode := r.FormValue("fromCharsetTranscode"); fromCharsetTranscode != "" {
		taskTranscode["fromCharset"] = fromCharsetTranscode
	}

	if toCharsetTranscode := r.FormValue("toCharsetTranscode"); toCharsetTranscode != "" {
		taskTranscode["toCharset"] = toCharsetTranscode
	}

	return taskTranscode
}

func taskARCHIVE(r *http.Request) map[string]string {
	taskArchive := make(map[string]string)

	if filesArchive := r.FormValue("filesArchive"); filesArchive != "" {
		taskArchive["files"] = filesArchive
	}

	if compressionLevelArchive := r.FormValue("compressionLevelArchive"); compressionLevelArchive != "" {
		taskArchive["compressionLevel"] = compressionLevelArchive
	}

	outputPathArchiveName := r.FormValue("outputPathArchiveName")

	if outputPathArchiveExt := r.FormValue("outputPathArchiveExt"); outputPathArchiveExt != "" &&
		outputPathArchiveName != "" {
		taskArchive["outputPath"] = outputPathArchiveName + outputPathArchiveExt
	}

	return taskArchive
}

func taskEXTRACT(r *http.Request) map[string]string {
	taskExtract := make(map[string]string)

	archiveExtractName := r.FormValue("archiveExtractName")

	if archiveExtractExt := r.FormValue("archiveExtractExt"); archiveExtractExt != "" && archiveExtractName != "" {
		taskExtract["archive"] = archiveExtractName + archiveExtractExt
	}

	if outputDirExtract := r.FormValue("outputDirExtract"); outputDirExtract != "" {
		taskExtract["outputDir"] = outputDirExtract
	}

	return taskExtract
}

//nolint:dupl // method for task Icap
func taskICAP(r *http.Request) map[string]string {
	taskIcap := make(map[string]string)

	if uploadURLIcap := r.FormValue("uploadURLIcap"); uploadURLIcap != "" {
		taskIcap["uploadURL"] = uploadURLIcap
	}
	timeout := ""

	if h := r.FormValue("timeoutIcapH"); h != "" {
		timeout += h + "h"
	}

	if m := r.FormValue("timeoutIcapM"); m != "" {
		timeout += m + "m"
	}

	if s := r.FormValue("timeoutIcapS"); s != "" {
		timeout += s + "s"
	}

	if ms := r.FormValue("timeoutIcapMS"); ms != "" {
		timeout += ms + "ms"
	}

	if timeout != "" {
		taskIcap["timeout"] = timeout
	}

	if allowFileModificationsIcap := r.FormValue("allowFileModificationsIcap"); allowFileModificationsIcap != "" {
		taskIcap["allowFileModifications"] = allowFileModificationsIcap
	}

	if onErrorIcap := r.FormValue("onErrorIcap"); onErrorIcap != "" {
		taskIcap["onError"] = onErrorIcap
	}

	if onErrorMovePathIcap := r.FormValue("onErrorMovePathIcap"); onErrorMovePathIcap != "" {
		taskIcap["onErrorMovePath"] = onErrorMovePathIcap
	}

	return taskIcap
}

func taskENCRYPT(r *http.Request) map[string]string {
	taskEncrypt := make(map[string]string)

	if outputFileEncrypt := r.FormValue("outputFileEncrypt"); outputFileEncrypt != "" {
		taskEncrypt["outputFile"] = outputFileEncrypt
	}

	if keepOriginalEncrypt := r.FormValue("keepOriginalEncrypt"); keepOriginalEncrypt != "" {
		taskEncrypt["keepOriginal"] = keepOriginalEncrypt
	}

	if methodEncrypt := r.FormValue("methodEncrypt"); methodEncrypt != "" {
		taskEncrypt["method"] = methodEncrypt
	}

	if keyNameEncrypt := r.FormValue("keyNameEncrypt"); keyNameEncrypt != "" {
		taskEncrypt["keyName"] = keyNameEncrypt
	}

	return taskEncrypt
}

func taskDECRYPT(r *http.Request) map[string]string {
	taskDecrypt := make(map[string]string)

	if outputFileDecrypt := r.FormValue("outputFileDecrypt"); outputFileDecrypt != "" {
		taskDecrypt["outputFile"] = outputFileDecrypt
	}

	if keepOriginalDecrypt := r.FormValue("keepOriginalDecrypt"); keepOriginalDecrypt != "" {
		taskDecrypt["keepOriginal"] = keepOriginalDecrypt
	}

	if methodDecrypt := r.FormValue("methodDecrypt"); methodDecrypt != "" {
		taskDecrypt["method"] = methodDecrypt
	}

	if keyNameDecrypt := r.FormValue("keyNameDecrypt"); keyNameDecrypt != "" {
		taskDecrypt["keyName"] = keyNameDecrypt
	}

	return taskDecrypt
}

func taskSIGN(r *http.Request) map[string]string {
	taskSign := make(map[string]string)

	if outputFileSign := r.FormValue("outputFileSign"); outputFileSign != "" {
		taskSign["outputFile"] = outputFileSign
	}

	if methodSign := r.FormValue("methodSign"); methodSign != "" {
		taskSign["method"] = methodSign
	}

	if keyNameSign := r.FormValue("keyNameSign"); keyNameSign != "" {
		taskSign["keyName"] = keyNameSign
	}

	return taskSign
}

func taskVERIFY(r *http.Request) map[string]string {
	taskVerify := make(map[string]string)

	if signatureFileVerify := r.FormValue("signatureFileVerify"); signatureFileVerify != "" {
		taskVerify["signatureFile"] = signatureFileVerify
	}

	if methodVerify := r.FormValue("methodVerify"); methodVerify != "" {
		taskVerify["method"] = methodVerify
	}

	if keyNameVerify := r.FormValue("keyNameVerify"); keyNameVerify != "" {
		taskVerify["keyName"] = keyNameVerify
	}

	return taskVerify
}

//nolint:dupl // method for task encrypt and sign
func taskENCRYPTandSIGN(r *http.Request) map[string]string {
	taskES := make(map[string]string)

	if outputFileES := r.FormValue("outputFileEncrypt&Sign"); outputFileES != "" {
		taskES["outputFile"] = outputFileES
	}

	if keepOriginalES := r.FormValue("keepOriginalEncrypt&Sign"); keepOriginalES != "" {
		taskES["keepOriginal"] = keepOriginalES
	}

	if methodES := r.FormValue("methodEncrypt&Sign"); methodES != "" {
		taskES["method"] = methodES
	}

	if encryptKeyNameES := r.FormValue("encryptKeyNameEncrypt&Sign"); encryptKeyNameES != "" {
		taskES["encryptKeyName"] = encryptKeyNameES
	}

	if signKeyNameES := r.FormValue("signKeyNameEncrypt&Sign"); signKeyNameES != "" {
		taskES["signKeyName"] = signKeyNameES
	}

	return taskES
}

//nolint:dupl // method for task decrypt and verify
func taskDECRYPTandVERIFY(r *http.Request) map[string]string {
	taskDV := make(map[string]string)

	if outputFileDV := r.FormValue("outputFileDecrypt&verify"); outputFileDV != "" {
		taskDV["outputFile"] = outputFileDV
	}

	if keepOriginalDV := r.FormValue("keepOriginalDecrypt&verify"); keepOriginalDV != "" {
		taskDV["keepOriginal"] = keepOriginalDV
	}

	if methodDV := r.FormValue("methodDecrypt&verify"); methodDV != "" {
		taskDV["method"] = methodDV
	}

	if decryptKeyNameDV := r.FormValue("decryptKeyNameDecrypt&verify"); decryptKeyNameDV != "" {
		taskDV["decryptKeyName"] = decryptKeyNameDV
	}

	if verifyKeyNameDV := r.FormValue("verifyKeyNameDecrypt&verify"); verifyKeyNameDV != "" {
		taskDV["verifyKeyName"] = verifyKeyNameDV
	}

	return taskDV
}
