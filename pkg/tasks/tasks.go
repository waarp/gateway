package tasks

import (
	"reflect"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	Copy       = "COPY"
	CopyRename = "COPYRENAME"
	Move       = "MOVE"
	MoveRename = "MOVERENAME"
	Delete     = "DELETE"

	Exec       = "EXEC"
	ExecMove   = "EXECMOVE"
	ExecOutput = "EXECOUTPUT"

	Rename      = "RENAME"
	Transfer    = "TRANSFER"
	Preregister = "PREREGISTER"

	Encrypt          = "ENCRYPT"
	Decrypt          = "DECRYPT"
	Sign             = "SIGN"
	Verify           = "VERIFY"
	EncryptAndSign   = "ENCRYPT&SIGN"
	DecryptAndVerify = "DECRYPT&VERIFY"

	Archive = "ARCHIVE"
	Extract = "EXTRACT"

	Transcode     = "TRANSCODE"
	ChangeNewline = "CHNEWLINE"

	Icap  = "ICAP"
	Email = "EMAIL"
)

//nolint:gochecknoinits //init is required here
func init() {
	// File operations
	model.ValidTasks[Copy] = newRunner[*copyTask]
	model.ValidTasks[CopyRename] = newRunner[*copyRenameTask]
	model.ValidTasks[Move] = newRunner[*moveTask]
	model.ValidTasks[MoveRename] = newRunner[*moveRenameTask]
	model.ValidTasks[Delete] = newRunner[*deleteTask]

	// Execution tasks
	model.ValidTasks[Exec] = newRunner[*execTask]
	model.ValidTasks[ExecMove] = newRunner[*execMoveTask]
	model.ValidTasks[ExecOutput] = newRunner[*execOutputTask]

	// Transfer tasks
	model.ValidTasks[Rename] = newRunner[*renameTask] // "RENAME" is in fact a "change target" task
	model.ValidTasks[Transfer] = newRunner[*TransferTask]
	model.ValidTasks[Preregister] = newRunner[*TransferPreregister]

	// File encryption & signing
	model.ValidTasks[Encrypt] = newRunner[*encrypt]
	model.ValidTasks[Decrypt] = newRunner[*decrypt]
	model.ValidTasks[Sign] = newRunner[*sign]
	model.ValidTasks[Verify] = newRunner[*verify]
	model.ValidTasks[EncryptAndSign] = newRunner[*encryptSign]
	model.ValidTasks[DecryptAndVerify] = newRunner[*decryptVerify]

	// Archiving & compression
	model.ValidTasks[Archive] = newRunner[*archiveTask]
	model.ValidTasks[Extract] = newRunner[*extractTask]

	// Content manipulation
	model.ValidTasks[Transcode] = newRunner[*transcodeTask]
	model.ValidTasks[ChangeNewline] = newRunner[*chNewlineTask]

	// Network
	model.ValidTasks[Icap] = newRunner[*icapTask]
	model.ValidTasks[Email] = newRunner[*emailTask]
}

func newRunner[T model.TaskRunner]() model.TaskRunner {
	typ := reflect.TypeFor[T]().Elem()

	//nolint:errcheck,forcetypeassert //assertion cannot fail here, no need to check
	return reflect.New(typ).Interface().(model.TaskRunner)
}
