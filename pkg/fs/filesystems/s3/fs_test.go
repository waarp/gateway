package s3

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/fstest"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

const (
	accessKeyIDVar = "AWS_ACCESS_KEY_ID"
	secretKeyVar   = "AWS_SECRET_ACCESS_KEY"
	regionVar      = "AWS_DEFAULT_REGION"
	bucketVar      = "AWS_BUCKET"
)

func initTestFS(tb testing.TB) *filesystem {
	tb.Helper()

	accessKeyID := os.Getenv(accessKeyIDVar)
	secretKey := os.Getenv(secretKeyVar)
	options := map[string]any{
		"bucket": os.Getenv(bucketVar),
		"region": os.Getenv(regionVar),
	}

	s3fs, fsErr := newS3FS(accessKeyID, secretKey, options)
	require.NoError(tb, fsErr, "failed to create S3 filesystem")

	return s3fs
}

func cleanupFS(tb testing.TB) func() {
	tb.Helper()

	// A simple "DeleteAll" would also do the trick, but this is WAY faster.
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		fSys := initTestFS(tb)
		resp, listErr := fSys.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: &fSys.bucket,
			Prefix: aws.String("TestFS/"),
		})
		require.NoError(tb, listErr)

		objIDs := make([]types.ObjectIdentifier, len(resp.Contents))
		for i, obj := range resp.Contents {
			objIDs[i] = types.ObjectIdentifier{Key: obj.Key}
		}

		_, delErr := fSys.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: &fSys.bucket,
			Delete: &types.Delete{Objects: objIDs},
		})
		require.NoError(tb, delErr)
	}
}

func TestFS(t *testing.T) {
	// skipRegexp is a map containing the paths of all the filesystem tests
	// which should be skipped, either because the tests aren't working properly,
	// or because the filesystem does not support the feature being tested.
	skipTests := map[string]func(facet, cutset string) bool{
		// S3 does not allow changing metadata of existing files (including the
		// metadata), so all the methods which depend on it must be skipped.
		"TestFS/s3_FS/fs.Chmod/":   strings.HasPrefix,
		"TestFS/s3_FS/fs.Chtimes/": strings.HasPrefix,

		// The following tests are bugged because the hackpadfs.MkdirAll function
		// does not correctly ignore the error returned by Mkdir when the new
		// directory already exists. As a result, the tests fail, even though
		// the filesystem behaves correctly.
		"TestFS/s3_FS/fs.MkdirAll/all_directories_exist": strings.EqualFold,
		"TestFS/s3_FS/fs_concurrent.MkdirAll":            strings.EqualFold,

		// S3 only allows writing files sequentially. As such, the S3 files do
		// not implement the WriterAt interface. However, the hackpadfs test
		// suite does not account for that fact, and instead of skipping the
		// test (like it should in this case), it fails it. So we skip the test
		// manually.
		"TestFS/s3_File/file.WriteAt/": strings.HasPrefix,
	}

	t.Cleanup(cleanupFS(t))

	testContext := fstest.FSOptions{
		Name: "s3",
		TestFS: func(tb testing.TB) fstest.SetupFS {
			tb.Helper()

			// fstest REQUIRES that the file system tests be run in parallel.
			// To avoid having to create a new bucket for each test, we instead
			// create a directory, and then chroot into it.
			// All object (files and dirs) are then cleaned when all the tests
			// have ended (see cleanupFS function above).
			fSys := initTestFS(tb)
			require.NoError(tb, hackpadfs.MkdirAll(fSys, tb.Name(), 0))
			fSys.root = tb.Name()

			return fSys
		},
		Constraints: fstest.Constraints{
			// S3 does not have file modes, so we disable testing on file modes.
			FileModeMask: 0,
			// Because tests are run on a chrooted filesystem, paths in errors
			// returned by said filesystems will be prefixed with the root. This
			// breaks tests, so we enable this option to ignore that prefix.
			AllowErrPathPrefix: true,
		},
		ShouldSkip: func(facets fstest.Facets) bool {
			for cutset, match := range skipTests {
				if match(facets.Name, cutset) {
					return true
				}
			}

			return false
		},
	}

	fstest.FS(t, testContext)
	fstest.File(t, testContext)
}

func (f *filesystem) Chmod(string, fs.FileMode) error            { return nil }
func (f *filesystem) Chtimes(string, time.Time, time.Time) error { return nil }
