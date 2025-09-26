package blob

import (
	"context"
	"fmt"
	"os"
)

// Open selects a blob.Store implementation using environment variables.
//
//	COLONYCORE_BLOB_DRIVER: fs|s3|memory (default fs)
//	COLONYCORE_BLOB_FS_ROOT: directory root when driver=fs (default ./blobdata)
//	(S3 specific variables documented in s3.go)
func Open(ctx context.Context) (Store, error) {
	driver := os.Getenv("COLONYCORE_BLOB_DRIVER")
	if driver == "" {
		driver = string(DriverFilesystem)
	}
	switch Driver(driver) {
	case DriverFilesystem:
		root := os.Getenv("COLONYCORE_BLOB_FS_ROOT")
		return NewFilesystem(root)
	case DriverS3:
		return OpenFromEnv(ctx)
	case DriverMemory:
		return NewMemory(), nil
	default:
		return nil, fmt.Errorf("unknown blob driver %s", driver)
	}
}
