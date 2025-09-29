package gui

import "code.waarp.fr/apps/gateway/gateway/pkg/tasks"

//nolint:gochecknoglobals // Constant
var TaskTypes = []string{
	tasks.Copy,
	tasks.CopyRename,
	tasks.Delete,
	tasks.Exec,
	tasks.ExecMove,
	tasks.ExecOutput,
	tasks.Move,
	tasks.MoveRename,
	tasks.Rename,
	tasks.Transfer,
	tasks.Transcode,
	tasks.Archive,
	tasks.Extract,
	tasks.Icap,
	tasks.Email,
	tasks.Encrypt,
	tasks.Decrypt,
	tasks.Sign,
	tasks.Verify,
	tasks.EncryptAndSign,
	tasks.DecryptAndVerify,
}
