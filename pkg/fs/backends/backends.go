package backends

import (
	"fmt"

	rfs "github.com/rclone/rclone/fs"

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
func register(kind string, mkfs fs.FSMaker, aliases ...string) {
	regInfo, err := rfs.Find(kind)
	if err != nil {
		panic(fmt.Sprintf("unknown rclone backend %q", kind))
	}

	backend := fs.Backend{
		Type:    kind,
		Make:    mkfs,
		Options: regInfo.Options,
	}

	for _, alias := range aliases {
		fs.Register(alias, backend)
	}
}
