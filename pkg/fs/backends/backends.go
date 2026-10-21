package backends

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/azure"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/gcs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/onedrive"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/s3"
)

//nolint:gochecknoinits //init is used by design
func init() {
	register(s3.Storage, s3.NewFS, "s3", "aws")
	register(azure.BlobStorage, azure.NewBlobFS, "azblob", "azureblob")
	register(azure.FilesStorage, azure.NewFilesFS, "azfiles", "azurefiles")
	register(gcs.Storage, gcs.NewFS, "gcs", "google")
	register(onedrive.Storage, onedrive.NewFS, "onedrive", "sharepoint")
}

// register records a backend under its type and its aliases, with the
// storage of rclone it is built on.
func register(storage string, mkfs fs.FSMaker, kinds ...string) {
	for _, kind := range kinds {
		fs.Register(kind, mkfs)
		fs.Describe(kind, fs.Library{Name: storage})
	}
}
