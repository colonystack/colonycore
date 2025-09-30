package s3

import (
	"colonycore/internal/blob/core"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Store implements core.Store using an S3-compatible backend (AWS S3 or MinIO).
// Minimal surface area: single bucket. Keys map to object keys directly.
type Store struct {
	client  *s3.Client
	bucket  string
	presign *s3.PresignClient
	baseURL *url.URL // optional explicit endpoint base for constructing local-style URLs
}

// Config holds explicit construction parameters (mostly for tests). For prod
// we rely primarily on environment variables.
type Config struct {
	Region          string
	Bucket          string
	Endpoint        string // optional; if set enables custom endpoint (e.g. MinIO)
	AccessKeyID     string // optional (falls back to default credentials chain)
	SecretAccessKey string // optional
	SessionToken    string // optional
	PathStyle       bool
}

// Environment variables (documented in README):
//   COLONYCORE_BLOB_DRIVER=s3
//   COLONYCORE_BLOB_S3_BUCKET=<bucket> (required)
//   COLONYCORE_BLOB_S3_REGION=<region> (default us-east-1)
//   COLONYCORE_BLOB_S3_ENDPOINT=<url> (optional, for MinIO)
//   COLONYCORE_BLOB_S3_PATH_STYLE=true|false (default false)
//   AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_SESSION_TOKEN (optional)

// New creates an S3 blob store from Config.
func New(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket required")
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	var loadOpts []func(*config.LoadOptions) error
	if region != "" {
		loadOpts = append(loadOpts, config.WithRegion(region))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.PathStyle {
			o.UsePathStyle = true
		}
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	ps := s3.NewPresignClient(client)
	var base *url.URL
	if cfg.Endpoint != "" {
		if u, err := url.Parse(cfg.Endpoint); err == nil {
			base = u
		}
	}
	return &Store{client: client, bucket: cfg.Bucket, presign: ps, baseURL: base}, nil
}

// OpenFromEnv constructs an S3 store from process environment.
func OpenFromEnv(ctx context.Context) (*Store, error) {
	bucket := os.Getenv("COLONYCORE_BLOB_S3_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("COLONYCORE_BLOB_S3_BUCKET required for s3 driver")
	}
	cfg := Config{
		Bucket:    bucket,
		Region:    os.Getenv("COLONYCORE_BLOB_S3_REGION"),
		Endpoint:  os.Getenv("COLONYCORE_BLOB_S3_ENDPOINT"),
		PathStyle: strings.EqualFold(os.Getenv("COLONYCORE_BLOB_S3_PATH_STYLE"), "true"),
	}
	return New(ctx, cfg)
}

func (s *Store) Driver() core.Driver { return core.DriverS3 }

func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts core.PutOptions) (core.Info, error) {
	input := &s3.PutObjectInput{Bucket: &s.bucket, Key: &key, Body: r}
	if opts.ContentType != "" {
		input.ContentType = &opts.ContentType
	}
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}
	// Emulate create-only via Head first.
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err == nil {
		return core.Info{}, fmt.Errorf("blob %s already exists", key)
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return core.Info{}, err
	}
	return s.Head(ctx, key)
}

func (s *Store) Get(ctx context.Context, key string) (core.Info, io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return core.Info{}, nil, err
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	info := s.fromHead(key, size, out.ContentType, out.ETag, out.Metadata, out.LastModified)
	return info, out.Body, nil
}

func (s *Store) Head(ctx context.Context, key string) (core.Info, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return core.Info{}, err
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return s.fromHead(key, size, out.ContentType, out.ETag, out.Metadata, out.LastModified), nil
}

func (s *Store) Delete(ctx context.Context, key string) (bool, error) {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return false, err
	}
	// Head to confirm existence pre-delete is extra round trip; for simplicity assume existed if no error.
	return true, nil
}

func (s *Store) List(ctx context.Context, prefix string) ([]core.Info, error) {
	var infos []core.Info
	var token *string
	for {
		out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &s.bucket, Prefix: &prefix, ContinuationToken: token})
		if err != nil {
			return nil, err
		}
		for _, obj := range out.Contents {
			var size int64
			if obj.Size != nil {
				size = *obj.Size
			}
			infos = append(infos, core.Info{Key: aws.ToString(obj.Key), Size: size, LastModified: aws.ToTime(obj.LastModified)})
		}
		if out.IsTruncated != nil && *out.IsTruncated && out.NextContinuationToken != nil {
			token = out.NextContinuationToken
			continue
		}
		break
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Key < infos[j].Key })
	return infos, nil
}

func (s *Store) PresignURL(ctx context.Context, key string, opts core.SignedURLOptions) (string, error) {
	method := strings.ToUpper(opts.Method)
	if method == "" {
		method = "GET"
	}
	if method != "GET" {
		return "", core.ErrUnsupported
	}
	expiry := opts.Expiry
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	pout, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key}, func(po *s3.PresignOptions) { po.Expires = expiry })
	if err != nil {
		return "", err
	}
	return pout.URL, nil
}

func (s *Store) fromHead(key string, size int64, contentType *string, etag *string, md map[string]string, lastModified *time.Time) core.Info {
	var ct, et string
	if contentType != nil {
		ct = *contentType
	}
	if etag != nil {
		et = strings.Trim(*etag, "\"")
	}
	lm := time.Now().UTC()
	if lastModified != nil {
		lm = *lastModified
	}
	return core.Info{Key: key, Size: size, ContentType: ct, ETag: et, Metadata: md, LastModified: lm}
}
