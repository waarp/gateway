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

	EncryptAES          = "ENCRYPT-AES"
	DecryptAES          = "DECRYPT-AES"
	EncryptPGP          = "ENCRYPT-PGP"
	DecryptPGP          = "DECRYPT-PGP"
	EncryptAndSignPGP   = "ENCRYPT&SIGN-PGP"
	DecryptAndVerifyPGP = "DECRYPT&VERIFY-PGP"
	SignHMAC            = "SIGN-HMAC"
	VerifyHMAC          = "VERIFY-HMAC"
	SignPGP             = "SIGN-PGP"
	VerifyPGP           = "VERIFY-PGP"
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
	model.ValidTasks[EncryptAES] = &encryptAES{}
	model.ValidTasks[DecryptAES] = &decryptAES{}
	model.ValidTasks[EncryptPGP] = &encryptPGP{}
	model.ValidTasks[DecryptPGP] = &decryptPGP{}
	model.ValidTasks[EncryptAndSignPGP] = &encryptSignPGP{}
	model.ValidTasks[DecryptAndVerifyPGP] = &decryptVerifyPGP{}
	model.ValidTasks[SignHMAC] = &signHMAC{}
	model.ValidTasks[VerifyHMAC] = &verifyHMAC{}
	model.ValidTasks[SignPGP] = &signPGP{}
	model.ValidTasks[VerifyPGP] = &verifyPGP{}
}
