package tasks

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

const (
	Copy       = "COPY"
	CopyRename = "COPYRENAME"
	Move       = "MOVE"
	MoveRename = "MOVERENAME"
	Delete     = "DELETE"

	Exec       = "EXEC"
	ExecMove   = "EXECMOVE"
	ExecOutput = "EXECOUTPUT"

	Rename   = "RENAME"
	Transfer = "TRANSFER"

	Encrypt          = "ENCRYPT"
	Decrypt          = "DECRYPT"
	Sign             = "SIGN"
	Verify           = "VERIFY"
	EncryptAndSign   = "ENCRYPT&SIGN"
	DecryptAndVerify = "DECRYPT&VERIFY"
)

//nolint:gochecknoinits //init is required here
func init() {
	// File operations
	model.ValidTasks[Copy] = &copyTask{}
	model.ValidTasks[CopyRename] = &copyRenameTask{}
	model.ValidTasks[Move] = &moveTask{}
	model.ValidTasks[MoveRename] = &moveRenameTask{}
	model.ValidTasks[Delete] = &deleteTask{}

	// Execution tasks
	model.ValidTasks[Exec] = &execTask{}
	model.ValidTasks[ExecMove] = &execMoveTask{}
	model.ValidTasks[ExecOutput] = &execOutputTask{}

	// Transfer tasks
	model.ValidTasks[Rename] = &renameTask{} // "RENAME" is in fact a "change target" task
	model.ValidTasks[Transfer] = &TransferTask{}

	// File encryption & signing
	model.ValidTasks[Encrypt] = &encrypt{}
	model.ValidTasks[Decrypt] = &decrypt{}
	model.ValidTasks[Sign] = &sign{}
	model.ValidTasks[Verify] = &verify{}
	model.ValidTasks[EncryptAndSign] = &encryptSign{}
	model.ValidTasks[DecryptAndVerify] = &decryptVerify{}
}
