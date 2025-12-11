package backends

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/azure"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/gcs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/backends/s3"
)

//nolint:gochecknoinits //init is used by design
func init() {
	// S3
	fs.Register("s3", s3.NewFS)
	fs.Register("aws", s3.NewFS)

	// Azure
	fs.Register("azblob", azure.NewBlobFS)
	fs.Register("azureblob", azure.NewBlobFS)
	fs.Register("azfiles", azure.NewFilesFS)
	fs.Register("azurefiles", azure.NewFilesFS)

	// Google Cloud
	fs.Register("gcs", gcs.NewFS)
	fs.Register("google", gcs.NewFS)
}
