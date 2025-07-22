package gui

import (
	"net/http"
)

//nolint:gochecknoglobals // Constant
var SupportedTranscode = []string{
	"UTF-8",
	"UTF-8 BOM",
	"UTF-16BE",
	"UTF-16LE",
	"UTF-16BE BOM",
	"UTF-16LE BOM",
	"UTF-32BE",
	"UTF-32LE",
	"UTF-32BE BOM",
	"UTF-32LE BOM",
	"IBM Code Page 037",
	"IBM Code Page 273",
	"IBM Code Page 437",
	"IBM Code Page 500",
	"IBM Code Page 850",
	"IBM Code Page 852",
	"IBM Code Page 855",
	"IBM Code Page 858",
	"IBM Code Page 860",
	"IBM Code Page 862",
	"IBM Code Page 863",
	"IBM Code Page 865",
	"IBM Code Page 866",
	"IBM Code Page 1047",
	"IBM Code Page 1140",
	"IBM Code Page 1141",
	"IBM Code Page 1148",
	"ISO 8859-1",
	"ISO 8859-2",
	"ISO 8859-3",
	"ISO 8859-4",
	"ISO 8859-5",
	"ISO 8859-6",
	"ISO 8859-6E",
	"ISO 8859-6I",
	"ISO 8859-7",
	"ISO 8859-8",
	"ISO 8859-8E",
	"ISO 8859-8I",
	"ISO 8859-9",
	"ISO 8859-10",
	"ISO 8859-13",
	"ISO 8859-14",
	"ISO 8859-15",
	"ISO 8859-16",
	"KOI8-R",
	"KOI8-U",
	"Macintosh",
	"Macintosh Cyrillic",
	"Windows 874",
	"Windows 1250",
	"Windows 1251",
	"Windows 1252",
	"Windows 1253",
	"Windows 1254",
	"Windows 1255",
	"Windows 1256",
	"Windows 1257",
	"Windows 1258",
	"X-User-Defined",
}

//nolint:gochecknoglobals // Constant
var TaskTypes = []string{
	"COPY",
	"COPYRENAME",
	"DELETE",
	"EXEC",
	"EXECMOVE",
	"EXECOUTPUT",
	"MOVE",
	"MOVERENAME",
	"RENAME",
	"TRANSFER",
	"TRANSCODE",
	"ARCHIVE",
	"EXTRACT",
	"ICAP (BETA)",
	"ENCRYPT",
	"DECRYPT",
	"SIGN",
	"VERIFY",
	"ENCRYPT&SIGN",
	"DECRYPT&VERIFY",
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

	if infoTransfer := r.FormValue("infoTransfer"); infoTransfer != "" {
		taskTransfer["info"] = infoTransfer
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
